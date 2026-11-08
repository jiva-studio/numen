/**
 * The window composed: the vault it reads, the kinds it declares, and the
 * few requests one kind makes of another.
 */
import { computed, ref, watch, type Ref } from 'vue'
import { formatErrorMessage } from '@numen/core-client'
import { createWindowHost, SLOT, type SlotHost } from '@numen/plugin-host'
import { pluginCatalog } from '@/shared/clients'
import { runInBackground } from '@numen/ui'
import { core } from '../vault'
import { createNoteLookup } from '../tabs/editorTabs'
import {
  createCommands,
  runSupport,
  type Command,
  type CommandTarget,
  type NoteLookup,
} from '@/features/command-palette'
import { fileOpeners, useWindowTabs } from '@/entities/tab'
import { createMessageLog } from '@/shared/notices/messages'
import { createNoticeQueue, getTabPaneRect, useWindowFrame } from '@numen/shell'
import { WORDS as words } from '@/shared/words'
import { useTabLinks } from '@/features/tab-linking'
import { createTabSync } from '../tabs/useTabSync'
import { createTabLinkSession, getWindowReport } from '../tabs/tabLinkSession'
import { createMovedTabOpener, createTabMover } from '../tabs/tabMove'
import { closeWindow, closeWindowWhenEmpty, focusWindow } from './windowRuntime'
import { createCommandNotes } from '../commands/commandDeps'

import { createNoteEditors } from '../tabs/useNoteEditors'
import { useSettings } from './useSettings'
import { useVaults } from './useVaults'
import type { TabsHold } from '../commands/leaveVault'
import { parseSavedTabs, useOpenTabs } from '../tabs/useOpenTabs'
import type { MovedTab } from '../ports/vault'
import { useCommands } from '../commands/useCommands'
import { createTabCommands, type TabCommands } from '../tabs/tabCommands'
import { createVersionCommands } from '../commands/versionCommands'
import { getPaletteId, useDeclaredCommands } from '../commands/declaredCommands'
import { createWindowSlots } from './windowSlots'
import { createBackendHandlers } from '../commands/backendHandlers'
import { listDeclaredItemLists, runDeclaredHandler } from '@/entities/plugin'
import { createCommandItemLists, type CommandItemLists } from '../commands/commandItemLists'
import { getNamedIcon } from '@/shared/icons'
import { useWindowDisplay } from './useWindowDisplay'
import { createEditorVaults } from './windowVaults'
import { createWindowCommands } from './windowCommands'
import { createCommandNoteReader } from '../commands/usePluginCommands'
import { useNoticeTasks } from '../shell/useNoticeTasks'
import { useWindowKinds } from './useWindowKinds'
import { createBackgroundTaskHost } from '../shell/backgroundTasks'
import { resolveTaskAction } from '../shell/taskActions'
import { useAppHotkeys } from '../commands/useAppHotkeys'
import { useAppBootstrap } from './useAppBootstrap'
import {
  createPluginSupport,
  getFrontendPluginIds,
  watchPluginSettings,
} from '../plugins/useWindowPlugins'
import { createPluginNotes } from '../plugins/pluginNotes'

import { getSettingsConfig } from './settingsConfig'
import { createDisplayOptions, createTargetResolver } from './useWindowDisplayOptions'
import type { PluginServices } from '../plugins/pluginServices'
import type { MessageWriter } from '@/shared/notices/messages'

interface WindowCommandsOptions {
  tabsHold: TabsHold
  windowTabs: ReturnType<typeof useWindowTabs>
  getTarget: () => CommandTarget
  knows: NoteLookup
  settings: ReturnType<typeof useSettings>
  runs: ReturnType<typeof runSupport>
  vaultsModule: ReturnType<typeof useVaults>
  editing: ReturnType<typeof createNoteEditors>
  kinds: ReturnType<typeof useWindowKinds>
  window: ReturnType<typeof useWindowDisplay>
  writeMessage: MessageWriter
  log: ReturnType<typeof createMessageLog>
  pluginHost: import('@numen/plugin-host').PluginHost
  declaredCommands: ReturnType<typeof useDeclaredCommands>
  slots?: SlotHost | undefined
  getAppCommands: () => readonly Command[]
  readCommandNote: ReturnType<typeof createCommandNoteReader>
}

