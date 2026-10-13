/**
 * The window put together: the vault it reads, the kinds it declares, and the
 * few things one kind asks of another.
 */
import { watch } from 'vue'
import { core } from './vault'
import { presets } from '@/entities/deck'
import { runSupport, type CommandTarget, type NoteLookup } from '@/features/command-palette'
import { AGENT, FILES, PLEX, createWorkspace, fileOpeners, useWindowTabs } from '@/entities/tab'
import { createMessageLog } from '@/shared/notices/messages'
import { createMediaTypeProbe } from '@/entities/media'
import { WORDS as words } from '@/shared/words'
import { useTabLinks } from '@/features/tab-linking'
import { useTabSync } from './useTabSync'

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

import type { MessageWriter } from '@/shared/notices/messages'
import type { PathRename } from '@/shared/paths'
import type { Span } from '@/shared/span'
import type { VaultPort } from '@/app/ports/vault'

interface DisplayWiring {
  editing: ReturnType<typeof useNoteEditors>
  getCommands: () => ReturnType<typeof useCommands>
  getKinds: () => ReturnType<typeof useWindowKinds>
  tabOpeners: ReturnType<typeof fileOpeners>
  getVaults: () => ReturnType<typeof useVaults>
}

function createDisplayOptions(deps: DisplayWiring) {
  const { editing, getCommands, getKinds, tabOpeners, getVaults } = deps
  return {
    onVaultChanged: async (paths: readonly string[], renamed: readonly PathRename[]) => {
      editing.notes.applyPathChanges(paths, renamed)
      editing.decks.applyPathChanges(paths, renamed)
      editing.stencils.applyPathChanges(paths, renamed)
      editing.schedules.applyPathChanges(paths, renamed)
      getCommands().commands.applyRenames(renamed)
      await getKinds().files.refreshChangedPaths(paths, renamed)
      await getKinds().plexes.refresh(renamed)
    },
    reportChange: editing.changes.reportChange,
    travelTo: (path: string) => getKinds().plexes.travel(path),
    openFileAt: (path: string, spans: readonly Span[]) => void tabOpeners.openFileAt(path, spans),
    reload: () => getVaults().reload(),
  }
}

function createTargetResolver(
  held: ReturnType<typeof useWindowTabs>,
  getVaults: () => ReturnType<typeof useVaults>,
  window: ReturnType<typeof useWindowDisplay>,
) {
  const getFront = () => {
    const front = held.handle.front()
    const tab = front?.id ?? ''
    const on = front && held.getTab(tab)?.kind.getTarget?.(front.state)
    return { tab, kind: front?.kind ?? null, on }
  }
  const getMade = (file: string) => (file && getVaults().makes.value.get(file)) || {}

  return (): CommandTarget => {
    const { tab, kind, on } = getFront()
    const file = on?.file ?? ''
    return {
      tab,
      kind,
      path: on?.path ?? '',
      title: on?.title ?? '',
      file,
      source: on?.source ?? null,
      made: getMade(file),
      vault: getVaults().shown.value,
      isReady: !window.failure.value,
    }
  }
}

interface DeckPresetHandlers {
  openPreset: (path: string, title?: string) => Promise<void> | void
  writeMessage: MessageWriter
}

async function resolveDeckPreset(
  vaultCore: VaultPort,
  path: string,
  handlers: DeckPresetHandlers,
): Promise<void> {
  const kind = (await vaultCore.fileKinds([path])).get(path)
  if (kind?.type !== 'deck') return handlers.openPreset(path)
  const answer = await presets.getDeckPreset(path)
  if (!answer.ok) return handlers.writeMessage(words.errors[answer.error], 'error')
  const read = answer.value.preset
  if (!read?.path) return handlers.writeMessage(words.noPreset, 'caution')
  handlers.openPreset(read.path, read.title)
}

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
}

function initWindowCommands(deps: WindowCommandsOptions) {
  const {
    held,
    getTarget,
    knows,
    settings,
    runs,
    vaultsModule,
    editing,
    kinds,
    window,
    writeMessage,
    log,
  } = deps
  return useCommands({
    core,
    words,
    log,
    held,
    getTarget,
    knows,
    kept: settings.kept,
    runs,
    coverage: vaultsModule.coverage,
    making: editing.making,
    made: kinds.made,
    setVaultName: vaultsModule.setVaultName,
    reload: vaultsModule.reload,
    loadArtifactStates: vaultsModule.loadArtifactStates,
    reached: editing.reached,
    openPreset: (path) =>
      resolveDeckPreset(core, path, {
        openPreset: (p, t) => editing.schedules.openPreset(p, t),
        writeMessage,
      }),
    dressed: settings.dressed,
    oneName: settings.oneName,
    nesting: settings.nesting,
    rest: settings.rest,
    recorded: kinds.recorded,
    pointed: kinds.pointed,
    files: () => kinds.files,
    plexes: () => kinds.plexes,
    agents: () => kinds.agents,
    getOpeningNote: () => window.opening.value,
    writeMessage,
  })
}

