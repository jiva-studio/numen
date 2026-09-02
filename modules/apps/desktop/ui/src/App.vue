<script setup lang="ts">
/**
 * The window: one vault, and tabs to divide the screen between.
 *
 * What a tab of each kind holds is that kind's own, in `plex/`, `agent/`,
 * `document/`, `note/` and `files/`; keeping tabs of any kind at all is in
 * `windowing.ts`; when to ask the vault again is in `showing.ts`. What is left
 * here is the vault this window reads, the kinds it draws, and the few things
 * one kind asks of another.
 */
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { closeTab, conversation, Notices, Palette, Workspace } from '@numen/ui'
import type { Notice } from '@numen/ui'
import '@numen/ui/styles.css'
import { cards, core, documents, recordings, running, vaults } from './vault'
import type { Attention, Listed } from './core'
import { showing } from './showing'
import { standing } from './plex/standing'
import { reading } from './document/reading'
import { cornerOf } from './corner'
import type { Meaning } from './meaning'
import { editing } from './note/editing'
import { drawn } from './note/drawn'
import { CREATABLE, creating } from './note/creating'
import {
  asksCommands,
  commanding,
  creates,
  deedOf,
  MAKING,
  offering,
  type Holds,
  type Knows,
  type Shown,
  type Where,
} from './commanding'
import { chorded, commandFor, keysOf } from './keying'
import { iconFor, iconOfKind } from './icons'
import { themes } from './theme'
import { APPEARANCE, DRESSING, INTERFACE_SCALE, MODE, TEXT_SCALE, wearing } from './wearing'
import { reviewing } from './reviewing'
import { OFF, ON, SYNCING, syncing } from './syncing'
import { HANGING, PARTS, hanging } from './hanging'
import { does, reaching, type Doing, type Store } from './doing'
import { finding } from './finding'
import { lands, type Places } from './landing'
import { putting } from './putting'
import { leaving } from './leaving'
import { raising } from './raising'
import { windowing } from './windowing'
import Leaving from './Leaving.vue'
import Failure from './Failure.vue'
import { Welcome, opensVault } from '@numen/ui'
import { COMMANDS, vaultsOn, waysIn } from './welcome/welcoming'
import { telling } from './telling'
import { agentKind, talking } from './agent/kind'
import { decking } from './cards/deck'
import { stencilling } from './cards/stencil'
import { presets } from './preset/core'
import { presetting } from './preset/kind'
import { configuring } from './settings/configuring'
import { settling } from './settings/kind'
import { documentKind, documenting, type Held as DocumentHeld } from './document/kind'
import { recordingKind } from './recording/kind'
import { listening, type Listening as RecordingHeld } from './recording/listening'
import { filesKind } from './files/kind'
import { listing as folders } from './files/listing'
import { noting, type Held as NoteHeld } from './note/kind'
import { plexKind, plexing, type Held as PlexHeld } from './plex/kind'
import { core as agent } from './agent/core'
import { WORDS as talk } from './agent/words'
import { WORDS as cut } from './cards/words'
import { WORDS as words } from './words'
import { VERSION } from './version'
import {
  AGENT,
  CONVERSATION,
  DOCUMENT,
  FILES,
  NOTE,
  PLEX,
  RECORDING,
  SETTINGS,
  named,
  opening,
} from './workspace'

const drawings = drawn()
const notes = editing(core, undefined, drawings.arrived)
/** Everything the window has said, each part of it under a name of its own. */
const tell = telling()
const making = creating(core, tell.under('made'))
/** The page drawn again, which is a clean window on the vault that arrived. */
const reloads = () => globalThis.location.reload()
const window = showing(
  core,
  undefined,
  async (paths, renamed) => {
    notes.changed(paths, renamed)
    decks.changed(paths, renamed)
    stencils.changed(paths, renamed)
    schedules.changed(paths, renamed)
    commands.follows(renamed)
    await files.changed(paths, renamed)
    await plexes.again(renamed)
  },
  drawings.told,
  (path) => plexes.travel(path),
  (path, runs) => void puts.opensAt(path, runs),
  reloads,
)
/** What the window answers when the application says it is going. */
const going = leaving(core)
going.holds(notes.flush)