function initWindowCommands(deps: WindowCommandsOptions) {
  const { windowTabs, settings, vaultsModule, editing, kinds, window, writeMessage } = deps
  return useCommands({
    declaredCommands: deps.declaredCommands,
    ...(deps.slots ? { slots: deps.slots } : {}),
    getAppCommands: deps.getAppCommands,
    readCommandNote: deps.readCommandNote,
    core,
    words,
    log: deps.log,
    windowTabs,
    getTarget: deps.getTarget,
    knows: deps.knows,
    paletteLists: settings.paletteLists,
    runs: deps.runs,
    coverage: vaultsModule.coverage,
    making: editing.making,
    setVaultName: vaultsModule.setVaultName,
    reload: vaultsModule.reload,
    tabsHold: deps.tabsHold,
    reached: editing.reached,
    settingPalette: settings.settingPalette,
    rest: settings.rest,
    travel: (path) => kinds.travel(path),
    leavePath: (from, to) => kinds.leavePath(from, to),
    getOpeningNote: () => window.opening.value,
    writeMessage,
    pluginHost: deps.pluginHost,
  })
}

interface AppLifecycleDeps {
  vaultsModule: ReturnType<typeof useVaults>
  windowTabs: ReturnType<typeof useWindowTabs>
  settings: ReturnType<typeof useSettings>
  window: ReturnType<typeof useWindowDisplay>
  editing: ReturnType<typeof createNoteEditors>
  pluginLoader: import('@numen/plugin-host').PluginLoader
  waitForDeclaredCommands: () => Promise<void>
  writeMessage: MessageWriter
  /** Called once the tabs the window opens with are open. */
  onLayoutStarted(): void
  tabLinkSession: { start(): Promise<void>; close(): void }
  windowId: string
  /** Opens the tab of a detach and accepts it; runs in the background. */
  openDetachedTab(moved: MovedTab): void
}

const readDetachedTab = async (windowId: string, tabId: string) => {
  try {
    return await core.readDetachedTab(windowId, tabId)
  } catch {
    return null
  }
}

const readSavedTabs = async () => {
  try {
    return await core.readOpenTabs()
  } catch {
    // Stryker disable next-line StringLiteral: empty tabs array skips restoreTabs so activeId is unused
    return { tabs: [], activeId: '' }
  }
}

function initAppLifecycle(deps: AppLifecycleDeps) {
  const { vaultsModule, windowTabs, settings, window, editing } = deps
  const startLayout = async () => {
    if (!vaultsModule.activeVault.value.id) return
    // Stryker disable next-line ConditionalExpression,EqualityOperator,StringLiteral: location is always defined in the window environment
    const search = typeof globalThis.location !== 'undefined' ? globalThis.location.search : ''
    const params = search ? new URLSearchParams(search) : null
    const detachedId = params?.get('id')
    if (detachedId) {
      // A window made for a detach shows only that tab. With no detach to read, it opens no tab
      // and closes; it does not restore the tabs of the main window.
      const detached = await readDetachedTab(deps.windowId, detachedId)
      if (detached) deps.openDetachedTab(detached)
      deps.onLayoutStarted()
      return
    }
    const saved = await readSavedTabs()
    if (saved.tabs.length > 0) {
      windowTabs.restoreTabs(parseSavedTabs(saved), saved.activeId)
    } else {
      // A new vault places its tabs once the plugins of its tabs have started or failed.
      await deps.pluginLoader.waitForTabPlugins()
      await windowTabs.openPlacedTabs((kind, error) =>
        deps.writeMessage(`The ${kind} tab did not open: ${formatErrorMessage(error)}`, 'error'),
      )
    }
    deps.onLayoutStarted()
  }

  return useAppBootstrap({
    loadVaults: () => vaultsModule.loadVaults(),
    startSettings: () => settings.start(),
    startLayout,
    // The plugins the settings name are active before the layout opens the tabs of their kinds.
    startPlugins: async () => {
      const plugins = settings.rest.plugins.value
      await deps.pluginLoader.trackPlugins(getFrontendPluginIds(plugins))
      await deps.pluginLoader.start()
    },
    // The title bar and the shortcuts read the commands the plugins declare, so the window waits for
    // their first list as it waits for the plugins of the slots of the window.
    waitForInterfacePlugins: async () => {
      await Promise.all([deps.pluginLoader.waitForInterface(), deps.waitForDeclaredCommands()])
    },
    startWindow: () => {
      runInBackground(deps.tabLinkSession.start(), (error) =>
        deps.writeMessage(`The tab links stopped: ${formatErrorMessage(error)}`, 'error'),
      )
      return window.start()
    },
    startEditing: () =>
      runInBackground(editing.fileFlush.start(), (error) =>
        deps.writeMessage(`Saving on quit stopped: ${formatErrorMessage(error)}`, 'error'),
      ),
    closeWindow: () => {
      deps.tabLinkSession.close()
      window.close()
    },
    closeEditing: () => editing.close(),
    closeTabs: () => windowTabs.close(),
    closeSettings: () => settings.close(),
    closePlugins: async () => {
      await deps.pluginLoader.close()
    },
    reportError: (error) =>
      deps.writeMessage(`The window did not start cleanly: ${formatErrorMessage(error)}`, 'error'),
  })
}

