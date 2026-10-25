/**
 * The window put together: the vault it reads, the kinds it declares, and the
 * few things one kind asks of another.
 */
import { watch } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import { core } from '../vault'
import { openCommandTab } from '../commands/commandTabs'
import { createNoteLookup } from '../tabs/editorTabs'
import { runSupport, type CommandTarget, type NoteLookup } from '@/features/command-palette'
import { fileOpeners, useWindowTabs } from '@/entities/tab'
import { createMessageLog } from '@/shared/notices/messages'
import { WORDS as words } from '@/shared/words'
import { useTabLinks } from '@/features/tab-linking'
import { createTabSync } from '../tabs/useTabSync'
import { useCrossWindowTabs } from '../tabs/useCrossWindowTabs'

import { createNoteEditors } from '../tabs/useNoteEditors'
import { useSettings } from './useSettings'
import { useVaults } from './useVaults'
import { useOpenTabs } from '../tabs/useOpenTabs'
import { useCommands } from '../commands/useCommands'
import { useDeclaredCommands } from '../commands/declaredCommands'
import { useWindowDisplay } from './useWindowDisplay'
import { useWindowNotices } from '../shell/useWindowNotices'
import { useWindowKinds } from './useWindowKinds'
import { useWindowStatusBar } from '../shell/useWindowStatusBar'
import { resolveTaskAction, resolveTaskPlugin } from '../shell/taskActions'
import { useAppHotkeys } from '../commands/useAppHotkeys'
import { useAppBootstrap } from './useAppBootstrap'
import { createPluginSupport, watchPluginSettings } from '../plugins/useWindowPlugins'
import { createPluginNotes } from '../plugins/pluginNotes'

import { createDisplayOptions, createTargetResolver } from './useWindowDisplayOptions'
import type { PluginServices } from '../plugins/pluginServices'
import type { MessageWriter } from '@/shared/notices/messages'

interface WindowKindsWithCommands extends ReturnType<typeof useWindowKinds> {
  recorded?: { reloadTranscript?: (path: string) => void; onDelete?: (path: string) => void }
  pointed?: { reloadTranscript?: (path: string) => void; onDelete?: (path: string) => void }
}

interface WindowCommandsOptions {
  windowTabs: ReturnType<typeof useWindowTabs>
  getTarget: () => CommandTarget
  knows: NoteLookup
  settings: ReturnType<typeof useSettings>
  runs: ReturnType<typeof runSupport>
  vaultsModule: ReturnType<typeof useVaults>
  editing: ReturnType<typeof createNoteEditors>
  kinds: WindowKindsWithCommands
  window: ReturnType<typeof useWindowDisplay>
  writeMessage: MessageWriter
  log: ReturnType<typeof createMessageLog>
  pluginHost?: import('@/entities/plugin').PluginHost
}