raising(notes, going)

const { indexing, failure, trouble, unwatched, unreachable, holds } = window
/** What carrying a command out leaves the person to be told. */
const told = tell.under('command')
const { chunks, embedded, embedding, tasks } = window

/** How far this vault has been read for meaning, as the window last heard. */
const meaning = (): Meaning => ({
  chunks: chunks.value,
  embedded: embedded.value,
  embedding: embedding.value,
})

/** Everything the window has to say, as the corner draws it. */
const notices = computed<readonly Notice[]>(() =>
  cornerOf(
    tasks.value,
    tell.said.value,
    {
      unwatched: unwatched.value,
      unread: trouble.value,
      lost: window.lost.value || dressed.lost.value,
      reading: indexing.value,
      holds: holds.value,
    },
    meaning(),
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
const puts = putting(core)

/** The notes the window has open: what each is called, and what each tab of one holds. */
const noted = noting(core, notes, drawings, held.host, puts)

/** The decks and the stencils the window has open, each saved the way a note is. */
const decks = decking(cards, presets, held.host, puts)
const stencils = stencilling(cards, held.host, puts, tell.under('stencil'))

/** The presets the window has open, each written as one group of settings. */
const schedules = presetting(presets, held.host, puts, tell.under('preset'))

going.holds(decks.flush)
going.holds(stencils.flush)
going.holds(schedules.flush)
raising(decks, going)
raising(stencils, going)

/**
 * The notes being carried from one pane of the window to another: the tree
 * says what it has lifted, and a plex draws a line to them. Neither knows the
 * other is there.
 */
const carried = ref<readonly string[]>([])

/** Whether a node hangs the parts of its note under the box, and how many. */
const hungParts = hanging(core, words, tell.under('hanging'))

/** The plex tabs, and the one the person is looking at. */
const plexes = plexKind(held.host, () => standing(core), {
  makes: making,
  ready: () => !failure.value && !indexing.value,
  hangs: () => hungParts.hangs.value,
  parts: () => hungParts.parts.value,
  opens: (path, title, showing, line) => void puts.opens(path, title, showing, line),
  inside: (paths) => core.headings(paths),
  asks: (text) => void agents.asks(text),
  runs: (id, path, title) => carries(id, { ...where(), path, title }),
  opening: () => window.opening.value,
  first: () => window.first(),
  carried: () => carried.value,
  says: (text) => told(text, 'refusal'),
  writes: async () => (await making.named('', []))?.path ?? '',
  creatable: CREATABLE,
})

/** The agent tabs, and the one a question about a note is put in. */
const agents = agentKind(held.host, () =>
  talking(conversation(agent, talk, named(CONVERSATION)), {
    opens: (path, ...runs) => void puts.opensAt(path, runs),
    beside: (path) => void puts.opens(path, '', 'beside'),
    resolve: (written) => core.resolve('', written),
    unreachable: () => unreachable.value,
  }),
)

/** The document tabs, each reading the document it is filed at. */
const read = documentKind(held.host, (path) => documenting(reading(documents, path)), puts)

/** The recording tabs, each playing the recording it is filed at. */
const heard = recordingKind(
  held.host,
  (path) => listening(recordings, path),
  {
    runs: (id, path, called) =>
      carries(id, { ...where(), path: '', title: called, file: path, source: 'recording' }),
  },
  puts,
)

// A transcript grows while a run goes, and the list of work is the only word of
// it the window gets.
watch(tasks, () => heard.ticked(tasks.value))

/** Where the window is taken when something is chosen, wherever it was chosen. */
const places: Places = {
  travel: (path) => plexes.travel(path),
  opensAt: (path, run) => puts.opensAt(path, [run]),
  opens: (path, title, line) => void puts.opens(path, title, 'here', line),
}

/**
 * A deck or a stencil made in a folder under the name it is given. The vault
 * names the file after it and answers where it stands, and a stencil is made
 * carrying the field its cards are named by. A vault that answers nothing at
 * all is said here, because the roads that ask for one carry no word of their
 * own.
 */
const makesCards = async (folder: string, name: string, stencil: boolean): Promise<string> => {
  try {
    const answer = stencil
      ? await cards.makeStencil(name, folder, [cut.newField])
      : await cards.makeDeck(name, folder)
    if (answer.refusal) {
      told(words.refused[answer.refusal], 'refusal')
      return ''
    }
    return answer.path
  } catch (error) {
    told(String(error), 'refusal')
    return ''
  }
}

/** The same, put in front of the person in a tab of its own. */
const opensCards = async (folder: string, name: string, stencil: boolean): Promise<string> => {
  const path = await makesCards(folder, name, stencil)
  if (!path) return ''
  puts.made(path, '', stencil ? 'stencil' : 'deck')
  return path
}

/**
 * A preset made in a folder under the name it is given, naming none of its
 * settings. Every key it does not carry stands at the default, so the decks
 * pointed at it are scheduled by the defaults until the person moves one.
 */
const makesPreset = async (folder: string, name: string): Promise<string> => {
  try {
    const answer = await presets.makes(name, folder)
    if (answer.refusal) {
      told(words.refused[answer.refusal], 'refusal')
      return ''
    }
    return answer.path
  } catch (error) {
    told(String(error), 'refusal')
    return ''
  }
}

/** The same, put in front of the person in a tab of its own. */
const opensMadePreset = async (folder: string, name: string): Promise<string> => {
  const path = await makesPreset(folder, name)
  if (!path) return ''
  puts.made(path, '', 'preset')
  return path
}

/** The tree of the vault, and what a gesture on a row of it comes to. */
const files = filesKind(held.host, () => folders(core), {
  lands: (landing) => void lands(landing, places),
  runs: (id, paths, name, source) => {
    const path = paths[0] ?? ''
    carries(id, { ...where(), path, title: name, file: path, source, others: paths.slice(1) })
  },
  moves: (from, to) => does(deedOf('move', { ...where(), path: from }, to), doing, words),
  carries: (paths) => {
    carried.value = paths
  },
  makes: (path) => does(deedOf('makeFolder', where(), path), doing, words),
  writes: async (folder) => (await making.named(folder, []))?.path ?? '',
  cuts: (folder, name) => makesCards(folder, name, false),
  stencils: (folder, name) => makesCards(folder, name, true),
  presets: (folder, name) => makesPreset(folder, name),
  says: (text) => told(text, 'refusal'),
})

/** The kinds this window draws. */
held.declares([
  noted.kind,
  plexes.kind,
  agents.kind,
  read.kind,
  heard.kind,
  files.kind,
  decks.kind,
  stencils.kind,
  schedules.kind,
])

/** The palette: one keystroke, and everything the words typed turn up. */
const palette = finding(core, words, undefined, meaning)

/** The vault this window is showing, as the list of vaults has it. */
const shown = ref<Shown>({ id: '', name: '' })

/** Every vault the installation holds, as the list last answered. */
const listed = ref<Listed>({ vaults: [], showing: '' })

/** What asking for the list of vaults leaves the person to be told. */
const unlisted = tell.under('listed')

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
 * run is over. A note tab means the note it holds and a plex tab the note it is
 * standing on; an agent means the note the plex the person was last in is
 * standing on; anything else means none. A document tab and a recording tab
 * each name the file they hold, which is what a run is asked over.
 */
const where = (): Where => {
  const ready = !failure.value && !indexing.value
  const vault = shown.value
  const front = held.host.front()
  const tab = front?.id ?? ''
  const kind = front?.kind ?? null
  const noRun = { file: '', source: null }
  if (kind === NOTE) {
    const note = held.host.holds<NoteHeld>(NOTE, tab)
    const path = note && notes.has(note.id) ? notes.where(note.id) : ''
    const title = note && path ? noted.titled(note.id) : ''
    return { tab, kind, path, title, ...noRun, vault, ready }
  }
  if (kind === PLEX) {
    const plex = held.host.holds<PlexHeld>(PLEX, tab)
    const path = plex?.view.here.value ?? ''
    const title = (path && plex?.nameOf(path)) || path
    return { tab, kind, path, title, ...noRun, vault, ready }
  }
  if (kind === AGENT) {
    const path = plexes.looking()
    const title = plexes.names(path) || path
    return { tab, kind, path, title, ...noRun, vault, ready }
  }
  if (kind === DOCUMENT || kind === RECORDING) {
    const file = held.host.holds<DocumentHeld | RecordingHeld>(kind, tab)?.path ?? ''
    const source = kind === DOCUMENT ? 'book' : 'recording'
    return { tab, kind, path: '', title: '', file, source, vault, ready }
  }
  return { tab, kind, path: '', title: '', ...noRun, vault, ready }
}

/**
 * The tab the person is looking at. A question is written into an agent tab, so
 * the tab in front of one is the one they were last in beside it.
 */
const looked = (): string => {
  const at = held.host.front()
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
 * A kind named here says what its tab holds; any other says what kind it is and
 * no more.
 */
const attends = (): Attention => ({
  front: looked(),
  tabs: held.tabs.value.map(({ id, title }) => {
    const kind = held.heldIn(id)?.kind.kind ?? ''
    const tab = { id, kind, title, path: '', at: 0, of: 0 }
    if (kind === NOTE) {
      const note = held.host.holds<NoteHeld>(NOTE, id)
      return { ...tab, path: note && notes.has(note.id) ? notes.where(note.id) : '' }
    }
    if (kind === PLEX) {
      return { ...tab, path: held.host.holds<PlexHeld>(PLEX, id)?.view.here.value ?? '' }
    }
    if (kind === DOCUMENT) {
      const page = held.host.holds<DocumentHeld>(DOCUMENT, id)
      if (!page) return tab
      return { ...tab, path: page.path, at: page.at.value + 1, of: page.pages.value }
    }
    if (kind === RECORDING) {
      const sound = held.host.holds<RecordingHeld>(RECORDING, id)
      if (!sound) return tab
      return { ...tab, path: sound.path, at: sound.heard.value, of: sound.length.value }
    }
    return tab
  }),
})

/** The same, told to the application as the window opens and whenever it changes. */
const attention = computed<Attention>(() => attends())
watch(
  attention,
  (open) => {
    void core.attending(open).catch((why) => {
      // An agent asking what is open is answered from what last arrived, so a
      // report that never lands leaves it reading a window that has moved on.
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
const stores: readonly Store[] = [
  {
    has: (id) => notes.has(id),
    where: (id) => notes.where(id),
    called: (id) => noted.titled(id),
    asking: (id) => notes.overtaken(id) !== null,
    settles: (id) => notes.settles(id),
    shuts: (id) => noted.shuts(id),
    holding: (path) => noted.holding(path),
  },
  decks.kept,
  stencils.kept,
]

/** The open files a command reaches, whichever of the stores holds each. */
const reached = reaching(stores, puts)

/** What one thing the quit is waiting on is called. */
const titled = (id: string): string => stores.find((one) => one.has(id))?.called(id) ?? ''

/** What the window knows about a note by the name it is filed under. */
const knows: Knows = {
  called: (path) => {
    const held = reached.holding(path)
    return held === null ? plexes.names(path) : titled(held)
  },
  holding: (path) => reached.holding(path),
}

/** How the window is drawn: the theme it wears, its half of a pair, its sizes. */
const dressed = wearing(themes, words, tell.under('worn'))

/** Whether a note's title and the name of its file are kept as one name. */
const oneName = syncing(core, words, tell.under('named'))

/** The hour a day of review begins at, on the clock on the wall. */
const dayBegins = reviewing(core, words, tell.under('reviewed'))

/** The rest of the settings file, which no command of the window turns. */
const rest = configuring(core, words, tell.under('configured'))

/**
 * Everything this installation is configured as, in a tab of its own. It holds
 * nothing: each row reaches the same value the command of that name reaches.
 */
const configured = settling(held.host, {
  themes: () => dressed.list.value,
  applied: () => dressed.applied.value,
  mode: () => dressed.mode.value,
  pinned: () => dressed.pinned.value,
  sizes: () => dressed.sized.value,
  bounds: () => dressed.bounds.value,
  chooses: (item) => void dressed.chooses(item),
  syncing: () => oneName.kept.value,
  choosesSyncing: (on) => void oneName.chooses(on ? ON : OFF),
  hangs: () => hungParts.hangs.value,
  parts: () => hungParts.parts.value,
  choosesHanging: (on) => void hungParts.chooses(on ? ON : OFF),
  choosesParts: (count) => void hungParts.choosesCount(`${count}`),
  dayStarts: () => dayBegins.starts.value,
  choosesDayStarts: (hour) => void dayBegins.chooses(hour),
  setting: (at) => rest.at(at),
  models: (at) => rest.offers(at),
  writes: (written) => void rest.chooses(written),
  file: () => rest.path.value,
})

held.declares([configured.kind])

// A size is drawn, and every open editor takes its measurements again. An
// editor watches its own box, and a size changes the type inside that box
// while the box itself stands.
watch(dressed.sized, () => noted.measures())

/** The lists the window itself holds, which a step of a command offers. */
const kept: Holds = {
  offers: (command, typed) => {
    if (command === APPEARANCE) return dressed.offers()
    if (command === MODE) return dressed.modes()
    if (command === INTERFACE_SCALE || command === TEXT_SCALE) return dressed.sizes(command, typed)
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
  const stands = (await core.standing([path])).get(path)
  if (stands?.type !== 'deck') return schedules.shows(path)
  const answer = await presets.scheduling(path)
  if (answer.refusal) return told(words.refused[answer.refusal], 'refusal')
  if (!answer.preset?.path) return told(words.noPreset, 'caution')
  schedules.shows(answer.preset.path, answer.preset.title)
}

/** The commands, over whatever is in front. */
const commands = commanding(core, words, where, knows, kept)

/** What the window offers a command being carried out. */
const doing: Doing = {
  makes: (title, from, seat) => making.calls(title, from, seat),
  renames: (path, title) => core.rename(path, title),
  removes: (path, destroy) => core.remove(path, destroy),
  moves: (from, to) => core.move(from, to),
  makesFolder: (path) => core.makeFolder(path),
  transcribes: (path) => running.transcribes(path),
  recognises: (path) => running.recognises(path),
  drops: async (path) => {
    const able = await running.drops(path)
    if (able) heard.dropped(path)
    return able
  },
  cuts: (folder, name) => opensCards(folder, name, false),
  stencils: (folder, name) => opensCards(folder, name, true),
  presets: (folder, name) => opensMadePreset(folder, name),
  reveals: (path) => void files.reveals(path),
  notes: reached,
  vaults,
  calls: (vault) => (shown.value = vault),
  reloads,
  travel: (path) => plexes.travel(path),
  leaves: (from, to) => plexes.leaves(from, to),
  opening: () => window.opening.value,
  opens: (kind) => void held.opens(kind),
  preset: (path) => opensPreset(path),
  // A kind with something to finish keeps its tab and closes it itself.
  closes: (tab) => {
    if (held.shut(tab)) layout.value = closeTab(layout.value, tab)
  },
  asks: (text) => void agents.asks(text),
  copies: (path) => void navigator.clipboard?.writeText(path),
  searches: () => {
    commands.shows(false)
    palette.shows(true)
  },
  appearance: (chosen) => dressed.chooses(chosen),
  syncing: (chosen) => oneName.chooses(chosen),
  hanging: (chosen) => hungParts.chooses(chosen),
  parts: (chosen) => hungParts.choosesCount(chosen),
  says: told,
}

/**
 * A command asked for, from the palette or from a menu on a node. One that
 * needs something asks for it, and the palette stands where it asks. One that
 * is not offered over what it was asked over says why.
 */
const carries = (id: string, at: Where) => {
  const deed = commands.asks(id, at)
  if (deed) return void does(deed, doing, words)
  if (commands.open.value) return palette.shows(false)
  told(commands.refused(id, at), 'refusal')
}

// What the welcome screen offers below the list of vaults.
const adding = computed(() => {
  const icon = iconFor('newVault')
  return {
    text: words.newVault,
    detail: words.newVaultDetail,
    ...(icon ? { icon } : {}),
    ...keysOf('newVault', navigator.userAgent),
  }
})

// The screen draws what it is given, so what stands in front of each way is
// chosen here, where the window's own icons are.
const ways = computed(() =>
  waysIn({ vault: shown.value.id, ready: where().ready }, words, navigator.userAgent).map(
    (one) => {
      const icon = iconFor(one.id)
      return { ...one, ...(icon ? { icon } : {}) }
    },
  ),
)
const onList = computed(() => vaultsOn(listed.value, words))

/**
 * A way in taken on the welcome screen. The commands are the window's own; the
 * rest are commands, and one that is refused says why.
 */
const runs = (id: string) => {
  if (id !== COMMANDS) return carries(id, where())
  palette.shows(false)
  commands.shows(true)
}

/** Whether the welcome screen is what the person is looking at and typing into. */
const welcoming = computed(
  () => held.tabs.value.length === 0 && !palette.open.value && !commands.open.value,
)

/** A vault chosen on the welcome screen, shown in this window in place of none. */
const opens = (id: string) => {
  const one = listed.value.vaults.find((vault) => vault.id === id)
  if (!one) return
  const vault: Shown = { id: one.id, name: one.name }
  void does(deedOf('openVault', { ...where(), vault }), doing, words)
}

/**
 * The keystrokes taken on the window: they belong to no pane. Two put the
 * palette up and take it down again, and the rest carry out the command the
 * chords name, which is the one written on that command's row.
 */
const asked = (event: KeyboardEvent) => {
  // A pane that has answered this keystroke keeps it.
  if (event.defaultPrevented) return
  // On the welcome screen a letter alone opens the vault standing at it, which
  // is the letter drawn on that row.
  if (welcoming.value) {
    const at = opensVault(event, onList.value.length)
    const one = at === null ? undefined : onList.value[at]
    if (one) {
      event.preventDefault()
      opens(one.id)
      return
    }
  }
  if (!chorded(event)) return
  const key = event.key.toLowerCase()
  // The two the window puts up are that letter alone. The same letter with
  // Shift is a chord of its own, and goes to whoever the table gives it to.
  if (!event.shiftKey && key === 'k') {
    event.preventDefault()
    commands.shows(false)
    palette.shows(!palette.open.value)
    return
  }
  if (!event.shiftKey && key === 'p') {
    event.preventDefault()
    palette.shows(false)
    commands.shows(!commands.open.value)
    return
  }
  const command = commandFor(key, event.shiftKey)
  if (!command) return
  event.preventDefault()
  carries(command, where())
}

/** What the palette draws: the commands while they are open, the search under. */
const field = computed(() =>
  commands.open.value
    ? {
        open: true,
        typed: commands.typed.value,
        bands: commands.bands.value,
        crumb: commands.crumb.value,
        step: commands.step.value,
        opensOn: commands.opensOn.value,
        placeholder: commands.placeholder.value,
      }
    : {
        open: palette.open.value,
        typed: palette.typed.value,
        bands: offering(palette.bands.value, palette.typed.value, words, where()),
        crumb: '',
        step: '',
        opensOn: '',
        placeholder: words.find,
      },
)

/**
 * Something typed in the field. The one character that means the commands is
 * the one typed into a field holding nothing.
 */
const typing = (text: string) => {
  if (commands.open.value) return void commands.typing(text)
  if (!asksCommands(palette.typed.value, text)) return void palette.typing(text)
  palette.shows(false)
  commands.shows(true)
}

/** Something chosen, and the window taken there or the command carried out. */
const went = async (item: string, action: string) => {
  if (commands.open.value) {
    const deed = commands.chose(item, action)
    if (!deed) return
    commands.shows(false)
    await does(deed, doing, words)
    return
  }
  if (item === MAKING) {
    const name = palette.typed.value.trim()
    palette.shows(false)
    await does(creates(action, name, where()), doing, words)
    return
  }
  const landing = palette.chose(item, action)
  palette.shows(false)
  await lands(landing, places)
}

/** Escape: a step of a command goes, and the palette itself at the last of them. */
const dismissed = () => (commands.open.value ? commands.leaves() : palette.shows(false))

/** Backspace in an empty field: the step goes, and the first hands back the search. */
const back = () => {
  if (commands.open.value && commands.backs()) palette.shows(true)
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
  await starts()
  void window.start()
  void going.start()
  void dressed.start()
  void oneName.start()
  void hungParts.start()
  void dayBegins.start()
  void rest.start()
})
onUnmounted(() => {
  globalThis.removeEventListener('keydown', asked)
  window.close()
  drawings.close()
  going.close()
  held.close()
  dressed.close()
})
</script>

<template>
  <main>
    <Failure :failure="failure" />

    <Workspace
      v-model="layout"
      class="below"
      :tabs="held.tabs.value"
      @close="shut"
      @show="held.shown"
    >
      <template #icon="{ id }">
        <component :is="tabIcon(id)" v-if="tabIcon(id)" class="tab-icon" />
      </template>

      <template #tab="{ id }">
        <component
          :is="held.heldIn(id)!.kind.draws"
          v-if="held.heldIn(id)"
          :held="held.heldIn(id)!.held"
        />

        <div v-else />
      </template>

      <template #silence>
        <Welcome
          :ways="ways"
          :vaults="onList"
          :heading="words.vaults"
          :offer="adding"
          :version="VERSION"
          @runs="runs"
          @opens="opens"
          @offers="carries('newVault', where())"
        />
      </template>
    </Workspace>

    <Notices
      :notices="notices"
      :name="words.working"
      :put-away="words.putAway"
      :more="words.more"
      @gone="tell.forget"
    />

    <Leaving :questions="going.questions.value" :called="titled" />

    <Palette
      :model-value="field.typed"
      :bands="field.bands"
      :open="field.open"
      :placeholder="field.placeholder"
      :crumb="field.crumb"
      :step="field.step"
      :opens-on="field.opensOn"
      :name="words.find"
      :actions-name="words.actions"
      :actions-placeholder="words.findAction"
      :actions-silence="words.noAction"
      @update:model-value="typing"
      @choose="went"
      @lit="commands.lights"
      @back="back"
      @dismiss="dismissed"
    >
      <!-- A command is drawn with its icon. What a search turns up is a note, a
           heading or a passage, and none of those is a command. -->
      <template v-if="commands.open.value" #icon="{ id }">
        <component :is="iconFor(id)" v-if="iconFor(id)" class="command-icon" />
      </template>
    </Palette>
  </main>
</template>

<style scoped>
main {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

.below {
  flex: 1;
  min-height: 0;
}

/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.command-icon,
.tab-icon {
  inline-size: 100%;
  block-size: 100%;
  stroke-width: 1.75;
  opacity: 0.75;
}
</style>
