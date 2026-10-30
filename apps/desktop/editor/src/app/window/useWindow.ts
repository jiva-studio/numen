/**
 * The window composed: the vault it reads, the kinds it declares, and the
 * few requests one kind makes of another.
 */
import { ref } from 'vue'
import { formatErrorMessage } from '@numen/plugin-sdk'
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
import { useCrossWindowTabs } from '../tabs/useCrossWindowTabs'

import { createNoteEditors } from '../tabs/useNoteEditors'
import { useSettings } from './useSettings'
import { useVaults } from './useVaults'
import { parseSavedTabs, useOpenTabs } from '../tabs/useOpenTabs'
import { useCommands } from '../commands/useCommands'
import { useDeclaredCommands } from '../commands/declaredCommands'
import { useFileMenuCommands } from '../commands/fileMenuCommands'
import { useWindowDisplay } from './useWindowDisplay'
import { useWindowNotices } from '../shell/useWindowNotices'
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
  pluginHost?: import('@/entities/plugin').PluginHost
  fileMenuCommands?: ReturnType<typeof useFileMenuCommands>
}

function initWindowCommands(deps: WindowCommandsOptions) {
  const { windowTabs, settings, vaultsModule, editing, kinds, window, writeMessage } = deps
  return useCommands({
    declaredCommands: useDeclaredCommands(),
    fileMenuCommands:
      deps.fileMenuCommands ??
      useFileMenuCommands({
        onRun: (file) => deps.pluginHost?.notifyVaultChange({ paths: [file] }),
        onError: (error) => writeMessage(formatErrorMessage(error), 'error'),
      }),
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
    reached: editing.reached,
    appearance: settings.appearance,
    oneName: settings.oneName,
    rest: settings.rest,
    travel: (path) => kinds.travel(path),
    leavePath: (from, to) => kinds.leavePath(from, to),
    getOpeningNote: () => window.opening.value,
    writeMessage,
    ...(deps.pluginHost !== undefined ? { pluginHost: deps.pluginHost } : {}),
  })
}

interface AppLifecycleDeps {
  vaultsModule: ReturnType<typeof useVaults>
  windowTabs: ReturnType<typeof useWindowTabs>
  settings: ReturnType<typeof useSettings>
  window: ReturnType<typeof useWindowDisplay>
  editing: ReturnType<typeof createNoteEditors>
  pluginLoader?: import('@numen/plugin-host').PluginLoader
  writeMessage: MessageWriter
  /** Called once the tabs the window opens with are open. */
  onLayoutStarted(): void
}

/** The tabs the core stored for the vault, and none where the core cannot be queried. */
const readSavedTabs = async () => {
  try {
    return await core.readOpenTabs()
  } catch {
    return { tabs: [], activeId: '' }
  }
}

function initAppLifecycle(deps: AppLifecycleDeps) {
  const { vaultsModule, windowTabs, settings, window, editing } = deps
  const startLayout = async () => {
    if (!vaultsModule.activeVault.value.id) return
    const search = typeof globalThis.location !== 'undefined' ? globalThis.location.search : ''
    const params = search ? new URLSearchParams(search) : null
    const initialTab = params?.get('tab')
    if (initialTab) {
      const colonIdx = initialTab.indexOf(':')
      const kind = colonIdx !== -1 ? initialTab.slice(0, colonIdx) : initialTab
      const target = colonIdx !== -1 ? initialTab.slice(colonIdx + 1) : ''
      const openedTab = await windowTabs.openTabOfKind(kind, target)
      if (openedTab) {
        windowTabs.layout.value = {
          root: { kind: 'pane', id: 'main', tabs: [openedTab], active: openedTab },
          axis: 'horizontal',
          focus: 'main',
        }
        return
      }
    }
    const saved = await readSavedTabs()
    if (saved.tabs.length > 0) {
      windowTabs.restoreTabs(parseSavedTabs(saved), saved.activeId)
    } else {
      await windowTabs.openPlacedTabs((kind, error) =>
        deps.writeMessage(`The ${kind} tab did not open: ${formatErrorMessage(error)}`, 'error'),
      )
    }
    deps.onLayoutStarted()
  }

  useAppBootstrap({
    loadVaults: () => vaultsModule.loadVaults(),
    startSettings: () => settings.start(),
    startLayout,
    // The plugins the settings name are active before the layout opens the tabs of their kinds.
    startPlugins: async () => {
      const plugins = settings.rest?.plugins?.value ?? []
      await deps.pluginLoader?.trackPlugins(getFrontendPluginIds(plugins))
      await deps.pluginLoader?.start()
    },
    startWindow: () => window.start(),
    startEditing: () =>
      runInBackground(editing.fileFlush.start(), (error) =>
        deps.writeMessage(`Saving on quit stopped: ${formatErrorMessage(error)}`, 'error'),
      ),
    closeWindow: () => window.close(),
    closeEditing: () => editing.close(),
    closeTabs: () => windowTabs.close(),
    closeSettings: () => settings.close(),
    closePlugins: async () => {
      await deps.pluginLoader?.close()
    },
    reportError: (error) =>
      deps.writeMessage(`The window did not start cleanly: ${formatErrorMessage(error)}`, 'error'),
  })
}

