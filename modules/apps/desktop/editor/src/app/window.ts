/**
 * The window put together: the vault it reads, the kinds it declares, and the
 * few things one kind asks of another.
 */
import { onMounted, onUnmounted, watch } from 'vue'
import { core } from './vault'
import { presets } from '../widgets/preset-editor/core'
import { runSupport } from '../shared/command/runs'
import type { CommandTarget } from '../shared/command/target'
import type { NoteLookup } from '../shared/command/lists'
import { fileOpeners } from '../shared/tabs/openers'
import { useWindowTabs } from '../shared/tabs/windowTabs'
import { messageLog } from '../shared/notices/messages'
import { createMediaTypeProbe } from '../shared/media/player'
import { WORDS as words } from '../shared/words'
import { AGENT, FILES, PLEX, begun } from '../shared/tabs/workspace'

import { useEditing } from './editing'
import { useSettings } from './settings'
import { useVaults } from './vaults'
import { useAttention } from './attention'
import { useCommands } from './commands'
import { useWindowShowing } from './showing'
import { useWindowNotices } from './notices'
import { useWindowKinds } from './kinds'

/** Everything the window is made of, made once and handed to what draws it. */
export const useWindow = () => {
  const log = messageLog()
  const told = log.under('command')
  const held = useWindowTabs()
  const { layout } = held
  const puts = fileOpeners(core)
  const runs = runSupport()
  const plays = createMediaTypeProbe()

  const editing = useEditing({
    core,
    log,
    puts,
    held,
    day: () => settings.dayBegins.day.value,
  })

  const window = useWindowShowing(core, {
    told: async (paths, renamed) => {
      editing.notes.changed(paths, renamed)
      editing.decks.changed(paths, renamed)
      editing.stencils.changed(paths, renamed)
      editing.schedules.changed(paths, renamed)
      commandsModule.commands.follows(renamed)
      await kinds.files.refreshChangedPaths(paths, renamed)
      await kinds.plexes.again(renamed)
    },
    drawing: editing.changes.told,
    wanted: (path) => kinds.plexes.travel(path),
    reads: (path, spans) => void puts.opensAt(path, spans),
    reloads: () => vaultsModule.reloads(),
  })

  const settings = useSettings({
    core,
    words,
    log,
    held,
    onSizeChanged: () => editing.noted.measures(),
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
    const on = front && held.heldIn(tab)?.kind.over?.(front.state)
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
    (file) => void vaultsModule.carrying(file),
    { immediate: true },
  )

  const carries = (id: string, target: CommandTarget) => commandsModule.carries(id, target)

  const kinds = useWindowKinds({
    core,
    log,
    puts,
    held,
    runs,
    plays,
    editing,
    settings,
    vaults: vaultsModule,
    window,
    where,
    carries,
    doing: () => commandsModule.doing,
  })

  const knows: NoteLookup = {
    called: (path) => {
      const heldId = editing.reached.holding(path)
      return heldId === null ? kinds.plexes.names(path) : editing.titled(heldId)
    },
    holding: (path) => editing.reached.holding(path),
  }

  const attention = useAttention({ core, held })

  const opensPreset = async (path: string): Promise<void> => {
    const stands = (await core.fileKinds([path])).get(path)
    if (stands?.type !== 'deck') return editing.schedules.shows(path)
    const answer = await presets.scheduling(path)
    if (answer.error) return told(words.errors[answer.error], 'error')
    if (!answer.preset?.path) return told(words.noPreset, 'caution')
    editing.schedules.shows(answer.preset.path, answer.preset.title)
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
    reloads: vaultsModule.reloads,
    carrying: vaultsModule.carrying,
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

  const shut = (id: string, hold: () => void) => {
    if (!held.shut(id)) hold()
  }

  const starts = async () => {
    if (!vaultsModule.shown.value.id) return
    const plex = await held.opens(PLEX)
    const talk = await held.opens(AGENT)
    const tree = await held.opens(FILES)
    layout.value = begun(plex, talk, tree)
  }

  onMounted(async () => {
    globalThis.addEventListener('keydown', commandsModule.asked)
    await vaultsModule.listing()
    await settings.start()
    await starts()
    void window.start()
    void editing.going.start()
  })

  onUnmounted(() => {
    globalThis.removeEventListener('keydown', commandsModule.asked)
    window.close()
    editing.close()
    held.close()
    settings.close()
  })

  return {
    carries,
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
    tabIcon: attention.tabIcon,
    titled: editing.titled,
    where,
  }
}