const getDeclaredCommandsDeps = (
  hostHolder: { current?: import('@numen/plugin-host').PluginHost },
  settingsHolder: { current?: ReturnType<typeof useSettings> },
) => ({
  isListed: (id: string): boolean =>
    hostHolder.current?.getPaletteCommands().some((one) => one.id === id) ?? false,
  hasBackend: (pluginId: string): boolean =>
    settingsHolder.current?.rest.plugins.value.find((one) => one.id === pluginId)?.hasBackend ??
    true,
  getConfig: () => getWindowConfig(settingsHolder.current),
})

const getWindowConfig = (settings: ReturnType<typeof useSettings> | undefined) =>
  settings
    ? getSettingsConfig([
        { id: 'app', name: '', settings: settings.rest.app.value },
        ...settings.rest.plugins.value.map((one) => ({
          id: one.id,
          name: one.name,
          settings: one.settings,
        })),
      ])
    : {}

function createPluginReports(
  log: ReturnType<typeof createMessageLog>,
  getPluginName: (pluginId: string) => string,
) {
  const write = (pluginId: string, text: string, severity: 'warning' | 'error') =>
    log.getWriter(`plugin:${pluginId}`)(text, severity)
  return {
    reportViewIssue: (pluginId: string, message: string) =>
      write(pluginId, `${getPluginName(pluginId)}: ${message}`, 'error'),
    reportCommandIssue: (pluginId: string, message: string) =>
      write(pluginId, `${getPluginName(pluginId)}: ${message}`, 'warning'),
    reportError: (pluginId: string, error: unknown) =>
      write(
        pluginId,
        words.pluginError(getPluginName(pluginId), formatErrorMessage(error)),
        'error',
      ),
    reportLoadError: (pluginId: string, error: unknown) =>
      write(
        pluginId,
        words.pluginStopped(
          getPluginName(pluginId),
          `${words.pluginLoadFailed}: ${formatErrorMessage(error)}`,
        ),
        'error',
      ),
  }
}

const createWindowHandlers = (
  declaredCommands: ReturnType<typeof useDeclaredCommands>,
  writeMessage: (text: string, severity: 'info' | 'error') => void,
  tabOpeners: ReturnType<typeof fileOpeners>,
  itemLists: CommandItemLists,
) =>
  createBackendHandlers({
    getHandlers: () => declaredCommands.handlers.value,
    list: itemLists.listItems,
    run: runDeclaredHandler,
    writeMessage,
    openFile: (path) => tabOpeners.openFile(path, '', 'current').then(() => undefined),
  })

const createDeclaredParts = (
  hostHolder: { current?: import('@numen/plugin-host').PluginHost },
  settingsHolder: { current?: ReturnType<typeof useSettings> },
) => {
  const itemLists = createCommandItemLists({
    listLists: listDeclaredItemLists,
    fileKinds: (paths) => core.fileKinds(paths),
  })
  const declaredCommands = useDeclaredCommands({
    ...getDeclaredCommandsDeps(hostHolder, settingsHolder),
    listItems: itemLists.listItems,
  })
  return { itemLists, declaredCommands }
}

const createNoteKindReader = (itemLists: CommandItemLists) => async (path: string) => {
  const kind = (await itemLists.fileKinds([path])).get(path)
  return {
    noteType: kind?.type ?? '',
    ...(kind?.properties ? { properties: kind.properties } : {}),
  }
}

