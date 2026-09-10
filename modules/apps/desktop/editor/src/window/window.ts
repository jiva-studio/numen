/**
 * The window put together: the vault it reads, the kinds it declares, and the
 * few things one kind asks of another.
 *
 * Everything here is made once, as the window opens, and the tabs are handed
 * what they hold. What a tab of a kind holds is that kind's own.
 */
import { computed, onMounted, onUnmounted, shallowRef, watch } from 'vue'
import { useConversation } from '@numen/ui'
import type { Notice } from '@numen/ui'
import { core } from './vault'
import { documents } from '../document-tab/wire'
import { books } from '../book-tab/wire'
import { recordings } from '../shared/media/wire'
import { troubleWords } from '@numen/wire'
import { running } from '../shared/artifacts'
import { cards } from '../shared/flashcards/cards'
import { presets } from '../flashcards-preset-tab/core'
import type { Source } from '../shared/core'
import { useWindowShowing } from './showing'
import { usePlexView } from '../plex-tab/view'
import { useDocumentReader } from '../document-tab/open'
import { useBookReader } from '../book-tab/open'
import { WORDS as bookWords } from '../book-tab/words'
import { cornerOf } from '../shared/notices/corner'
import { CREATABLE } from '../note-tab/maker'
import { runSupport } from '../shared/command/runs'
import type { CommandTarget } from '../shared/command/target'
import type { NoteLookup } from '../shared/command/lists'
import { invocationOf } from '../shared/command/target'
import { does } from '../shared/command/handlers'
import { lands, type DestinationDeps } from '../shared/command/destination'
import { fileOpeners } from '../shared/tabs/openers'
import { fileMakers } from '../shared/tabs/makers'
import { useWindowTabs } from '../shared/tabs/windowTabs'
import { messageLog } from '../shared/notices/messages'
import { agentKind, useAgentConversation } from '../agent-tab/kind'
import { documentKind, useDocumentTab } from '../document-tab/kind'
import { bookKind, useBookTab } from '../book-tab/kind'
import { recordingKind, type MediaTabDeps } from '../shared/media/kind'
import { RECORDINGS } from '../media-recording-tab/kind'
import { URLS } from '../media-url-tab/kind'
import { useTranscript } from '../shared/media/transcript'
import { createMediaTypeProbe } from '../shared/media/player'
import { filesKind } from '../files-tab/kind'
import { useFileTree } from '../files-tab/listing'
import { plexKind } from '../plex-tab/kind'
import { core as agent } from '../agent-tab/core'
import { WORDS as talk } from '../agent-tab/words'
import { WORDS as cardWords } from '../shared/flashcards/words'
import { WORDS as words } from '../shared/words'
import { AGENT, CONVERSATION, FILES, PLEX, begun, minted } from '../shared/tabs/workspace'

import { openEditing } from './editing'
import { useSettings } from './settings'
import { useVaults } from './vaults'
import { useAttention } from './attention'
import { useCommands } from './commands'

