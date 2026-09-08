/**
 * The window put together: the vault it reads, the kinds it declares, and the
 * few things one kind asks of another.
 *
 * Everything here is made once, as the window opens, and the tabs are handed
 * what they hold. What a tab of a kind holds is that kind's own.
 */
import { computed, onMounted, onUnmounted, ref, shallowRef, watch } from 'vue'
import { conversation } from '@numen/ui'
import type { Notice } from '@numen/ui'
import { core, vaults } from './vault'
import { documents, recordings } from './assets'
import { troubleWords } from '@numen/wire'
import { running } from './artifacts'
import { cards } from './cards/vault'
import type { Attention, ArtifactStates, Source, VaultList } from './core'
import { showing } from './showing'
import { view } from './plex/view'
import { openDocument } from './document/open'
import { cornerOf } from './notices/corner'
import type { IndexCoverage } from './notices/coverage'
import { openNotes } from './note/notes'
import { noteChanges } from './note/changes'
import { CREATABLE, noteMaker } from './note/maker'
import { runSupport } from './command/runs'
import { invocationOf, type VaultRef, type CommandTarget } from './command/target'
import type { NoteLookup, PaletteLists } from './command/lists'
import { commandPalette } from './command/palette'
import { chorded, commandFor } from './command/chords'
import { iconOfKind } from './icons'
import { themes } from './settings/theme'
import {
  APPEARANCE,
  DRESSING,
  INTERFACE_SCALE,
  MODE,
  TEXT_SCALE,
  windowAppearance,
} from './settings/appearance'
import { reviewSetting } from './settings/review'
import { OFF, ON, SYNCING, syncSetting } from './settings/sync'
import { HANGING, PARTS, hanging } from './settings/hanging'
import { reaching, type CommandDeps, type Store } from './command/deps'
import { does } from './command/handlers'
import { search } from './command/search'
import { lands, type DestinationDeps } from './command/destination'
import { fileMakers, fileOpeners } from './tabs/openers'
import { flushing } from './saving/flushing'
import { raisesConflicts } from './saving/conflicts'
import { windowing } from './tabs/windowing'
import { messageLog } from './notices/messages'
import { agentKind, talking } from './agent/kind'
import { decking } from './cards/deckTabs'
import { stencilling } from './cards/stencilTabs'
import { presets } from './preset/core'
import { presetting } from './preset/kind'
import { settingsStore } from './settings/store'
import { settling } from './settings/controls/kind'
import { editingSettingsFile } from './settings/file/kind'
import { documentKind, documenting } from './document/kind'
import { RECORDINGS, recordingKind, URLS, type MediaTabDeps } from './media/kind'
import { transcript } from './media/transcript'
import { playable } from './media/player'
import { filesKind } from './files/kind'
import { listing as folders } from './files/listing'
import { noting } from './note/kind'
import { plexKind } from './plex/kind'
import { core as agent } from './agent/core'
import { WORDS as talk } from './agent/words'
import { WORDS as cardWords } from './cards/words'
import { WORDS as words } from './words'
import { AGENT, CONVERSATION, FILES, PLEX, named, opening } from './tabs/workspace'

