/**
 * The window composed: the vault it reads, the kinds it declares, and the
 * few requests one kind makes of another.
 */
import { computed, ref, watch, type Ref } from 'vue'
import { waitFor } from './waitFor'
import { formatErrorMessage } from '@numen/core-client'
import { pluginCatalog } from '@/shared/clients'
import { runInBackground } from '@numen/ui'
import { core } from '../vault'
import { createNoteLookup } from '../tabs/editorTabs'
import { runSupport, type CommandTarget, type NoteLookup } from '@/features/command-palette'
import { fileOpeners, useWindowTabs } from '@/entities/tab'
import { createMessageLog } from '@/shared/notices/messages'
import { createNoticeQueue } from '@numen/shell'
import { WORDS as words } from '@/shared/words'
import { useTabLinks } from '@/features/tab-linking'
import { createTabSync } from '../tabs/useTabSync'
import { createTabLinkSession, getLinkableTabs } from '../tabs/tabLinkSession'

import { createNoteEditors } from '../tabs/useNoteEditors'
import { useSettings } from './useSettings'
import { useVaults } from './useVaults'
import type { TabsHold } from '../commands/leaveVault'
import { parseSavedTabs, useOpenTabs } from '../tabs/useOpenTabs'
import { useCommands } from '../commands/useCommands'
import { useDeclaredCommands } from '../commands/declaredCommands'
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
}