interface AppLifecycleDeps {
  vaultsModule: ReturnType<typeof useVaults>
  held: ReturnType<typeof useWindowTabs>
  settings: ReturnType<typeof useSettings>
  window: ReturnType<typeof useWindowDisplay>
  editing: ReturnType<typeof useNoteEditors>
}

function initAppLifecycle(deps: AppLifecycleDeps) {
  const { vaultsModule, held, settings, window, editing } = deps
  const startLayout = async () => {
    if (!vaultsModule.shown.value.id) return
    const plex = await held.openTabOfKind(PLEX)
    const talk = await held.openTabOfKind(AGENT)
    const tree = await held.openTabOfKind(FILES)
    held.layout.value = createWorkspace(plex, talk, tree)
  }

  useAppBootstrap({
    loadVaults: () => vaultsModule.loadVaults(),
    startSettings: () => settings.start(),
    startLayout,
    startWindow: () => window.start(),
    startEditing: () => void editing.fileFlush.start(),
    closeWindow: () => window.close(),
    closeEditing: () => editing.close(),
    closeTabs: () => held.close(),
    closeSettings: () => settings.close(),
  })
}

/** Everything the window is made of, made once and handed to what draws it. */
export const useWindow = () => {
  const log = createMessageLog()
  const writeMessage = log.getWriter('command')
  const held = useWindowTabs()
  const tabLinks = useTabLinks()
  const tabOpeners = fileOpeners({ ...core, reconcileTab: held.reconcileTab })
  const runs = runSupport()
  const plays = createMediaTypeProbe()

  let commandsModule: ReturnType<typeof useCommands>
  let kinds: ReturnType<typeof useWindowKinds>
  let vaultsModule: ReturnType<typeof useVaults>

  const editing = useNoteEditors({
    core,
    log,
    tabOpeners,
    held,
    day: () => settings.dayBegins.day.value,
  })

  const tabSync = useTabSync({ held, tabLinks, editing })

  const window = useWindowDisplay(
    core,
    createDisplayOptions({
      editing,
      getCommands: () => commandsModule,
      getKinds: () => kinds,
      tabOpeners,
      getVaults: () => vaultsModule,
    }),
  )

  const settings = useSettings({
    core,
    words,
    log,
    windowTabs: held,
    onSizeChanged: () => editing.noted.measureAll(),
  })

  vaultsModule = useVaults({
    core,
    words,
    log,
    chunks: window.chunks,
    embedded: window.embedded,
    isEmbedding: window.isEmbedding,
  })

  const notices = useWindowNotices({ log, window, settings, vaults: vaultsModule })
  const statusBar = useWindowStatusBar({ tasks: window.tasks })
  const getTarget = createTargetResolver(held, () => vaultsModule, window)

  watch(
    () => getTarget().file,
    (file) => void vaultsModule.loadArtifactStates(file),
    { immediate: true },
  )

  const runCommand = (id: string, target: CommandTarget) => commandsModule.runCommand(id, target)

  kinds = useWindowKinds({
    core,
    log,
    tabOpeners,
    held,
    runs,
    plays,
    editing,
    settings,
    vaults: vaultsModule,
    window,
    getTarget,
    runCommand,
    commandDeps: () => commandsModule.commandDeps,
    tabLinks,
    tabSync,
  })

  const knows: NoteLookup = {
    getTitle: (path) => {
      const heldId = editing.reached.getTabAt(path)
      return heldId === null ? kinds.plexes.getName(path) : editing.getTitle(heldId)
    },
    getTabAt: (path) => editing.reached.getTabAt(path),
  }

  const openTabs = useOpenTabs({ core, windowTabs: held })

  commandsModule = initWindowCommands({
    held,
    getTarget,
    knows,
    settings,
    runs,
    vaultsModule,
    editing,
    kinds,
    window,
    writeMessage,
    log,
  })

  const closeTab = (id: string, hold?: () => void) => {
    tabLinks.removeClosedTab(id)
    if (!held.releaseTab(id)) hold?.()
  }

  useAppHotkeys(commandsModule.onKeyDown)
  initAppLifecycle({ vaultsModule, held, settings, window, editing })

  return {
    runCommand,
    commands: commandsModule.commands,
    commandDeps: commandsModule.commandDeps,
    failure: window.failure,
    fileFlush: editing.fileFlush,
    held,
    layout: held.layout,
    listed: vaultsModule.listed,
    log,
    notices,
    statusBar,
    palette: commandsModule.palette,
    destinations: kinds.destinations,
    closeTab,
    getTabIcon: openTabs.getTabIcon,
    getTitle: editing.getTitle,
    getTarget,
    tabLinks,
  }
}
