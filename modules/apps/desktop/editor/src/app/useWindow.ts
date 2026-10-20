/**
 * The window put together: the vault it reads, the kinds it declares, and the
 * few things one kind asks of another.
 */
import { watch } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import { core } from './vault'
import { resolveDeckPreset } from './deckPreset'
import { runSupport, type CommandTarget, type NoteLookup } from '@/features/command-palette'
import { fileOpeners, useWindowTabs } from '@/entities/tab'
import { createMessageLog } from '@/shared/notices/messages'
import { createMediaTypeProbe } from '@/entities/media'
import { WORDS as words } from '@/shared/words'
import { useTabLinks } from '@/features/tab-linking'
import { useTabSync } from './useTabSync'
import { useCrossWindowTabs } from './useCrossWindowTabs'

import { useNoteEditors } from './useNoteEditors'
import { useSettings } from './useSettings'
import { useVaults } from './useVaults'
import { useOpenTabs } from './useOpenTabs'
import { useCommands } from './useCommands'
import { useWindowDisplay } from './useWindowDisplay'
import { useWindowNotices } from './useWindowNotices'
import { useWindowKinds } from './useWindowKinds'
import { useWindowStatusBar } from './useWindowStatusBar'
import { useAppHotkeys } from './useAppHotkeys'
import { useAppBootstrap } from './useAppBootstrap'
import { createPluginSupport, watchPluginSettings } from './useWindowPlugins'
import { createPluginNotes } from './pluginNotes'

import { createDisplayOptions, createTargetResolver } from './useWindowDisplayOptions'
import type { PluginServices } from './pluginServices'
import type { MessageWriter } from '@/shared/notices/messages'

interface WindowCommandsOptions {
  held: ReturnType<typeof useWindowTabs>
  getTarget: () => CommandTarget
  knows: NoteLookup
  settings: ReturnType<typeof useSettings>
  runs: ReturnType<typeof runSupport>
  vaultsModule: ReturnType<typeof useVaults>
  editing: ReturnType<typeof useNoteEditors>
  kinds: ReturnType<typeof useWindowKinds>
  window: ReturnType<typeof useWindowDisplay>
  writeMessage: MessageWriter
  log: ReturnType<typeof createMessageLog>
  pluginHost?: import('@/entities/plugin').PluginHost
}