/** Everything the window is made of, made once and handed to what draws it. */
export const useWindow = () => {
  const log = messageLog()
  const told = log.under('command')
  const held = useWindowTabs()
  const { layout } = held
  const puts = fileOpeners(core)
  const runs = runSupport()
  const plays = createMediaTypeProbe()

  const editing = openEditing({
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
      await files.refreshChangedPaths(paths, renamed)
      await plexes.again(renamed)
    },
    drawing: editing.changes.told,
    wanted: (path) => plexes.travel(path),
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

  const notices = computed<readonly Notice[]>(() =>
    cornerOf(
      window.tasks.value,
      log.messages.value,
      {
        unwatched: window.unwatched.value,
        unread: window.trouble.value,
        lost: window.lost.value || settings.dressed.lost.value,
        reading: window.indexing.value,
        holds: window.holds.value,
      },
      vaultsModule.coverage(),
      words,
    ),
  )

  const dragged = shallowRef<readonly string[]>([])

  const plexes = plexKind(held.handle, () => usePlexView(core), {
    makes: editing.making,
    ready: computed(() => !window.failure.value),
    hangs: settings.hungParts.hangs,
    parts: settings.hungParts.parts,
    opens: (path, title, showing, line) => void puts.opens(path, title, showing, line),
    inside: (paths) => core.headings(paths),
    asks: (text) => void agents.askQuestion(text),
    runs: (id, path, title) => carries(id, { ...where(), path, title }),
    opening: window.opening,
    first: () => window.first(),
    dragged,
    says: (text) => told(text, 'refusal'),
    writes: async () => (await editing.making.createUntitled('', []))?.path ?? '',
    creatable: CREATABLE,
  })

  const agents = agentKind(
    held.handle,
    () =>
      useAgentConversation(useConversation(agent, talk, minted(CONVERSATION)), {
        opens: (path, ...runs) => void puts.opensAt(path, runs),
        beside: (path) => void puts.opens(path, '', 'beside'),
        resolve: (written) => core.resolve('', written),
        unreachable: () => window.unreachable.value,
      }),
    () => {
      const path = plexes.looking()
      return { path, title: plexes.names(path) || path }
    },
  )

  const read = documentKind(held.handle, (path) => useDocumentTab(useDocumentReader(documents, path)), puts)

  const turned = bookKind(
    held.handle,
    (path) => useBookTab(useBookReader(books, path, bookWords, log.under('book'))),
    puts,
  )

  const over = (source: Source): MediaTabDeps => ({
    runs: (id, path, called) =>
      carries(id, {
        ...where(),
        path: '',
        title: called,
        file: path,
        source,
        made: vaultsModule.makes.value.get(path) ?? {},
      }),
    canRun: (run) => runs.canRun(run),
  })

  const recorded = recordingKind(
    held.handle,
    (path) => useTranscript(recordings, path, { plays }),
    over('recording'),
    puts,
    RECORDINGS,
  )

  const pointed = recordingKind(
    held.handle,
    (path) => useTranscript(recordings, path, { plays }),
    over('url'),
    puts,
    URLS,
  )

  watch(window.tasks, () => {
    recorded.ticked(window.tasks.value)
    pointed.ticked(window.tasks.value)
  })

  const places: DestinationDeps = {
    travel: (path) => plexes.travel(path),
    opensAt: (path, run) => puts.opensAt(path, [run]),
    opens: (path, title, line) => void puts.opens(path, title, 'here', line),
  }

  const fetches = async (path: string): Promise<void> => {
    try {
      await running.fetches(path)
    } catch (error) {
      told(troubleWords(error), 'refusal')
      return
    }
    editing.notes.changed([path])
  }

  const made = fileMakers(
    {
      makeDeck: (title, folder) => cards.makeDeck(title, folder),
      makeStencil: (title, folder, fields) => cards.makeStencil(title, folder, fields),
      makePreset: (title, folder) => presets.makes(title, folder),
      makeURL: async (address, folder) => {
        const made = await core.makeURL(address, folder)
        if (made.path) void fetches(made.path)
        return made
      },
    },
    puts,
    { refused: words.refused },
    told,
  )

  const files = filesKind(held.handle, () => useFileTree(core), {
    lands: (landing) => void lands(landing, places),
    runs: (id, paths, name, source) => {
      const path = paths[0] ?? ''
      carries(id, {
        ...where(),
        path,
        title: name,
        file: path,
        source,
        made: vaultsModule.makes.value.get(path) ?? {},
        others: paths.slice(1),
      })
    },
    moves: (from, to) => does(invocationOf('move', { ...where(), path: from }, to), commandsModule.doing, words),
    drags: (paths) => {
      dragged.value = paths
    },
    makes: (path) => does(invocationOf('makeFolder', where(), path), commandsModule.doing, words),
    writes: async (folder) => (await editing.making.createUntitled(folder, []))?.path ?? '',
    decks: (folder, name) => made.makes('deck', folder, name),
    stencils: (folder, name) => made.makes('stencil', folder, name, [cardWords.newField]),
    presets: (folder, name) => made.makes('preset', folder, name),
    imports: (folder, address) => made.imports(folder, address),
    says: (text) => told(text, 'refusal'),
    canRun: (run) => runs.canRun(run),
  })

  held.declares([
    ...editing.kinds,
    ...settings.kinds,
    plexes.kind,
    agents.kind,
    read.kind,
    turned.kind,
    recorded.kind,
    pointed.kind,
    files.kind,
  ])

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

  const knows: NoteLookup = {
    called: (path) => {
      const heldId = editing.reached.holding(path)
      return heldId === null ? plexes.names(path) : editing.titled(heldId)
    },
    holding: (path) => editing.reached.holding(path),
  }

  const attention = useAttention({ core, held })

  const opensPreset = async (path: string): Promise<void> => {
    const stands = (await core.fileKinds([path])).get(path)
    if (stands?.type !== 'deck') return editing.schedules.shows(path)
    const answer = await presets.scheduling(path)
    if (answer.refusal) return told(words.refused[answer.refusal], 'refusal')
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
    made,
    shown: vaultsModule.shown,
    reloads: vaultsModule.reloads,
    carrying: vaultsModule.carrying,
    reached: editing.reached,
    opensPreset,
    dressed: settings.dressed,
    oneName: settings.oneName,
    hungParts: settings.hungParts,
    recorded,
    pointed,
    files: () => files,
    plexes: () => plexes,
    agents: () => agents,
    opening: () => window.opening.value,
    told,
  })

  const carries = commandsModule.carries

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
    places,
    shut,
    tabIcon: attention.tabIcon,
    titled: editing.titled,
    where,
  }
}
