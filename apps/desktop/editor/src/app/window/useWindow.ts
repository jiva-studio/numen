/**
 * The window composed: the vault it reads, the kinds it declares, and the
 * few requests one kind makes of another.
 */
import { computed, ref, watch, type Ref } from 'vue'
import { waitFor } from './waitFor'
import { formatErrorMessage } from '@numen/core-client'
import type { Menus } from '@numen/plugin-host'
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
import { createNoticeQueue } from '@numen/shell'
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
import { useDeclaredCommands } from '../commands/declaredCommands'
import { createPluginMenus } from '../commands/pluginMenus'
import { getNamedIcon } from '@/shared/icons'
import { useFileMenuCommands } from '../commands/fileMenuCommands'
import { useWindowDisplay } from './useWindowDisplay'
import { useNoticeTasks } from '../shell/useNoticeTasks'
import { useWindowKinds } from './useWindowKinds'
import { useWindowStatusBar } from '../shell/useWindowStatusBar'
import { resolveTaskAction } from '../shell/taskActions'
import { useAppHotkeys } from '../commands/useAppHotkeys'
import { useAppBootstrap } from './useAppBootstrap'
import {
  createPluginSupport,
  getFrontendPluginIds,
  watchPluginSettings,
} from '../plugins/useWindowPlugins'
import { createPluginNotes } from '../plugins/pluginNotes'

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
  fileMenuCommands: ReturnType<typeof useFileMenuCommands>
  declaredCommands: ReturnType<typeof useDeclaredCommands>
  menus: Menus
}

function initWindowCommands(deps: WindowCommandsOptions) {
  const { windowTabs, settings, vaultsModule, editing, kinds, window, writeMessage } = deps
  return useCommands({
    declaredCommands: deps.declaredCommands,
    fileMenuCommands: deps.fileMenuCommands,
    menus: deps.menus,
    core,
    words,
    log: deps.log,
    windowTabs,
    getTarget: deps.getTarget,
    knows: deps.knows,
    kept: settings.paletteLists,
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
  arePluginsStarting: Ref<boolean>
  writeMessage: MessageWriter
  /** Called once the tabs the window opens with are open. */
  onLayoutStarted(): void
  tabLinkSession: { start(): Promise<void>; close(): void }
  windowId: string
  /** Opens the tab of a detach and accepts it; runs in the background. */
  openDetachedTab(moved: MovedTab): void
}

/** How long a vault with no saved tabs waits for the plugins that place tabs. */
const NEW_VAULT_PLUGINS_WAIT_MS = 5_000

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
  let pluginsStarted: Promise<void> = Promise.resolve()
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
      // A new vault places the tabs of the kinds registered, so it gives the plugins still starting a while longer.
      await waitFor(pluginsStarted, NEW_VAULT_PLUGINS_WAIT_MS)
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
    startPlugins: () => {
      pluginsStarted = (async () => {
        try {
          const plugins = settings.rest.plugins.value
          await deps.pluginLoader.trackPlugins(getFrontendPluginIds(plugins))
          await deps.pluginLoader.start()
        } finally {
          deps.arePluginsStarting.value = false
        }
      })()
      return pluginsStarted
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
})

function initWindowCore(settingsHolder: { current?: ReturnType<typeof useSettings> }) {
  /* Stryker disable OptionalChaining,ObjectLiteral,StringLiteral,Regex: window core wiring delegates to mocked collaborators in unit tests */
  const getPluginName = (pluginId: string): string =>
    settingsHolder.current?.rest.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId
  const log = createMessageLog()
  const writeMessage = log.getWriter('command')
  const arePluginsStarting = ref(true)
  const windowTabs = useWindowTabs({
    catalog: pluginCatalog,
    arePluginsStarting,
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
  const declaredCommands = useDeclaredCommands({
    ...getDeclaredCommandsDeps(hostHolder, settingsHolder),
  })
  const { pluginHost, pluginLoader } = createPluginSupport({
    getCommandDeclarations: () => declaredCommands.declarations.value,
    reportCommandIssue: (pluginId, message) =>
      log.getWriter(`plugin:${pluginId}`)(`${getPluginName(pluginId)}: ${message}`, 'warning'),
    windowTabs,
    tabLinks,
    tabOpeners,
    settingsHolder,
    services: pluginServices,
    notices: pluginNotices,
    reportError: (pluginId, error) =>
      log.getWriter(`plugin:${pluginId}`)(
        words.pluginError(getPluginName(pluginId), formatErrorMessage(error)),
        'error',
      ),
    reportLoadError: (pluginId, error) => {
      log.getWriter(`plugin:${pluginId}`)(
        words.pluginStopped(
          getPluginName(pluginId),
          `${words.pluginLoadFailed}: ${formatErrorMessage(error)}`,
        ),
        'error',
      )
    },
    ...pluginNotes,
    resolveLink: async (from, address) => {
      const rawName = address.replace(/^name:\/\//, '')
      return (await core.resolveLinks(from, `[[${rawName}]]`))[0]?.path ?? ''
    },
  })
  hostHolder.current = pluginHost
  /* Stryker restore OptionalChaining,ObjectLiteral,StringLiteral,Regex */
  return {
    declaredCommands,
    log,
    writeMessage,
    windowId,
    windowTabs,
    arePluginsStarting,
    tabLinks,
    tabReports,
    tabOpeners,
    runs,
    pluginHost,
    pluginLoader,
    pluginServices,
    pluginNotices,
    pluginNotes,
  }
}

type WindowCore = ReturnType<typeof initWindowCore>

interface WindowContext {
  commands?: ReturnType<typeof useCommands>
  kinds?: ReturnType<typeof useWindowKinds>
  vaults?: ReturnType<typeof useVaults>
  tabsHold?: TabsHold
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

  const runCommand = (id: string, target: CommandTarget) => getCommands(ctx).runCommand(id, target)
  let editorCommands: readonly Command[] | undefined
  const menus = createPluginMenus({
    getPlacements: () => base.declaredCommands.placements.value,
    getPluginCommands: () => base.pluginHost.getPaletteCommands(),
    getDeclaredCommands: () => base.declaredCommands.commands.value,
    getEditorCommands: () => (editorCommands ??= createCommands(words)),
    reportIssue: (pluginId, message) => {
      const name = settings.rest.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId
      base.log.getWriter(`plugin:${pluginId}`)(`${name}: ${message}`, 'warning')
    },
  })
  const fileMenuCommands = useFileMenuCommands({
    onRun: (file) => base.pluginHost.notifyVaultChange({ paths: [file] }),
    onError: (error) => base.writeMessage(formatErrorMessage(error), 'error'),
  })

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
    getContextMenuItems: (target) => base.pluginHost.getContextMenuItems(target),
    runContextMenu: (id, target) => base.pluginHost.runContextMenu(id, target),
    getFileMenuItems: (path) => fileMenuCommands.getForFile(path),
    getContextMenuIcon: (id) => base.pluginHost.getContextMenuIcon(id),
    getPluginCommandIcon: (id) => {
      const name = base.pluginHost.getPaletteCommands().find((one) => one.id === id)?.icon
      return name ? getNamedIcon(name) : fileMenuCommands.getIcon(id)
    },
    getNoteCommands: () => getCommands(ctx).getNoteCommands(),
  })

  Object.assign(base.pluginServices, getKinds(ctx).services, { taskList: window.tasks })

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
    menus,
    fileMenuCommands,
  })

  return { getTarget, runCommand, menuApi: { menus, runChoice: getCommands(ctx).runChoice } }
}