function initWindowCommands(deps: WindowCommandsOptions) {
  const { windowTabs, settings, vaultsModule, editing, kinds, window, writeMessage } = deps
  return useCommands({
    declaredCommands: useDeclaredCommands(),
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
    made: kinds.made,
    setVaultName: vaultsModule.setVaultName,
    reload: vaultsModule.reload,
    loadArtifactStates: vaultsModule.loadArtifactStates,
    reached: editing.reached,
    openPreset: (path) =>
      openCommandTab({ vaultCore: core, windowTabs, writeMessage }, path, 'preset'),
    appearance: settings.appearance,
    oneName: settings.oneName,
    nesting: settings.nesting,
    rest: settings.rest,
    recorded: kinds.recorded ?? {},
    pointed: kinds.pointed ?? {},
    files: () => kinds.files,
    travel: (path) => kinds.travel(path),
    leavePath: (from, to) => kinds.leavePath(from, to),
    agents: () => kinds.agents,
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
  pluginLoader?: import('@/features/plugin-loader').PluginLoader
  writeMessage: MessageWriter
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
    await windowTabs.openPlacedTabs((kind, error) =>
      deps.writeMessage(`The ${kind} tab did not open: ${formatErrorMessage(error)}`, 'error'),
    )
  }

  useAppBootstrap({
    loadVaults: () => vaultsModule.loadVaults(),
    startSettings: () => settings.start(),
    startLayout,
    // The plugins the settings name are active before the layout opens the tabs of their kinds.
    startPlugins: async () => {
      const plugins = settings.rest?.plugins?.value ?? []
      await deps.pluginLoader?.trackPlugins(plugins.map((plugin) => plugin.id))
      await deps.pluginLoader?.start()
    },
    startWindow: () => window.start(),
    startEditing: () => void editing.fileFlush.start(),
    closeWindow: () => window.close(),
    closeEditing: () => editing.close(),
    closeTabs: () => windowTabs.close(),
    closeSettings: () => settings.close(),
    closePlugins: async () => {
      await deps.pluginLoader?.close()
    },
  })
}

function initWindowCore(settingsHolder: { current?: ReturnType<typeof useSettings> }) {
  const log = createMessageLog()
  const writeMessage = log.getWriter('command')
  const windowTabs = useWindowTabs()
  const tabLinks = useTabLinks()
  const tabOpeners = fileOpeners({ ...core, reconcileTab: windowTabs.reconcileTab })
  const runs = runSupport()
  const pluginServices: PluginServices = {}
  const { pluginHost, pluginLoader } = createPluginSupport({
    windowTabs,
    tabLinks,
    tabOpeners,
    settingsHolder,
    services: pluginServices,
    reportError: (pluginId, error) =>
      log.getWriter(`plugin:${pluginId}`)(
        `Plugin ${pluginId} failed: ${error instanceof Error ? error.message : String(error)}`,
        'error',
      ),
    ...createPluginNotes(core),
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
  }
}

type WindowCore = ReturnType<typeof initWindowCore>

interface WindowContext {
  commands?: ReturnType<typeof useCommands>
  kinds?: WindowKindsWithCommands
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
    }),
  )

  const settings = useSettings({
    core,
    words,
    log: base.log,
    windowTabs: base.windowTabs,
    onSizeChanged: () => editing.noted.measureAll(),
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

  watch(
    () => getTarget().file,
    (file) => void getVaults(ctx).loadArtifactStates(file),
    { immediate: true },
  )

  const runCommand = (id: string, target: CommandTarget) => getCommands(ctx).runCommand(id, target)

  ctx.kinds = useWindowKinds({
    core,
    log: base.log,
    tabOpeners: base.tabOpeners,
    windowTabs: base.windowTabs,
    runs: base.runs,
    editing,
    settings,
    vaults: getVaults(ctx),
    window,
    getTarget,
    runCommand,
    commandDeps: () => getCommands(ctx).commandDeps,
    tabLinks: base.tabLinks,
    tabSync,
    getContextMenuItems: (target) => base.pluginHost.getContextMenuItems(target),
    runContextMenu: (id, target) => base.pluginHost.runContextMenu(id, target),
  })

  Object.assign(base.pluginServices, getKinds(ctx).services)

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
  watchPluginSettings(settings, base.pluginLoader)

  const notices = useWindowNotices({ log: base.log, window, settings, vaults: getVaults(ctx) })
  const statusBar = useWindowStatusBar({
    tasks: window.tasks,
    resolveAction: (task) => resolveTaskAction(task, settings.rest.plugins?.value ?? []),
    onAction: (id) => {
      const task = window.tasks.value.find((t) => t.id === id)
      const pluginId = task
        ? resolveTaskPlugin(task, settings.rest.plugins?.value ?? [])
        : undefined
      settings.configured.openSettings(pluginId)
    },
  })
  const openTabs = useOpenTabs({ core, windowTabs: base.windowTabs })

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
  })

  const api = {
    runCommand,
    commands: getCommands(ctx).commands,
    commandDeps: getCommands(ctx).commandDeps,
    failure: window.failure,
    fileFlush: editing.fileFlush,
    windowTabs: base.windowTabs,
    layout: base.windowTabs.layout,
    vaultList: getVaults(ctx).vaultList,
    log: base.log,
    notices,
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