function initWindowCore(settingsHolder: { current?: ReturnType<typeof useSettings> }) {
  /* Stryker disable OptionalChaining,ObjectLiteral,StringLiteral,Regex: window core wiring delegates to mocked collaborators in unit tests */
  const getPluginName = (pluginId: string): string =>
    settingsHolder.current?.rest.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId
  const log = createMessageLog()
  const writeMessage = log.getWriter('command')
  const frame = useWindowFrame()
  // The plugin loader is made after the tabs; until then every plugin counts as starting.
  const pluginStarts: { current?: (pluginId: string) => boolean } = {}
  const windowTabs = useWindowTabs({
    catalog: pluginCatalog,
    isPluginStarting: (pluginId) => pluginStarts.current?.(pluginId) ?? true,
    enablePlugin: async (pluginId) => {
      await settingsHolder.current?.rest.writeSetting([pluginId, 'enabled'], true)
    },
    openPlugin: (pluginId) =>
      globalThis.dispatchEvent(new CustomEvent('numen:open-settings', { detail: { pluginId } })),
  })
  const windowId = crypto.randomUUID()
  // The core ignores a link to a tab it has no report of, so a link or unlink
  // waits for the pending tab reports. The tab links session sets the wait.
  const tabReports = { wait: (): Promise<void> => Promise.resolve() }
  const tabLinks = useTabLinks({
    write: (change) => tabReports.wait().then(() => core.writeTabLinks(windowId, change)),
    onError: (error) =>
      writeMessage(`The tabs were not linked: ${formatErrorMessage(error)}`, 'error'),
  })
  const tabOpeners = fileOpeners({ ...core, reconcileTab: windowTabs.reconcileTab })
  const runs = runSupport()
  const pluginServices: PluginServices = {}
  const pluginNotices = createNoticeQueue()
  const pluginNotes = createPluginNotes(core)
  const hostHolder: { current?: import('@numen/plugin-host').PluginHost } = {}
  const { itemLists, declaredCommands } = createDeclaredParts(hostHolder, settingsHolder)
  const appCommands: { current?: () => readonly Command[] } = {}
  const slots = createWindowSlots({
    declaredCommands,
    hostHolder,
    getAppCommands: () => appCommands.current?.() ?? [],
    writeIssue: (pluginId, message, severity) =>
      log.getWriter(`plugin:${pluginId}`)(`${getPluginName(pluginId)}: ${message}`, severity),
  })
  const { pluginHost, pluginLoader } = createPluginSupport({
    slots: slots.hostDeps,
    getCommandDeclarations: () => declaredCommands.declarations.value,
    handlers: createWindowHandlers(declaredCommands, writeMessage, tabOpeners, itemLists),
    windowTabs,
    tabLinks,
    tabOpeners,
    settingsHolder,
    getPlatform: () => frame.platform.value,
    services: pluginServices,
    notices: pluginNotices,
    ...createPluginReports(log, getPluginName),
    ...pluginNotes,
    resolveLink: async (from, address) => {
      const rawName = address.replace(/^name:\/\//, '')
      return (await core.resolveLinks(from, `[[${rawName}]]`))[0]?.path ?? ''
    },
  })
  pluginStarts.current = pluginLoader.isPluginStarting
  hostHolder.current = pluginHost
  /* Stryker restore OptionalChaining,ObjectLiteral,StringLiteral,Regex */
  return {
    frame,
    declaredCommands,
    log,
    writeMessage,
    windowId,
    windowTabs,
    tabLinks,
    tabReports,
    tabOpeners,
    runs,
    pluginHost,
    pluginLoader,
    pluginServices,
    pluginNotices,
    pluginNotes,
    slotWindowValues: slots.windowValues,
    appCommands,
    itemLists,
  }
}

type WindowCore = ReturnType<typeof initWindowCore>

interface WindowContext {
  commands?: ReturnType<typeof useCommands>
  kinds?: ReturnType<typeof useWindowKinds>
  vaults?: ReturnType<typeof useVaults>
  tabsHold?: TabsHold
  tabCommands?: TabCommands
  versionCommands?: ReturnType<typeof createVersionCommands>
}

function getCommands(ctx: WindowContext): NonNullable<WindowContext['commands']> {
  // Stryker disable next-line ConditionalExpression: definite-assignment guard for cyclic window wiring
  if (!ctx.commands) throw new Error('window commands are read before they are created')
  return ctx.commands
}

function getKinds(ctx: WindowContext): NonNullable<WindowContext['kinds']> {
  // Stryker disable next-line ConditionalExpression: definite-assignment guard for cyclic window wiring
  if (!ctx.kinds) throw new Error('window kinds are read before they are created')
  return ctx.kinds
}

function getVaults(ctx: WindowContext): NonNullable<WindowContext['vaults']> {
  // Stryker disable next-line ConditionalExpression: definite-assignment guard for cyclic window wiring
  if (!ctx.vaults) throw new Error('window vaults are read before they are created')
  return ctx.vaults
}

function initWindowServices(
  base: WindowCore,
  ctx: WindowContext,
  settingsHolder: { current?: ReturnType<typeof useSettings> },
) {
  const editing = createNoteEditors({
    core,
    log: base.log,
    tabOpeners: base.tabOpeners,
    windowTabs: base.windowTabs,
  })

  const tabSync = createTabSync({
    windowTabs: base.windowTabs,
    tabLinks: base.tabLinks,
    sendSync: (sync) => core.syncTabPath(sync),
    onError: (error) => base.writeMessage(formatErrorMessage(error), 'error'),
  })

  const window = useWindowDisplay(
    core,
    createDisplayOptions({
      editing,
      getCommands: () => getCommands(ctx),
      getKinds: () => getKinds(ctx),
      tabOpeners: base.tabOpeners,
      windowTabs: base.windowTabs,
      getVaults: () => getVaults(ctx),
      notifyPlugins: (event) => base.pluginHost.notifyVaultChange(event),
      writeMessage: base.writeMessage,
    }),
  )

  const settings = useSettings({
    core,
    words,
    log: base.log,
    windowTabs: base.windowTabs,
    onSizeChanged: () => editing.noted.measureAll(),
    catalog: pluginCatalog,
    getSettingsViews: (pluginId) => base.pluginHost.getRegisteredSettingsViews(pluginId),
    getPlatform: () => base.frame.platform.value,
  })
  settingsHolder.current = settings

  ctx.vaults = useVaults({
    core,
    words,
    log: base.log,
    chunks: window.chunks,
    embedded: window.embedded,
    isEmbedding: window.isEmbedding,
  })

  return { editing, tabSync, window, settings }
}

type WindowServices = ReturnType<typeof initWindowServices>

function initWindowKindsAndCommands(
  base: WindowCore,
  ctx: WindowContext,
  services: WindowServices,
) {
  const { editing, settings, window, tabSync } = services
  const getTarget = createTargetResolver(base.windowTabs, () => getVaults(ctx), window)
  base.appCommands.current = () => getAppCommandList(ctx)
  base.slotWindowValues.current = () => {
    const inFront = getTarget()
    return {
      'window.isVaultOpen': inFront.vault.id !== '',
      'window.isVaultReady': inFront.isReady,
      'window.isTabOpen': inFront.tab !== '',
    }
  }

  const runCommand = (id: string, target: CommandTarget) => getCommands(ctx).runCommand(id, target)
  ctx.kinds = useWindowKinds({
    core,
    log: base.log,
    tabOpeners: base.tabOpeners,
    windowTabs: base.windowTabs,
    editing,
    pluginNotes: base.pluginNotes,
    settings,
    window,
    getTarget,
    runCommand,
    commandDeps: () => getCommands(ctx).commandDeps,
    tabLinks: base.tabLinks,
    tabSync,
    getPluginCommandIcon: (id) => {
      const name =
        base.pluginHost.getPaletteCommands().find((one) => one.id === id)?.icon ??
        base.declaredCommands.commands.value?.find((one) => getPaletteId(one) === id)?.icon
      return name ? getNamedIcon(name) : undefined
    },
  })

  Object.assign(base.pluginServices, getKinds(ctx).services, {
    taskList: window.tasks,
    backgroundTasks: createBackgroundTasks(base, window, settings),
  })

  const knows: NoteLookup = createNoteLookup(editing, base.windowTabs)

  ctx.commands = initWindowCommands({
    tabsHold: {
      pause: async () => ctx.tabsHold!.pause(),
      resume: () => ctx.tabsHold!.resume(),
    },
    windowTabs: base.windowTabs,
    getTarget,
    knows,
    settings,
    runs: base.runs,
    vaultsModule: getVaults(ctx),
    editing,
    kinds: getKinds(ctx),
    window,
    writeMessage: base.writeMessage,
    log: base.log,
    pluginHost: base.pluginHost,
    declaredCommands: base.declaredCommands,
    slots: base.pluginHost.slots,
    getAppCommands: () => getAppCommandList(ctx),
    readCommandNote: createCommandNoteReader(base.itemLists),
  })

  return {
    getTarget,
    runCommand,
    menuApi: { slots: base.pluginHost.slots, runChoice: getCommands(ctx).runChoice },
  }
}

function isSecondaryWindow(): boolean {
  // Stryker disable next-line ConditionalExpression,EqualityOperator,StringLiteral: location is always defined in the window environment
  if (typeof globalThis.location === 'undefined') return false
  const params = new URLSearchParams(globalThis.location.search)
  return params.get('window') === 'secondary' || params.has('secondary')
}

type WindowSettings = ReturnType<typeof useSettings>

function createBackgroundTasks(
  base: WindowCore,
  window: WindowServices['window'],
  settings: WindowSettings,
) {
  const noticeTasks = useNoticeTasks({
    log: base.log,
    plugins: base.pluginNotices,
    window,
    settings,
  })
  return createBackgroundTaskHost({
    tasks: computed(() => [...window.tasks.value, ...noticeTasks.value]),
    cancelTask: (id) => core.cancelTask(id),
    reportCancelError: (error) =>
      base.writeMessage(`The task was not cancelled: ${formatErrorMessage(error)}`, 'error'),
    resolveAction: resolveTaskAction,
    openSettings: (pluginId) => settings.configured.openSettings(pluginId),
  })
}

function createTabCloser(base: WindowCore) {
  return (id: string, hold?: () => void): boolean => {
    if (!base.windowTabs.releaseTab(id)) {
      hold?.()
      return false
    }
    if (!hold) base.windowTabs.finishClose?.(id)
    return true
  }
}

function openPlacedTabs(base: WindowCore) {
  return runInBackground(
    base.windowTabs.openPlacedTabs((kind, error) =>
      base.writeMessage(`The ${kind} tab did not open: ${formatErrorMessage(error)}`, 'error'),
    ),
    (error) => base.writeMessage(`The tabs did not open: ${formatErrorMessage(error)}`, 'error'),
  )
}

interface WindowTabLinksDeps {
  readonly tabSync: WindowServices['tabSync']
  readonly closeTab: (id: string) => void
  readonly isHighlighted: Ref<boolean>
}

function createWindowTabLinks(base: WindowCore, deps: WindowTabLinksDeps) {
  const isMain = !isSecondaryWindow()
  const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))
  const getReport = () => getWindowReport(base.windowTabs, isMain)
  const onError = (error: unknown) =>
    base.writeMessage(`The tab links were not written: ${formatErrorMessage(error)}`, 'error')
  const session = createTabLinkSession({
    core,
    windowId: base.windowId,
    tabLinks: base.tabLinks,
    applySync: deps.tabSync.applySync,
    applyGroupPaths: deps.tabSync.applyGroupPaths,
    openMovedTab: (moved) => runInBackground(openMovedTab(moved), onError),
    setHighlighted: (isOn) => {
      deps.isHighlighted.value = isOn
    },
    getReport,
    wait,
    onError,
  })
  const openMovedTab = createMovedTabOpener({
    windowTabs: base.windowTabs,
    closeTab: deps.closeTab,
    acceptMove: session.acceptMove,
    focusWindow: () => focusWindow(),
    wait,
    onError: (error) =>
      base.writeMessage(`The moved tab did not open: ${formatErrorMessage(error)}`, 'error'),
  })
  base.tabReports.wait = session.waitForReports
  watch(() => JSON.stringify(getReport()), session.reportTabs)
  // Hiding the page cancels the watch in the core; a page restored from the
  // back-forward cache watches again.
  globalThis.addEventListener('pagehide', session.close)
  globalThis.addEventListener('pageshow', (event) => {
    if (!event.persisted) return
    runInBackground(session.reopen(), (error) =>
      base.writeMessage(`The tab links stopped: ${formatErrorMessage(error)}`, 'error'),
    )
  })
  return {
    ...session,
    openMovedTab: (moved: MovedTab) => runInBackground(openMovedTab(moved), onError),
  }
}