function initWindowCore(settingsHolder: { current?: ReturnType<typeof useSettings> }) {
  const getPluginName = (pluginId: string): string =>
    settingsHolder.current?.rest.plugins.value.find((one) => one.id === pluginId)?.name ?? pluginId
  const log = createMessageLog()
  const writeMessage = log.getWriter('command')
  const windowTabs = useWindowTabs()
  const tabLinks = useTabLinks()
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
  })
  return {
    log,
    writeMessage,
    windowTabs,
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
}

function getCommands(ctx: WindowContext): NonNullable<WindowContext['commands']> {
  if (!ctx.commands) throw new Error('window commands are read before they are created')
  return ctx.commands
}

function getKinds(ctx: WindowContext): NonNullable<WindowContext['kinds']> {
  if (!ctx.kinds) throw new Error('window kinds are read before they are created')
  return ctx.kinds
}

function getVaults(ctx: WindowContext): NonNullable<WindowContext['vaults']> {
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
  if (typeof globalThis.location === 'undefined') return false
  const params = new URLSearchParams(globalThis.location.search)
  return params.get('window') === 'secondary' || params.has('secondary')
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

  const notices = useWindowNotices({
    log: base.log,
    plugins: base.pluginNotices,
    window,
    settings,
    vaults: getVaults(ctx),
  })
  const statusBar = useWindowStatusBar({
    tasks: window.tasks,
    resolveAction: resolveTaskAction,
    onAction: (pluginId) => settings.configured.openSettings(pluginId),
  })
  const isWritingTabs = ref(false)
  const isLayoutReady = ref(false)
  const openTabs = useOpenTabs({
    core,
    windowTabs: base.windowTabs,
    isWriting: isWritingTabs,
    reportError: (error) =>
      base.writeMessage(`The open tabs were not saved: ${formatErrorMessage(error)}`, 'error'),
  })

  const closeTab = (id: string, hold?: () => void) => {
    base.tabLinks.removeClosedTab(id)
    if (!base.windowTabs.releaseTab(id)) hold?.()
  }

  useAppHotkeys(getCommands(ctx).onKeyDown)
  const crossWindowTabs = useCrossWindowTabs({
    windowTabs: base.windowTabs,
    closeWindowIfEmpty: isSecondaryWindow() ? () => window.close() : undefined,
  })
  initAppLifecycle({
    vaultsModule: getVaults(ctx),
    windowTabs: base.windowTabs,
    settings,
    window,
    editing,
    pluginLoader: base.pluginLoader,
    writeMessage: base.writeMessage,
    onLayoutStarted: () => {
      isLayoutReady.value = true
      isWritingTabs.value = !isSecondaryWindow()
    },
  })

  const api = {
    runCommand,
    openCurrentVault: () =>
      runInBackground(
        base.windowTabs.openPlacedTabs((kind, error) =>
          base.writeMessage(`The ${kind} tab did not open: ${formatErrorMessage(error)}`, 'error'),
        ),
        (error) =>
          base.writeMessage(`The tabs did not open: ${formatErrorMessage(error)}`, 'error'),
      ),
    isLayoutReady,
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
    notices,
    dismissNotice: (id: string) => {
      base.log.dismiss(id)
      base.pluginNotices.dismiss(id)
    },
    statusBar,
    palette: getCommands(ctx).palette,
    destinations: getKinds(ctx).destinations,
    closeTab,
    getTabIcon: openTabs.getTabIcon,
    getTitle: editing.getTitle,
    getTarget,
    tabLinks: base.tabLinks,
    detachTab: core.detachTab,
    dockToOtherWindow: crossWindowTabs.dockToOtherWindow,
  }
  return api
}
