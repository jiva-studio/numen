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
import { closeTab, Notices, Palette, Workspace } from '@numen/ui'
import type { Notice } from '@numen/ui'
import '@numen/ui/styles.css'
import { cards, core, documents, vaults } from './vault'
import type { Listed } from './core'
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
import { chorded, commandFor } from './keying'
import { iconFor, iconOfKind } from './icons'
import { themes } from './theme'
import { APPEARANCE, DRESSING, INTERFACE_SCALE, MODE, TEXT_SCALE, wearing } from './wearing'
import { SYNCING, syncing } from './syncing'
import { HANGING, PARTS, hanging } from './hanging'
import { does, reaching, type Doing, type Store } from './doing'
import { finding } from './finding'
import { lands, type Places } from './landing'
import { leaving } from './leaving'
import { raising } from './raising'
import { windowing } from './windowing'
import Leaving from './Leaving.vue'
import Failure from './Failure.vue'
import Welcome from './welcome/Welcome.vue'
import { COMMANDS, vaultsOn, waysIn } from './welcome/welcoming'
import { telling } from './telling'
import { agentKind, talking } from './agent/kind'
import { decking } from './cards/deck'
import { stencilling } from './cards/stencil'
import { documentKind, documenting } from './document/kind'
import { filesKind } from './files/kind'
import { listing as folders } from './files/listing'
import { noting, type Held as NoteHeld } from './note/kind'
import { plexKind, plexing, type Held as PlexHeld } from './plex/kind'
import { core as agent } from './agent/core'
import { conversation } from './agent/conversation'
import { WORDS as talk } from './agent/words'
import { WORDS as words } from './words'
import { VERSION } from './version'
import { AGENT, CONVERSATION, FILES, NOTE, PLEX, named, opening } from './workspace'

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
    commands.follows(renamed)
    await files.changed(paths, renamed)
    await plexes.again(renamed)
  },
  drawings.told,
  (path) => plexes.travel(path),
  (path, runs) => void read.opensAt(path, ...runs),
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

/** The notes the window has open: what each is called, and what each tab of one holds. */
const noted = noting(core, notes, drawings, held.host)

/** The decks and the stencils the window has open, each saved the way a note is. */
const decks = decking(cards, held.host)
const stencils = stencilling(cards, held.host)

going.holds(decks.flush)
going.holds(stencils.flush)
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
  opens: (path, title, showing) => noted.shows(path, title, showing),
  entersAt: (path, line) => noted.entersAt(path, line),
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
    looking: () => plexes.looking(),
    opens: (path, ...runs) => void read.opensAt(path, ...runs),
    unreachable: () => unreachable.value,
  }),
)

/** The document tabs, each reading the document it is filed at. */
const read = documentKind(held.host, (path) => documenting(reading(documents, path)))

/** Where the window is taken when something is chosen, wherever it was chosen. */
const places: Places = {
  travel: (path) => plexes.travel(path),
  opensAt: (path, run) => read.opensAt(path, run),
  shows: (path, title) => noted.shows(path, title),
  deck: (path, title) => decks.shows(path, title),
  stencil: (path, title) => stencils.shows(path, title),
  entersAt: (path, line) => noted.entersAt(path, line),
}

/**
 * A deck or a stencil made in a folder under the name it is given. The file
 * carries what it is from the moment it is written, so making one is a write of
 * no cards or of no fields.
 */
const makesCards = async (folder: string, name: string, stencil: boolean): Promise<string> => {
  const path = folder ? `${folder}/${name}` : name
  const answer = stencil
    ? await cards.writeStencil(path, [], [], null)
    : await cards.writeDeck(path, { preamble: '', cards: [], tail: '' }, null)
  if (answer.refusal) {
    told(words.refused[answer.refusal], 'refusal')
    return ''
  }
  return path
}

/** The same, put in front of the person in a tab of its own. */
const opensCards = async (folder: string, name: string, stencil: boolean): Promise<string> => {
  const path = await makesCards(folder, name, stencil)
  if (!path) return ''
  if (stencil) stencils.shows(path)
  else decks.shows(path)
  return path
}

/** The tree of the vault, and what a gesture on a row of it comes to. */
const files = filesKind(held.host, () => folders(core), {
  lands: (landing) => void lands(landing, places),
  runs: (id, paths, name) =>
    carries(id, { ...where(), path: paths[0] ?? '', title: name, others: paths.slice(1) }),
  moves: (from, to) => does(deedOf('move', { ...where(), path: from }, to), doing, words),
  carries: (paths) => {
    carried.value = paths
  },
  makes: (path) => does(deedOf('makeFolder', where(), path), doing, words),
  writes: async (folder) => (await making.named(folder, []))?.path ?? '',
  cuts: (folder, name) => makesCards(folder, name, false),
  stencils: (folder, name) => makesCards(folder, name, true),
  says: (text) => told(text, 'refusal'),
})

/** The kinds this window draws. */
held.declares([
  noted.kind,
  plexes.kind,
  agents.kind,
  read.kind,
  files.kind,
  decks.kind,
  stencils.kind,
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
 * What a command is over: the tab in front, and the note it means. A note tab
 * means the note it holds and a plex tab the note it is standing on; an agent
 * means the note the plex the person was last in is standing on; anything else
 * means none.
 */
const where = (): Where => {
  const ready = !failure.value && !indexing.value
  const vault = shown.value
  const front = held.host.front()
  const tab = front?.id ?? ''
  const kind = front?.kind ?? null
  if (kind === NOTE) {
    const note = held.host.holds<NoteHeld>(NOTE, tab)
    const path = note && notes.has(note.id) ? notes.where(note.id) : ''
    return { tab, kind, path, title: note && path ? noted.titled(note.id) : '', vault, ready }
  }
  if (kind === PLEX) {
    const plex = held.host.holds<PlexHeld>(PLEX, tab)
    const path = plex?.view.here.value ?? ''
    return { tab, kind, path, title: (path && plex?.nameOf(path)) || path, vault, ready }
  }
  if (kind === AGENT) {
    const path = plexes.looking()
    return { tab, kind, path, title: plexes.names(path) || path, vault, ready }
  }
  return { tab, kind, path: '', title: '', vault, ready }
}

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
const reached = reaching(stores, (path, title, showing) => noted.shows(path, title, showing))

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

/** The commands, over whatever is in front. */
const commands = commanding(core, words, where, knows, kept)

/** What the window offers a command being carried out. */
const doing: Doing = {
  makes: (title, from, seat) => making.calls(title, from, seat),
  renames: (path, title) => core.rename(path, title),
  removes: (path, destroy) => core.remove(path, destroy),
  moves: (from, to) => core.move(from, to),
  makesFolder: (path) => core.makeFolder(path),
  cuts: (folder, name) => opensCards(folder, name, false),
  stencils: (folder, name) => opensCards(folder, name, true),
  reveals: (path) => void files.reveals(path),
  notes: reached,
  vaults,
  calls: (vault) => (shown.value = vault),
  reloads,
  travel: (path) => plexes.travel(path),
  leaves: (from, to) => plexes.leaves(from, to),
  opening: () => window.opening.value,
  opens: (kind) => void held.opens(kind),
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

/** The ways in the welcome screen offers, and the vaults it draws. */
const ways = computed(() =>
  waysIn({ vault: shown.value.id, ready: where().ready }, words, navigator.userAgent),
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
          :words="words"
          :version="VERSION"
          @runs="runs"
          @opens="opens"
          @adds="carries('newVault', where())"
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