function closeWhenEmpty(base: WindowCore, isLayoutReady: Ref<boolean>) {
  closeWindowWhenEmpty({
    isLayoutReady,
    tabCount: () => base.windowTabs.tabs.value.length,
    closeWindow: () => closeWindow(),
    onError: (error) =>
      base.writeMessage(`The window did not close: ${formatErrorMessage(error)}`, 'error'),
  })
}

function createWindowTabMoves(
  base: WindowCore,
  editing: WindowServices['editing'],
  closeTab: (id: string) => boolean,
) {
  const notes = createCommandNotes(editing.reached, base.windowTabs)
  const mover = createTabMover({
    core,
    windowId: base.windowId,
    windowTabs: base.windowTabs,
    notes,
    closeTab,
    writeMessage: base.writeMessage,
    getPaneRect: (id) => getTabPaneRect(base.windowTabs.layout.value, id),
  })
  /** The mover writes its own errors; a failed highlight request is ignored. */
  const ignoreError = () => undefined
  type Point = { readonly x: number; readonly y: number }
  // The core does not order requests, so each highlight waits for the previous one; an off
  // request then cannot arrive before the on request it ends.
  let lastHighlight: Promise<void> = Promise.resolve()
  return {
    windowId: base.windowId,
    isTabAsking: (id: string) => notes.asking(id),
    highlightWindow: (windowId: string, isOn: boolean) => {
      lastHighlight = lastHighlight
        .then(() => core.highlightWindow(windowId, isOn))
        .catch(ignoreError)
    },
    detachTab: (id: string, point: Point) =>
      runInBackground(mover.detachTab(id, point), ignoreError),
    detachTabInPlace: (id: string) => runInBackground(mover.detachTabInPlace(id), ignoreError),
    moveTab: (id: string, windowId: string) =>
      runInBackground(mover.moveTab(id, windowId), ignoreError),
  }
}