function isSecondaryWindow(): boolean {
  // Stryker disable next-line ConditionalExpression,EqualityOperator,StringLiteral: location is always defined in the window environment
  if (typeof globalThis.location === 'undefined') return false
  const params = new URLSearchParams(globalThis.location.search)
  return params.get('window') === 'secondary' || params.has('secondary')
}

type WindowSettings = ReturnType<typeof useSettings>

function createStatusBar(
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
  return useWindowStatusBar({
    tasks: computed(() => [...window.tasks.value, ...noticeTasks.value]),
    cancelTask: (id) => core.cancelTask(id),
    reportCancelError: (error) =>
      base.writeMessage(`The task was not cancelled: ${formatErrorMessage(error)}`, 'error'),
    resolveAction: resolveTaskAction,
    onAction: (pluginId) => settings.configured.openSettings(pluginId),
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
    detachTabAt: (id: string, point: Point) =>
      runInBackground(mover.detachTabAt(id, point), ignoreError),
    moveTab: (id: string, windowId: string) =>
      runInBackground(mover.moveTab(id, windowId), ignoreError),
  }
}

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
  const openTabs = useOpenTabs({
    core,
    windowTabs: base.windowTabs,
    isWriting: isWritingTabs,
    reportError: (error) =>
      base.writeMessage(`The open tabs were not saved: ${formatErrorMessage(error)}`, 'error'),
  })
  ctx.tabsHold = openTabs

  const closeTab = createTabCloser(base)
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
    arePluginsStarting: base.arePluginsStarting,
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
    windowButtons: settings.windowButtons,
    runCommand,
    ...menuApi,
    openCurrentVault: () => openPlacedTabs(base),
    isLayoutReady,
    bootstrapFailure: bootstrap.failure,
    retryBootstrap: bootstrap.retry,
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
    statusBar: createStatusBar(base, window, settings),
    statusViews: base.pluginHost.statusViews,
    palette: getCommands(ctx).palette,
    destinations: getKinds(ctx).destinations,
    closeTab,
    getTabIcon: openTabs.getTabIcon,
    getKindIcon: openTabs.getKindIcon,
    getTitle: editing.getTitle,
    getTarget,
    tabLinks: base.tabLinks,
    isHighlighted,
    ...createWindowTabMoves(base, editing, closeTab),
  }
  return api
}
