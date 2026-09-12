/**
 * The window put together: the vault it reads, the kinds it declares, and the
 * few things one kind asks of another.
 */
import { watch } from 'vue'
import { core } from './vault'
import { presets } from '@/entities/deck'
import { runSupport, type CommandTarget, type NoteLookup } from '@/features/command-palette'
import { fileOpeners } from '@/entities/tab'
import { useWindowTabs } from '@/entities/tab'
import { messageLog } from '@/shared/notices/messages'
import { createMediaTypeProbe } from '@/entities/media'
import { WORDS as words } from '@/shared/words'
import { AGENT, FILES, PLEX, createWorkspace } from '@/entities/tab'

import { useNoteEditors } from './useNoteEditors'
import { useSettings } from './useSettings'
import { useVaults } from './useVaults'
import { useOpenTabs } from './useOpenTabs'
import { useCommands } from './useCommands'
import { useWindowDisplay } from './useWindowDisplay'
import { useWindowNotices } from './useWindowNotices'
import { useWindowKinds } from './useWindowKinds'
import { useAppHotkeys } from './useAppHotkeys'
import { useAppBootstrap } from './useAppBootstrap'

/** Everything the window is made of, made once and handed to what draws it. */
export const useWindow = () => {
  const log = messageLog()
  const writeMessage = log.under('command')
  const held = useWindowTabs()
  const { layout } = held
  const tabOpeners = fileOpeners(core)
  const runs = runSupport()
  const plays = createMediaTypeProbe()

  const editing = useNoteEditors({
    core,
    log,
    tabOpeners,
    held,
    day: () => settings.dayBegins.day.value,
  })

  const window = useWindowDisplay(core, {
    onVaultChanged: async (paths, renamed) => {
      editing.notes.changed(paths, renamed)
      editing.decks.changed(paths, renamed)
      editing.stencils.changed(paths, renamed)
      editing.schedules.changed(paths, renamed)
      commandsModule.commands.applyRenames(renamed)
      await kinds.files.refreshChangedPaths(paths, renamed)
      await kinds.plexes.refresh(renamed)
    },
    reportChange: editing.changes.reportChange,
    travelTo: (path) => kinds.plexes.travel(path),
    openFileAt: (path, spans) => void tabOpeners.openFileAt(path, spans),
    reload: () => vaultsModule.reload(),
  })

  const settings = useSettings({
    core,
    words,
    log,
    held,
    onSizeChanged: () => editing.noted.measureAll(),
  })

  const vaultsModule = useVaults({
    core,
    words,
    log,
    chunks: window.chunks,
    embedded: window.embedded,
    isEmbedding: window.isEmbedding,
  })

  const notices = useWindowNotices({
    log,
    window,
    settings,
    vaults: vaultsModule,
  })

  const getTarget = (): CommandTarget => {
    const front = held.handle.front()
    const tab = front?.id ?? ''
    const on = front && held.getTab(tab)?.kind.over?.(front.state)
    const file = on?.file ?? ''
    return {
      tab,
      kind: front?.kind ?? null,
      path: on?.path ?? '',
      title: on?.title ?? '',
      file,
      source: on?.source ?? null,
      made: (file && vaultsModule.makes.value.get(file)) || {},
      vault: vaultsModule.shown.value,
      ready: !window.failure.value,
    }
  }

  // The file in front decides what is offered over it, so what it carries is
  // asked for as it arrives.
  watch(
    () => getTarget().file,
    (file) => void vaultsModule.loadArtifactStates(file),
    { immediate: true },
  )

  const runCommand = (id: string, target: CommandTarget) => commandsModule.runCommand(id, target)

  const kinds = useWindowKinds({
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
  })

  const knows: NoteLookup = {
    getTitle: (path) => {
      const heldId = editing.reached.holding(path)
      return heldId === null ? kinds.plexes.getName(path) : editing.getTitle(heldId)
    },
    holding: (path) => editing.reached.holding(path),
  }

  const openTabs = useOpenTabs({ core, held })

  const openPreset = async (path: string): Promise<void> => {
    const kind = (await core.fileKinds([path])).get(path)
    if (kind?.type !== 'deck') return editing.schedules.openPreset(path)
    const answer = await presets.scheduling(path)
    if (answer.error) return writeMessage(words.errors[answer.error], 'error')
    if (!answer.preset?.path) return writeMessage(words.noPreset, 'caution')
    editing.schedules.openPreset(answer.preset.path, answer.preset.title)
  }

  const commandsModule = useCommands({
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
    shown: vaultsModule.shown,
    reload: vaultsModule.reload,
    loadArtifactStates: vaultsModule.loadArtifactStates,
    reached: editing.reached,
    openPreset,
    dressed: settings.dressed,
    oneName: settings.oneName,
    hungParts: settings.hungParts,
    recorded: kinds.recorded,
    pointed: kinds.pointed,
    files: () => kinds.files,
    plexes: () => kinds.plexes,
    agents: () => kinds.agents,
    opening: () => window.opening.value,
    writeMessage,
  })

  const closeTab = (id: string, hold?: () => void) => {
    if (!held.releaseTab(id)) hold?.()
  }

  const startLayout = async () => {
    if (!vaultsModule.shown.value.id) return
    const plex = await held.openTabOfKind(PLEX)
    const talk = await held.openTabOfKind(AGENT)
    const tree = await held.openTabOfKind(FILES)
    layout.value = createWorkspace(plex, talk, tree)
  }

  useAppHotkeys(commandsModule.onKeyDown)

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

  return {
    runCommand,
    commands: commandsModule.commands,
    commandDeps: commandsModule.commandDeps,
    failure: window.failure,
    fileFlush: editing.fileFlush,
    held,
    layout,
    listed: vaultsModule.listed,
    log,
    notices,
    palette: commandsModule.palette,
    destinations: kinds.destinations,
    closeTab,
    tabIcon: openTabs.tabIcon,
    getTitle: editing.getTitle,
    getTarget,
  }
}