function createWindowTabCommands(base: WindowCore, moves: ReturnType<typeof createWindowTabMoves>) {
  const { windowTabs, tabLinks } = base
  const tabCommands = createTabCommands({
    tabLinks,
    getTabs: () => windowTabs.tabs.value,
    getKind: (id) => windowTabs.getTab(id)?.kind.kind,
    isLinkable: (id) => windowTabs.getTab(id)?.kind.navigate !== undefined,
    windowId: moves.windowId,
    isAsking: moves.isTabAsking,
    detach: moves.detachTabInPlace,
    move: moves.moveTab,
  })
  tabCommands.register()
  return tabCommands
}

function createWindowVersionCommands(base: WindowCore) {
  const versionCommands = createVersionCommands({
    listItems: base.itemLists.listItems,
    run: (command, path, itemId) =>
      runDeclaredHandler({ pluginId: '', command }, { path, values: {}, itemId }),
  })
  versionCommands.register()
  return versionCommands
}

const getAppCommandList = (ctx: WindowContext): readonly Command[] => [
  ...(ctx.tabCommands?.commands ?? []),
  ...(ctx.versionCommands?.commands ?? []),
]

const getBootstrapApi = (bootstrap: ReturnType<typeof useAppBootstrap>) => ({
  bootstrapFailure: bootstrap.failure,
  isStarted: bootstrap.isStarted,
  retryBootstrap: bootstrap.retry,
})

