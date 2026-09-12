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
  const told = log.under('command')
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
    told: async (paths, renamed) => {
      editing.notes.changed(paths, renamed)
      editing.decks.changed(paths, renamed)
      editing.stencils.changed(paths, renamed)
      editing.schedules.changed(paths, renamed)
      commandsModule.commands.applyRenames(renamed)
      await kinds.files.refreshChangedPaths(paths, renamed)
      await kinds.plexes.again(renamed)
    },
    drawing: editing.changes.reportChange,
    wanted: (path) => kinds.plexes.travel(path),
    reads: (path, spans) => void tabOpeners.opensAt(path, spans),
    reloads: () => vaultsModule.reload(),
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
    embedding: window.embedding,
  })

  const notices = useWindowNotices({
    log,
    window,
    settings,
    vaults: vaultsModule,
  })

  const where = (): CommandTarget => {
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
    () => where().file,
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
    where,
    runCommand,
    doing: () => commandsModule.doing,
  })

  const knows: NoteLookup = {
    called: (path) => {
      const heldId = editing.reached.holding(path)
      return heldId === null ? kinds.plexes.getName(path) : editing.getTitle(heldId)
    },
    holding: (path) => editing.reached.holding(path),
  }

  const openTabs = useOpenTabs({ core, held })

  const opensPreset = async (path: string): Promise<void> => {
    const kind = (await core.fileKinds([path])).get(path)
    if (kind?.type !== 'deck') return editing.schedules.openPreset(path)
    const answer = await presets.scheduling(path)
    if (answer.error) return told(words.errors[answer.error], 'error')
    if (!answer.preset?.path) return told(words.noPreset, 'caution')
    editing.schedules.openPreset(answer.preset.path, answer.preset.title)
  }

  const commandsModule = useCommands({
    core,
    words,
    log,
    held,
    where,
    knows,
    kept: settings.kept,
    runs,
    coverage: vaultsModule.coverage,
    making: editing.making,
    made: kinds.made,
    shown: vaultsModule.shown,
    reloads: vaultsModule.reload,
    loadArtifactStates: vaultsModule.loadArtifactStates,
    reached: editing.reached,
    opensPreset,
    dressed: settings.dressed,
    oneName: settings.oneName,
    hungParts: settings.hungParts,
    recorded: kinds.recorded,
    pointed: kinds.pointed,
    files: () => kinds.files,
    plexes: () => kinds.plexes,
    agents: () => kinds.agents,
    opening: () => window.opening.value,
    told,
  })

  const shut = (id: string, hold?: () => void) => {
    if (!held.shut(id)) hold?.()
  }

  const startLayout = async () => {
    if (!vaultsModule.shown.value.id) return
    const plex = await held.opens(PLEX)
    const talk = await held.opens(AGENT)
    const tree = await held.opens(FILES)
    layout.value = createWorkspace(plex, talk, tree)
  }

  useAppHotkeys(commandsModule.asked)

  useAppBootstrap({
    loadVaults: () => vaultsModule.loadVaults(),
    startSettings: () => settings.start(),
    startLayout,
    startWindow: () => window.start(),
    startEditing: () => void editing.going.start(),
    closeWindow: () => window.close(),
    closeEditing: () => editing.close(),
    closeTabs: () => held.close(),
    closeSettings: () => settings.close(),
  })

  return {
    runCommand,
    commands: commandsModule.commands,
    doing: commandsModule.doing,
    failure: window.failure,
    going: editing.going,
    held,
    layout,
    listed: vaultsModule.listed,
    log,
    notices,
    palette: commandsModule.palette,
    places: kinds.places,
    shut,
    tabIcon: openTabs.tabIcon,
    getTitle: editing.getTitle,
    where,
  }
}