/** Everything the window is made of, made once and handed to what draws it. */
export const useWindow = () => {
  const changes = noteChanges()
  const notes = openNotes(core, { replaced: changes.arrived })
  /** Every message the window holds, each part of it under a name of its own. */
  const log = messageLog()
  const making = noteMaker(core, log.under('made'))
  /** The page drawn again, which is a clean window on the vault that arrived. */
  const reloads = () => globalThis.location.reload()
  const window = showing(core, {
    told: async (paths, renamed) => {
      notes.changed(paths, renamed)
      decks.changed(paths, renamed)
      stencils.changed(paths, renamed)
      schedules.changed(paths, renamed)
      commands.follows(renamed)
      await files.changed(paths, renamed)
      await plexes.again(renamed)
    },
    drawing: changes.told,
    wanted: (path) => plexes.travel(path),
    reads: (path, runs) => void puts.opensAt(path, runs),
    reloads,
  })
  /** What the window answers when the application says it is going. */
  const going = flushing(core)
  going.holds(notes.flush)

  raisesConflicts(notes, going)

  const { indexing, failure, trouble, unwatched, unreachable, holds } = window
  /** What carrying a command out leaves the person to be told. */
  const told = log.under('command')
  const { chunks, embedded, embedding, tasks } = window

  /** How far this vault has been read for meaning, as the window was last told. */
  const coverage = (): IndexCoverage => ({
    chunks: chunks.value,
    embedded: embedded.value,
    embedding: embedding.value,
  })

  /** Everything the window has to say, as the corner draws it. */
  const notices = computed<readonly Notice[]>(() =>
    cornerOf(
      tasks.value,
      log.messages.value,
      {
        unwatched: unwatched.value,
        unread: trouble.value,
        lost: window.lost.value || dressed.lost.value,
        reading: indexing.value,
        holds: holds.value,
      },
      coverage(),
      words,
    ),
  )

  /** The tabs of this window, whatever kind each of them holds. */
  const held = windowing()
  const { layout } = held

  /**
   * What a file of the vault is put in front of the person with. Every road to a
   * file comes through here, and the editors and the reader hand over their own
   * door as they are made: nothing else in the window holds one.
   */
  const puts = fileOpeners(core)

  /**
   * The runs this build cannot do at all, as this window has been told them. The
   * application answers a run it cannot do once, and this window stops offering
   * it wherever it is offered.
   */
  const runs = runSupport()

  /** The notes the window has open: what each is called, and what each tab of one holds. */
  /** Whether this window can play a kind of sound, asked of it once. */
  const plays = playable()

  const noted = noting(core, notes, changes, held.handle, puts)

  /** The decks and the stencils the window has open, each saved the way a note is. */
  const decks = decking(cards, presets, held.handle, puts)
  const stencils = stencilling(cards, held.handle, puts, log.under('stencil'))

  /** The hour a day of review begins at, on the clock on the wall. */
  const dayBegins = reviewSetting(core, words, log.under('reviewed'))

  /**
   * The presets the window has open, each written as one group of settings. The
   * day a preset counts a date from is the review day, which the tabs are told
   * and do not work out.
   */
  const schedules = presetting(
    presets,
    held.handle,
    puts,
    log.under('preset'),
    () => dayBegins.day.value,
  )

  going.holds(decks.flush)
  going.holds(stencils.flush)
  going.holds(schedules.flush)
  raisesConflicts(decks, going)
  raisesConflicts(stencils, going)

  /**
   * The notes being dragged from one pane of the window to another: the tree
   * says what it has lifted, and a plex draws a line to them. Neither knows the
   * other is there.
   */
  const dragged = shallowRef<readonly string[]>([])

  /** Whether a node hangs the parts of its note under the box, and how many. */
  const hungParts = hanging(core, words, log.under('hanging'))

  /** The plex tabs, and the one the person is looking at. */
  const plexes = plexKind(held.handle, () => view(core), {
    makes: making,
    ready: computed(() => !failure.value && !indexing.value),
    hangs: hungParts.hangs,
    parts: hungParts.parts,
    opens: (path, title, showing, line) => void puts.opens(path, title, showing, line),
    inside: (paths) => core.headings(paths),
    asks: (text) => void agents.asks(text),
    runs: (id, path, title) => carries(id, { ...where(), path, title }),
    opening: window.opening,
    first: () => window.first(),
    dragged,
    says: (text) => told(text, 'refusal'),
    writes: async () => (await making.named('', []))?.path ?? '',
    creatable: CREATABLE,
  })

  /** The agent tabs, and the one a question about a note is put in. */
  const agents = agentKind(
    held.handle,
    () =>
      talking(conversation(agent, talk, named(CONVERSATION)), {
        opens: (path, ...runs) => void puts.opensAt(path, runs),
        beside: (path) => void puts.opens(path, '', 'beside'),
        resolve: (written) => core.resolve('', written),
        unreachable: () => unreachable.value,
      }),
    () => {
      const path = plexes.looking()
      return { path, title: plexes.names(path) || path }
    },
  )

  /** The document tabs, each reading the document it is filed at. */
  const read = documentKind(held.handle, (path) => documenting(openDocument(documents, path)), puts)

  /** What a command asked for in one of these tabs is over: the file it holds. */
  const over = (source: Source): MediaTabDeps => ({
    runs: (id, path, called) =>
      carries(id, {
        ...where(),
        path: '',
        title: called,
        file: path,
        source,
        made: makes.value.get(path) ?? {},
      }),
    canRun: (run) => runs.canRun(run),
  })

  /** The recording tabs, each playing the recording it is filed at. */
  const recorded = recordingKind(
    held.handle,
    (path) => transcript(recordings, path, { plays }),
    over('recording'),
    puts,
    RECORDINGS,
  )

  /** The url tabs, each holding what was fetched from the address it points at. */
  const pointed = recordingKind(
    held.handle,
    (path) => transcript(recordings, path, { plays }),
    over('url'),
    puts,
    URLS,
  )

  // A transcript grows while a run goes, and the list of work is the only word of
  // it the window gets.
  watch(tasks, () => {
    recorded.ticked(tasks.value)
    pointed.ticked(tasks.value)
  })

  /** Where the window is taken when something is chosen, wherever it was chosen. */
  const places: DestinationDeps = {
    travel: (path) => plexes.travel(path),
    opensAt: (path, run) => puts.opensAt(path, [run]),
    opens: (path, title, line) => void puts.opens(path, title, 'here', line),
  }

  /**
   * What is at the address a link note points at, fetched, and the note read
   * again with it. A build that cannot fetch leaves the note pointing at the
   * address and nothing else.
   */
  const fetches = async (path: string): Promise<void> => {
    try {
      await running.fetches(path)
    } catch (error) {
      told(troubleWords(error), 'refusal')
      return
    }
    notes.changed([path])
  }

  /**
   * A deck, a stencil, a preset or a link made under the name it is given. A
   * preset names none of its settings, so the decks pointed at it are scheduled
   * by the defaults until the person moves one. A link is named by the address
   * it points at.
   */
  const made = fileMakers(
    {
      makeDeck: (title, folder) => cards.makeDeck(title, folder),
      makeStencil: (title, folder, fields) => cards.makeStencil(title, folder, fields),
      makesPreset: (title, folder) => presets.makes(title, folder),
      makesURL: async (address, folder) => {
        const made = await core.makeURL(address, folder)
        // What is at the address is fetched as the file is made: the person
        // pasted it to have what is there, and the file is called what the
        // address calls itself once that is known.
        if (made.path) void fetches(made.path)
        return made
      },
    },
    puts,
    { refused: words.refused, field: cardWords.newField },
    told,
  )

  /** The tree of the vault, and what a gesture on a row of it comes to. */
  const files = filesKind(held.handle, () => folders(core), {
    lands: (landing) => void lands(landing, places),
    runs: (id, paths, name, source) => {
      const path = paths[0] ?? ''
      carries(id, {
        ...where(),
        path,
        title: name,
        file: path,
        source,
        made: makes.value.get(path) ?? {},
        others: paths.slice(1),
      })
    },
    moves: (from, to) => does(invocationOf('move', { ...where(), path: from }, to), doing, words),
    drags: (paths) => {
      dragged.value = paths
    },
    makes: (path) => does(invocationOf('makeFolder', where(), path), doing, words),
    writes: async (folder) => (await making.named(folder, []))?.path ?? '',
    decks: (folder, name) => made.makes('deck', folder, name),
    stencils: (folder, name) => made.makes('stencil', folder, name),
    presets: (folder, name) => made.makes('preset', folder, name),
    imports: (folder, address) => made.imports(folder, address),
    says: (text) => told(text, 'refusal'),
    canRun: (run) => runs.canRun(run),
  })

  /** The kinds this window draws. */
  held.declares([
    noted.kind,
    plexes.kind,
    agents.kind,
    read.kind,
    recorded.kind,
    pointed.kind,
    files.kind,
    decks.kind,
    stencils.kind,
    schedules.kind,
  ])

  /** The palette: one keystroke, and everything the words typed turn up. */
  const palette = search(core, words, { coverage })

  /** The vault this window is showing, as the list of vaults has it. */
  const shown = ref<VaultRef>({ id: '', name: '' })

  /** Every vault the installation holds, as the list last answered. */
  const listed = ref<VaultList>({ vaults: [], showing: '' })

  /** What asking for the list of vaults leaves the person to be told. */
  const unlisted = log.under('listed')

  /** Which of the vaults on the list this window is showing, and what it is called. */
  const listing = async () => {
    try {
      const answer = await core.vaults()
      listed.value = answer
      const one = answer.vaults.find((vault) => vault.id === answer.showing)
      shown.value = one ? { id: one.id, name: one.name } : { id: '', name: '' }
    } catch {
      // The layout the window opens with is the one the list's answer decides.
      unlisted(words.unlistedVaults, 'refusal')
    }
  }

  /**
   * What a command is over: the tab in front, the note it means, and the file a
   * run is over. Each kind says that of one of its own tabs, and a kind that says
   * nothing is over neither.
   */
  const where = (): CommandTarget => {
    const front = held.handle.front()
    const tab = front?.id ?? ''
    const on = front && held.heldIn(tab)?.kind.at?.(front.state)
    const file = on?.file ?? ''
    return {
      tab,
      kind: front?.kind ?? null,
      path: on?.path ?? '',
      title: on?.title ?? '',
      file,
      source: on?.source ?? null,
      made: (file && makes.value.get(file)) || {},
      vault: shown.value,
      ready: !failure.value && !indexing.value,
    }
  }

  /**
   * What each file the window has asked about carries. A command over a file is
   * offered on what has been made from it, so this is asked as the file comes in
   * front and again whenever a run over it is asked for.
   */
  const makes = shallowRef<ReadonlyMap<string, ArtifactStates>>(new Map())

  /** What one file carries, asked of the application and kept. */
  const carrying = async (path: string) => {
    if (!path) return
    try {
      const held = await running.carries(path)
      makes.value = new Map(makes.value).set(path, held)
    } catch {
      // A file that cannot be asked about is one nothing is known of, and every
      // command over it is offered as it was before anything could be listed.
      makes.value = new Map(makes.value).set(path, {})
    }
  }

  // The file in front decides what is offered over it, so what it carries is
  // asked for as it arrives.
  watch(
    () => where().file,
    (file) => void carrying(file),
    { immediate: true },
  )

  /**
   * The tab the person is looking at. A question is written into an agent tab, so
   * the tab in front of one is the one they were last in beside it.
   */
  const looked = (): string => {
    const at = held.handle.front()
    if (at && at.kind !== AGENT) return at.id
    const beside = [...held.tabs.value]
      .reverse()
      .find((one) => held.heldIn(one.id)?.kind.kind !== AGENT)
    return beside?.id ?? at?.id ?? ''
  }

  /**
   * What the person has open, as whoever answers on their behalf is told it:
   * every tab, what it holds, and which of them is in front.
   *
   * A kind that says what one of its tabs holds says it here; any other says what
   * kind it is and no more.
   */
  const attends = (): Attention => ({
    front: looked(),
    tabs: held.tabs.value.map(({ id, title }) => {
      const one = held.heldIn(id)
      const said = one?.kind.attends?.(one.state)
      return {
        id,
        kind: one?.kind.kind ?? '',
        title,
        path: said?.path ?? '',
        ...(said?.document ? { document: said.document } : {}),
        ...(said?.recording ? { recording: said.recording } : {}),
      }
    }),
  })

  /** The same, told to the application as the window opens and whenever it changes. */
  const attention = computed<Attention>(() => attends())
  watch(
    attention,
    (open) => {
      void core.attending(open).catch((why) => {
        // An agent asking what is open is answered from what last arrived.
        console.error('what the window has open was not told:', why)
      })
    },
    { immediate: true },
  )

  /** What kind of tab this is drawn as, before the name it carries. */
  const tabIcon = (id: string) => iconOfKind(held.heldIn(id)?.kind.kind ?? '')

  /**
   * Every store the window keeps open files in. A command reaches a file through
   * whichever of them holds it, so a deck and a stencil answer a removal the way
   * a note does.
   */
  const stores: readonly Store[] = [noted.kept, decks.kept, stencils.kept]

  /** The open files a command reaches, whichever of the stores holds each. */
  const reached = reaching(stores, puts)

  /** What one thing the quit is waiting on is called. */
  const titled = (id: string): string => stores.find((one) => one.has(id))?.called(id) ?? ''

  /** What the window knows about a note by the name it is filed under. */
  const knows: NoteLookup = {
    called: (path) => {
      const held = reached.holding(path)
      return held === null ? plexes.names(path) : titled(held)
    },
    holding: (path) => reached.holding(path),
  }

  /** How the window is drawn: the theme it wears, its half of a pair, its sizes. */
  const dressed = windowAppearance(themes, words, log.under('worn'))

  /** Whether a note's title and the name of its file are kept as one name. */
  const oneName = syncSetting(core, words, log.under('named'))

  /** The rest of the settings file, which no command of the window turns. */
  const rest = settingsStore(core, words, log.under('configured'))

  /** The settings file itself, opened whole in a tab of its own. */
  const file = editingSettingsFile(held.handle, core, () => void rest.start())

  /**
   * Everything this installation is configured as, in a tab of its own. It holds
   * nothing: each row reaches the same value the command of that name reaches.
   */
  const configured = settling(held.handle, {
    themes: dressed.list,
    applied: dressed.applied,
    mode: dressed.mode,
    pinned: dressed.pinned,
    sizes: dressed.sized,
    bounds: dressed.bounds,
    chooses: (item) => void dressed.chooses(item),
    syncing: computed({
      get: () => oneName.kept.value,
      set: (on) => void oneName.chooses(on ? ON : OFF),
    }),
    hangs: computed({
      get: () => hungParts.hangs.value,
      set: (on) => void hungParts.chooses(on ? ON : OFF),
    }),
    parts: hungParts.parts,
    partsBounds: hungParts.ends,
    choosesParts: (count) => void hungParts.choosesCount(`${count}`),
    dayStarts: dayBegins.starts,
    latestDayStarts: dayBegins.latest,
    choosesDayStarts: (hour) => void dayBegins.chooses(hour),
    setting: (at) => rest.at(at),
    models: (at) => rest.offers(at),
    writes: (written) => void rest.chooses(written),
    file: rest.path,
    opensFile: () => file.shows(),
  })

  held.declares([configured.kind, file.kind])

  // A size is drawn, and every open editor takes its measurements again. An
  // editor watches its own box, and a size changes the type inside that box
  // while the box itself stands.
  watch(dressed.sized, () => noted.measures())

  /** The lists the window itself holds, which a step of a command offers. */
  const kept: PaletteLists = {
    offers: (command, typed) => {
      if (command === APPEARANCE) return dressed.offers()
      if (command === MODE) return dressed.modes()
      if (command === INTERFACE_SCALE || command === TEXT_SCALE)
        return dressed.sizes(command, typed)
      if (command === SYNCING) return oneName.offers()
      if (command === HANGING) return hungParts.offers()
      if (command === PARTS) return hungParts.counts()
      return []
    },
    // A theme and a size are worn where the keyboard stands, so that a person
    // sees them. A setting is written only once it is chosen.
    shows: (command, item) => {
      if (DRESSING.includes(command)) dressed.shows(item)
    },
  }

  /**
   * The preset of a note, put in front of the person. A deck is scheduled by the
   * preset its links name, and every other note is asked about as a preset itself.
   */
  const opensPreset = async (path: string): Promise<void> => {
    const stands = (await core.fileKinds([path])).get(path)
    if (stands?.type !== 'deck') return schedules.shows(path)
    const answer = await presets.scheduling(path)
    if (answer.refusal) return told(words.refused[answer.refusal], 'refusal')
    if (!answer.preset?.path) return told(words.noPreset, 'caution')
    schedules.shows(answer.preset.path, answer.preset.title)
  }

  /** The commands, over whatever is in front. */
  const commands = commandPalette(core, words, where, knows, kept, runs)

  /** What the window offers a command being carried out, one port to a job. */
  const doing: CommandDeps = {
    files: {
      makes: (title, from, seat) => making.calls(title, from, seat),
      renames: (path, title) => core.rename(path, title),
      removes: (path, destroy) => core.remove(path, destroy),
      moves: (from, to) => core.move(from, to),
      makesFolder: (path) => core.makeFolder(path),
    },
    runs: {
      carries: (path) => running.carries(path),
      makes: async (path, of) => {
        const outcome = await running.makes(path, of)
        // What the file carries has moved, and what is offered over it follows.
        void carrying(path)
        return outcome
      },
      fetches: async (path) => {
        const outcome = await running.fetches(path)
        void carrying(path)
        return outcome
      },
      corrects: async (path) => {
        const outcome = await running.corrects(path)
        void carrying(path)
        return outcome
      },
      deletesTranscript: async (path) => {
        const able = await running.deletesTranscript(path)
        if (able) {
          recorded.deleted(path)
          pointed.deleted(path)
        }
        void carrying(path)
        return able
      },
      deletesCopy: async (path) => {
        const able = await running.deletesCopy(path)
        void carrying(path)
        return able
      },
    },
    makers: made,
    vaults: {
      ...vaults,
      calls: (vault) => (shown.value = vault),
      reloads,
    },
    goes: {
      reveals: (path) => void files.reveals(path),
      travel: (path) => plexes.travel(path),
      leaves: (from, to) => plexes.leaves(from, to),
      opening: () => window.opening.value,
      opens: (kind) => void held.opens(kind),
      preset: (path) => opensPreset(path),
      closes: (tab) => held.drops(tab),
      asks: (text) => void agents.asks(text),
      searches: () => {
        commands.shows(false)
        palette.shows(true)
      },
    },
    settings: {
      appearance: (chosen) => dressed.chooses(chosen),
      syncing: (chosen) => oneName.chooses(chosen),
      hanging: (chosen) => hungParts.chooses(chosen),
      parts: (chosen) => hungParts.choosesCount(chosen),
    },
    notes: reached,
    runSupport: runs,
    copies: (path) => void navigator.clipboard?.writeText(path),
    says: told,
  }

  /**
   * A command asked for, from the palette or from a menu on a node. One that
   * needs something asks for it, and the palette stands where it asks. One that
   * is not offered over what it was asked over says why.
   */
  const carries = (id: string, at: CommandTarget) => {
    const invocation = commands.asks(id, at)
    if (invocation) return void does(invocation, doing, words)
    if (commands.open.value) return palette.shows(false)
    told(commands.refused(id, at), 'refusal')
  }

  /**
   * The keystrokes taken on the window: they belong to no pane. Each carries out
   * the command its chord names, which is the one written on that command's row.
   */
  const asked = (event: KeyboardEvent) => {
    // A pane that has answered this keystroke keeps it.
    if (event.defaultPrevented || !chorded(event)) return
    const command = commandFor(event.key.toLowerCase(), event.shiftKey)
    if (!command) return
    event.preventDefault()
    carries(command, where())
  }

  /** A tab lets go of what it held. A kind with something to finish keeps it. */
  const shut = (id: string, hold: () => void) => {
    if (!held.shut(id)) hold()
  }

  /**
   * A vault showing opens a plex holding the room and an agent over the files
   * beside it. Showing none, the window opens holding nothing at all.
   */
  const starts = async () => {
    if (!shown.value.id) return
    const plex = await held.opens(PLEX)
    const talk = await held.opens(AGENT)
    const tree = await held.opens(FILES)
    layout.value = opening(plex, talk, tree)
  }

  onMounted(async () => {
    globalThis.addEventListener('keydown', asked)
    // The layout turns on which vault the list names, and stands before anything
    // the vault says can open a tab of its own.
    await listing()
    // A tab counting in days is told the day before it opens, so nothing is
    // drawn from a day this installation does not count from.
    await dayBegins.start()
    await starts()
    void window.start()
    void going.start()
    void dressed.start()
    void oneName.start()
    void hungParts.start()
    void rest.start()
  })
  onUnmounted(() => {
    globalThis.removeEventListener('keydown', asked)
    window.close()
    changes.close()
    going.close()
    held.close()
    dressed.close()
  })

  return {
    carries,
    commands,
    doing,
    failure,
    going,
    held,
    layout,
    listed,
    log,
    notices,
    palette,
    places,
    shut,
    tabIcon,
    titled,
    where,
  }
}