function createWindowFrameServices(
  base: WindowCore,
  ctx: WindowContext,
  commands: Pick<ReturnType<typeof initWindowKindsAndCommands>, 'getTarget'>,
) {
  const { frame } = base
  let tableIds: readonly string[] | undefined
  const getAppCommands = () => getAppCommandList(ctx)
  const { items, runCommandToEnd } = getCommands(ctx).pluginRuns
  const windowCommands = createWindowCommands({
    getAppIds: () => [
      ...(tableIds ??= createCommands(words).map((one) => one.id)),
      ...getAppCommands().map((one) => one.id),
    ],
    getPluginIds: () => base.pluginHost.getPaletteCommands().map((one) => one.id),
    runCommandToEnd,
    listItems: items.listCommandItems,
    runWithItem: items.runWithItem,
    getTarget: commands.getTarget,
  })
  Object.assign(base.pluginServices, {
    commands: windowCommands,
    vaults: createEditorVaults({
      vaultList: getVaults(ctx).vaultList,
      getTarget: commands.getTarget,
      commandDeps: getCommands(ctx).commandDeps,
    }),
    window: createWindowHost(frame, (error) =>
      base.writeMessage(formatErrorMessage(error), 'error'),
    ),
  })
  return { frame, slots: base.pluginHost.slots, windowCommands }
}