function initWindowCommands(deps: WindowCommandsOptions) {
  const { windowTabs, settings, vaultsModule, editing, kinds, window, writeMessage } = deps
  return useCommands({
    declaredCommands: useDeclaredCommands(),
    fileMenuCommands: deps.fileMenuCommands,
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
    appearance: settings.appearance,
    oneName: settings.oneName,
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
  /** True until every plugin the window starts has started or failed. */
  arePluginsStarting: Ref<boolean>
  writeMessage: MessageWriter
  /** Called once the tabs the window opens with are open. */
  onLayoutStarted(): void
  /** The window's part in the tab links of every window, read once the layout is open. */
  tabLinkSession: { start(): Promise<void>; close(): void }
}

/** How long a vault with no saved tabs waits for the plugins that place tabs. */
const NEW_VAULT_PLUGINS_WAIT_MS = 5_000

/** The tabs the core stored for the vault, and none where the core cannot be queried. */
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
    const kind = params?.get('kind')
    if (kind) {
      const openedTab = await windowTabs.openTabOfKind(
        kind,
        params?.get('path') ?? '',
        params?.get('id') ?? '',
      )
      if (openedTab) {
        windowTabs.layout.value = {
          root: { kind: 'pane', id: 'main', tabs: [openedTab], active: openedTab },
          axis: 'horizontal',
          focus: 'main',
        }
        deps.onLayoutStarted()
        return
      }
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
      await settingsHolder.current?.rest.writeSetting(['plugins', pluginId, 'enabled'], true)
    },
    openPlugin: (pluginId) =>
      globalThis.dispatchEvent(new CustomEvent('numen:open-settings', { detail: { pluginId } })),
  })
  const windowId = crypto.randomUUID()
  const tabLinks = useTabLinks({
    write: (change) => core.writeTabLinks(windowId, change),
    onError: (error) =>
      writeMessage(`The tabs were not linked: ${formatErrorMessage(error)}`, 'error'),
  })
  const tabOpeners = fileOpeners({ ...core, reconcileTab: windowTabs.reconcileTab })
  const runs = runSupport()
  const pluginServices: PluginServices = {}
  const pluginNotices = createNoticeQueue()
  const pluginNotes = createPluginNotes(core)
  const { pluginHost, pluginLoader } = createPluginSupport({
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
  /* Stryker restore OptionalChaining,ObjectLiteral,StringLiteral,Regex */
  return {
    log,
    writeMessage,
    windowId,
    windowTabs,
    arePluginsStarting,
    tabLinks,
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
  /** Holds the open tabs back while the vault is switched; bound once the tabs are written. */
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
    getPluginCommandIcon: (id) =>
      base.pluginHost.getPaletteCommands().find((one) => one.id === id)?.icon ??
      fileMenuCommands.getIcon(id),
    showNote: (path) => getCommands(ctx).showNote(path),
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
    fileMenuCommands,
  })

  return { getTarget, runCommand }
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

type Point = { readonly x: number; readonly y: number }

/** Opening a tab in a new window and closing it in this one. */
function createTabDetacher(base: WindowCore, closeTab: (id: string) => void) {
  /** Detaches a tab to a new window at a point on the screen. */
  const detachTab = (id: string, point: Point): void => {
    const tab = base.windowTabs.getTab(id)
    if (!tab) return
    const path = tab.kind.getOpenTab?.(tab.state)?.path ?? ''
    core
      .detachTab({ id, kind: tab.kind.kind, path }, point)
      .then(() => closeTab(id))
      .catch((error: unknown) =>
        base.writeMessage(
          `The tab did not open in a new window: ${formatErrorMessage(error)}`,
          'error',
        ),
      )
  }
  /** Detaches a tab at a point relative to this window, as a context menu reports it. */
  const detachTabAt = (id: string, point: Point): void =>
    detachTab(id, { x: globalThis.screenX + point.x, y: globalThis.screenY + point.y })
  return { detachTab, detachTabAt }
}

/** The closing of a tab: the hold runs where the tab stays open. The core drops the links of a tab the window no longer reports. */
function createTabCloser(base: WindowCore) {
  return (id: string, hold?: () => void) => {
    if (!base.windowTabs.releaseTab(id)) {
      hold?.()
      return
    }
    if (!hold) base.windowTabs.finishClose?.(id)
  }
}

/** The tabs the vault placed, opened in the background; what fails is reported. */
function openPlacedTabs(base: WindowCore) {
  return runInBackground(
    base.windowTabs.openPlacedTabs((kind, error) =>
      base.writeMessage(`The ${kind} tab did not open: ${formatErrorMessage(error)}`, 'error'),
    ),
    (error) => base.writeMessage(`The tabs did not open: ${formatErrorMessage(error)}`, 'error'),
  )
}

/** The window's tab links session, told the linkable tabs again whenever they change. */
function createWindowTabLinks(base: WindowCore, tabSync: WindowServices['tabSync']) {
  const getOpenTabs = () => getLinkableTabs(base.windowTabs)
  const session = createTabLinkSession({
    core,
    windowId: base.windowId,
    tabLinks: base.tabLinks,
    applySync: tabSync.applySync,
    getOpenTabs,
    wait: (ms) => new Promise((resolve) => setTimeout(resolve, ms)),
    onError: (error) =>
      base.writeMessage(`The tab links were not written: ${formatErrorMessage(error)}`, 'error'),
  })
  watch(() => JSON.stringify(getOpenTabs()), session.reportTabs)
  return session
}

/** Everything the window is made of, made once and handed to what draws it. */
export const useWindow = () => {
  const settingsHolder: { current?: ReturnType<typeof useSettings> } = {}
  const base = initWindowCore(settingsHolder)
  const ctx: WindowContext = {}
  const services = initWindowServices(base, ctx, settingsHolder)
  const { getTarget, runCommand } = initWindowKindsAndCommands(base, ctx, services)
  const { editing, window, settings } = services
  watchPluginSettings(settings, base.pluginLoader, (error) =>
    base.writeMessage(`Plugins were not tracked: ${formatErrorMessage(error)}`, 'error'),
  )

  const statusBar = createStatusBar(base, window, settings)
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

  useAppHotkeys(getCommands(ctx).onKeyDown)
  const tabLinkSession = createWindowTabLinks(base, services.tabSync)
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
    onLayoutStarted: () => {
      isLayoutReady.value = true
      isWritingTabs.value = !isSecondaryWindow()
    },
  })

  const api = {
    runCommand,
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
    statusBar,
    statusViews: base.pluginHost.statusViews,
    palette: getCommands(ctx).palette,
    destinations: getKinds(ctx).destinations,
    closeTab,
    getTabIcon: openTabs.getTabIcon,
    getKindIcon: openTabs.getKindIcon,
    getTitle: editing.getTitle,
    getTarget,
    tabLinks: base.tabLinks,
    ...createTabDetacher(base, closeTab),
  }
  return api
}