function initWindowCommands(deps: WindowCommandsOptions) {
  const { held, settings, vaultsModule, editing, kinds, window, writeMessage } = deps
  const rec = kinds as unknown as {
    recorded?: { reloadTranscript?: (path: string) => void; onDelete?: (path: string) => void }
    pointed?: { reloadTranscript?: (path: string) => void; onDelete?: (path: string) => void }
  }
  return useCommands({
    core,
    words,
    log: deps.log,
    held,
    getTarget: deps.getTarget,
    knows: deps.knows,
    kept: settings.kept,
    runs: deps.runs,
    coverage: vaultsModule.coverage,
    making: editing.making,
    made: kinds.made,
    setVaultName: vaultsModule.setVaultName,
    reload: vaultsModule.reload,
    loadArtifactStates: vaultsModule.loadArtifactStates,
    reached: editing.reached,
    openPreset: (path) =>
      resolveDeckPreset(core, path, {
        openPreset: (presetPath) => void held.openTabOfKind('preset', presetPath),
        writeMessage,
      }),
    dressed: settings.dressed,
    oneName: settings.oneName,
    nesting: settings.nesting,
    rest: settings.rest,
    recorded: rec.recorded ?? {},
    pointed: rec.pointed ?? {},
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
  held: ReturnType<typeof useWindowTabs>
  settings: ReturnType<typeof useSettings>
  window: ReturnType<typeof useWindowDisplay>
  editing: ReturnType<typeof useNoteEditors>
  pluginLoader?: import('@/features/plugin-loader').PluginLoader
  writeMessage: MessageWriter
}

function initAppLifecycle(deps: AppLifecycleDeps) {
  const { vaultsModule, held, settings, window, editing } = deps
  const startLayout = async () => {
    if (!vaultsModule.shown.value.id) return
    const search = typeof globalThis.location !== 'undefined' ? globalThis.location.search : ''
    const params = search ? new URLSearchParams(search) : null
    const initialTab = params?.get('tab')
    if (initialTab) {
      const colonIdx = initialTab.indexOf(':')
      const kind = colonIdx !== -1 ? initialTab.slice(0, colonIdx) : initialTab
      const at = colonIdx !== -1 ? initialTab.slice(colonIdx + 1) : ''
      const opened = await held.openTabOfKind(kind, at)
      if (opened) {
        held.layout.value = {
          root: { kind: 'pane', id: 'main', tabs: [opened], active: opened },
          axis: 'horizontal',
          focus: 'main',
        }
        return
      }
    }
    await held.openPlacedTabs((kind, error) =>
      deps.writeMessage(`The ${kind} tab did not open: ${formatErrorMessage(error)}`, 'error'),
    )
  }

  useAppBootstrap({
    loadVaults: () => vaultsModule.loadVaults(),
    startSettings: () => settings.start(),
    startLayout,
    startPlugins: async () => {
      await deps.pluginLoader?.start()
    },
    startWindow: () => window.start(),
    startEditing: () => void editing.fileFlush.start(),
    closeWindow: () => window.close(),
    closeEditing: () => editing.close(),
    closeTabs: () => held.close(),
    closeSettings: () => settings.close(),
    closePlugins: async () => {
      await deps.pluginLoader?.close()
    },
  })
}

function initWindowCore(settingsHolder: { current?: ReturnType<typeof useSettings> }) {
  const log = createMessageLog()
  const writeMessage = log.getWriter('command')
  const held = useWindowTabs()
  const tabLinks = useTabLinks()
  const tabOpeners = fileOpeners({ ...core, reconcileTab: held.reconcileTab })
  const runs = runSupport()
  const plays = createMediaTypeProbe()
  const pluginServices: PluginServices = {}
  const { pluginHost, pluginLoader } = createPluginSupport({
    held,
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
    held,
    tabLinks,
    tabOpeners,
    runs,
    plays,
    pluginHost,
    pluginLoader,
    pluginServices,
  }
}

type WindowCore = ReturnType<typeof initWindowCore>

interface WindowContext {
  commands: ReturnType<typeof useCommands>
  kinds: ReturnType<typeof useWindowKinds>
  vaults: ReturnType<typeof useVaults>
}

function initWindowServices(
  base: WindowCore,
  ctx: WindowContext,
  settingsHolder: { current?: ReturnType<typeof useSettings> },
) {
  const editing = useNoteEditors({
    core,
    log: base.log,
    tabOpeners: base.tabOpeners,
    held: base.held,
    day: () => settingsHolder.current!.dayBegins.day.value,
  })

  const tabSync = useTabSync({
    held: base.held,
    tabLinks: base.tabLinks,
    onError: (error) => base.writeMessage(formatErrorMessage(error), 'error'),
  })

  const window = useWindowDisplay(
    core,
    createDisplayOptions({
      editing,
      getCommands: () => ctx.commands,
      getKinds: () => ctx.kinds,
      tabOpeners: base.tabOpeners,
      getVaults: () => ctx.vaults,
      notifyPlugins: (event) => base.pluginHost.notifyVaultChange(event),
    }),
  )

  const settings = useSettings({
    core,
    words,
    log: base.log,
    windowTabs: base.held,
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
  const getTarget = createTargetResolver(base.held, () => ctx.vaults, window)

  watch(
    () => getTarget().file,
    (file) => void ctx.vaults.loadArtifactStates(file),
    { immediate: true },
  )

  const runCommand = (id: string, target: CommandTarget) => ctx.commands.runCommand(id, target)

  ctx.kinds = useWindowKinds({
    core,
    log: base.log,
    tabOpeners: base.tabOpeners,
    held: base.held,
    runs: base.runs,
    plays: base.plays,
    editing,
    settings,
    vaults: ctx.vaults,
    window,
    getTarget,
    runCommand,
    commandDeps: () => ctx.commands.commandDeps,
    tabLinks: base.tabLinks,
    tabSync,
  })

  Object.assign(base.pluginServices, ctx.kinds.services)

  const knows: NoteLookup = {
    getTitle: (path) => {
      const heldId = editing.reached.getTabAt(path)
      return heldId === null ? '' : editing.getTitle(heldId)
    },
    getTabAt: (path) => editing.reached.getTabAt(path),
  }

  ctx.commands = initWindowCommands({
    held: base.held,
    getTarget,
    knows,
    settings,
    runs: base.runs,
    vaultsModule: ctx.vaults,
    editing,
    kinds: ctx.kinds,
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
  const ctx = {} as WindowContext
  const services = initWindowServices(base, ctx, settingsHolder)
  const { getTarget, runCommand } = initWindowKindsAndCommands(base, ctx, services)
  const { editing, window, settings } = services
  watchPluginSettings(settings, base.pluginLoader)

  const notices = useWindowNotices({ log: base.log, window, settings, vaults: ctx.vaults })
  const statusBar = useWindowStatusBar({ tasks: window.tasks })
  const openTabs = useOpenTabs({ core, windowTabs: base.held })

  const closeTab = (id: string, hold?: () => void) => {
    base.tabLinks.removeClosedTab(id)
    if (!base.held.releaseTab(id)) hold?.()
  }

  useAppHotkeys(ctx.commands.onKeyDown)
  const crossWindowTabs = useCrossWindowTabs({
    held: base.held,
    closeWindowIfEmpty: isSecondaryWindow() ? () => window.close() : undefined,
  })
  initAppLifecycle({
    vaultsModule: ctx.vaults,
    held: base.held,
    settings,
    window,
    editing,
    pluginLoader: base.pluginLoader,
    writeMessage: base.writeMessage,
  })

  return {
    runCommand,
    commands: ctx.commands.commands,
    commandDeps: ctx.commands.commandDeps,
    failure: window.failure,
    fileFlush: editing.fileFlush,
    held: base.held,
    layout: base.held.layout,
    listed: ctx.vaults.listed,
    log: base.log,
    notices,
    statusBar,
    palette: ctx.commands.palette,
    destinations: ctx.kinds.destinations,
    closeTab,
    getTabIcon: openTabs.getTabIcon,
    getTitle: editing.getTitle,
    getTarget,
    tabLinks: base.tabLinks,
    detachTab: core.detachTab,
    dockToOtherWindow: crossWindowTabs.dockToOtherWindow,
  }
}