const getPluginViews = (pluginHost: import('@numen/plugin-host').PluginHost) => ({
  titleBarView: pluginHost.getSlotView(SLOT.titleBar),
  statusBarView: pluginHost.getSlotView(SLOT.statusBar),
})

const useWindowOpenTabs = (base: WindowCore, isWriting: Ref<boolean>) =>
  useOpenTabs({
    core,
    windowTabs: base.windowTabs,
    isWriting,
    reportError: (error) =>
      base.writeMessage(`The open tabs were not saved: ${formatErrorMessage(error)}`, 'error'),
  })

/** Everything the window is made of, made once and handed to what draws it. */
export const useWindow = () => {
  const settingsHolder: { current?: ReturnType<typeof useSettings> } = {}
  const base = initWindowCore(settingsHolder)
  const ctx: WindowContext = {}
  const services = initWindowServices(base, ctx, settingsHolder)
  const { getTarget, runCommand, menuApi } = initWindowKindsAndCommands(base, ctx, services)
  const { editing, window, settings } = services
  watchPluginSettings(settings, base.pluginLoader, (error) =>
    base.writeMessage(`Plugins were not tracked: ${formatErrorMessage(error)}`, 'error'),
  )
  const isWritingTabs = ref(false)
  const isLayoutReady = ref(false)
  const openTabs = useWindowOpenTabs(base, isWritingTabs)
  ctx.tabsHold = openTabs

  const closeTab = createTabCloser(base)
  const moves = createWindowTabMoves(base, editing, closeTab)
  ctx.tabCommands = createWindowTabCommands(base, moves)
  ctx.versionCommands = createWindowVersionCommands(base)
  const isHighlighted = ref(false)
  useAppHotkeys(getCommands(ctx).onKeyDown)
  const tabLinkSession = createWindowTabLinks(base, {
    tabSync: services.tabSync,
    closeTab,
    isHighlighted,
  })
  const bootstrap = initAppLifecycle({
    vaultsModule: getVaults(ctx),
    windowTabs: base.windowTabs,
    settings,
    window,
    editing,
    pluginLoader: base.pluginLoader,
    waitForDeclaredCommands: () => base.declaredCommands.waitForFirstList(),
    writeMessage: base.writeMessage,
    tabLinkSession,
    windowId: base.windowId,
    openDetachedTab: tabLinkSession.openMovedTab,
    onLayoutStarted: () => {
      isLayoutReady.value = true
      isWritingTabs.value = !isSecondaryWindow()
    },
  })

  if (isSecondaryWindow()) closeWhenEmpty(base, isLayoutReady)

  const api = {
    runCommand,
    ...menuApi,
    openCurrentVault: () => openPlacedTabs(base),
    readNoteKind: createNoteKindReader(base.itemLists),
    isLayoutReady,
    ...getBootstrapApi(bootstrap),
    openSettings: (pluginId?: string, filter?: string) =>
      settings.configured.openSettings(pluginId, filter),
    commands: getCommands(ctx).commands,
    pluginWays: getCommands(ctx).pluginWays,
    commandDeps: getCommands(ctx).commandDeps,
    failure: window.failure,
    fileFlush: editing.fileFlush,
    windowTabs: base.windowTabs,
    layout: base.windowTabs.layout,
    vaultList: getVaults(ctx).vaultList,
    log: base.log,
    ...getPluginViews(base.pluginHost),
    palette: getCommands(ctx).palette,
    destinations: getKinds(ctx).destinations,
    closeTab,
    getTabIcon: openTabs.getTabIcon,
    getKindIcon: openTabs.getKindIcon,
    getTitle: editing.getTitle,
    getTarget,
    tabLinks: base.tabLinks,
    ...createWindowFrameServices(base, ctx, { getTarget }),
    isHighlighted,
    ...moves,
  }
  return api
}
