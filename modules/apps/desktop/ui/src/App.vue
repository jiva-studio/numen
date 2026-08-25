<script setup lang="ts">
/**
 * The window: one vault, and tabs to divide the screen between.
 *
 * What a tab of each kind holds is that kind's own, in `plex/`, `agent/`,
 * `document/` and `note/`; keeping tabs of any kind at all is in
 * `windowing.ts`; when to ask the vault again is in `showing.ts`. What is left
 * here is the vault this window reads, the kinds it draws, and the few things
 * one kind asks of another.
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { closeTab, Notices, Palette, Workspace } from '@numen/ui'
import type { Notice } from '@numen/ui'
import '@numen/ui/styles.css'
import { core, documents, vaults } from './vault'
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
  MAKING,
  offering,
  type Holds,
  type Knows,
  type Shown,
  type Where,
} from './commanding'
import { themes } from './theme'
import { APPEARANCE, DRESSING, FONT, INTERFACE, MODE, wearing } from './wearing'
import { does, type Doing } from './doing'
import { finding } from './finding'
import { lands } from './landing'
import { leaving } from './leaving'
import { raising } from './raising'
import { windowing } from './windowing'
import BlankTab from './BlankTab.vue'
import Leaving from './Leaving.vue'
import Trouble from './Trouble.vue'
import { agentKind, talking } from './agent/kind'
import { documentKind, documenting } from './document/kind'
import { noting, type Held as NoteHeld } from './note/kind'
import { plexKind, plexing, type Held as PlexHeld } from './plex/kind'
import { core as agent } from './agent/core'
import { conversation } from './agent/conversation'
import { WORDS as talk } from './agent/words'
import { WORDS as words } from './words'
import { AGENT, CONVERSATION, NOTE, PLEX, named, opening } from './workspace'

const drawings = drawn()
const notes = editing(core, undefined, drawings.arrived)
const making = creating(core)
/** The page drawn again, which is a clean window on the vault that arrived. */
const reloads = () => globalThis.location.reload()
const window = showing(
  core,
  undefined,
  async (paths, renamed) => {
    notes.changed(paths, renamed)
    commands.follows(renamed)
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
/**
 * The one thing the window says while it still works: what it lost touch with,
 * or what the plex in front could not show.
 */
const warning = computed(() => window.lost.value || plexes.trouble())
/** What carrying a command out left the person to be told. */
const told = ref('')
/** What a command said, and what could not be made, joined or worn. */
const unmade = computed(() => told.value || making.said.value || dressed.said.value)
const { chunks, embedded, embedding, tasks } = window

/** How far this vault has been read for meaning, as the window last heard. */
const meaning = (): Meaning => ({
  chunks: chunks.value,
  embedded: embedded.value,
  embedding: embedding.value,
})

/** Everything running behind the window, as the corner draws it. */
const notices = computed<readonly Notice[]>(() => cornerOf(tasks.value, meaning(), words))

/** The tabs of this window, whatever kind each of them holds. */
const held = windowing(words)
const { layout, blanks, becomes } = held

/** The notes the window has open: what each is called, and what each tab of one holds. */
const noted = noting(core, notes, drawings, held.host, {
  makes: async () => {
    const made = await making.start()
    if (!made) return ''
    noted.calls(made.path, made.title)
    return made.path
  },
})

/** The plex tabs, and the one the person is looking at. */
const plexes = plexKind(held.host, () => standing(core), {
  makes: making,
  ready: () => !failure.value && !indexing.value,
  opens: (path, title, showing) => noted.shows(path, title, showing),
  asks: (text) => void agents.asks(text),
  runs: (id, path, title) => carries(id, { ...where(), path, title }),
  opening: () => window.opening.value,
  first: () => window.first(),
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

/** The kinds this window draws, in the order a blank tab offers them. */
held.declares([noted.kind, plexes.kind, agents.kind, read.kind])

/** The palette: one keystroke, and everything the words typed turn up. */
const palette = finding(core, words, undefined, meaning)

/** The vault this window is showing, as the list of vaults has it. */
const shown = ref<Shown>({ id: '', name: '' })

/** Which of the vaults on the list this window is showing, and what it is called. */
const listing = async () => {
  try {
    const listed = await core.vaults()
    const one = listed.vaults.find((vault) => vault.id === listed.showing)
    shown.value = one ? { id: one.id, name: one.name } : { id: '', name: '' }
  } catch {
    // The commands over this vault are offered once the list has answered.
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

/** The identity of the note tab standing at a file, and nothing where none does. */
const holding = (path: string): string | null => noted.holding(path)

/** What the window knows about a note by the name it is filed under. */
const knows: Knows = {
  called: (path) => (holding(path) ? noted.called(path) : plexes.names(path)),
  holding,
}

/** How the window is drawn: the theme it wears, its half of a pair, its sizes. */
const dressed = wearing(themes, words)

/** The lists the window itself holds, which a step of a command offers. */
const kept: Holds = {
  offers: (command) => {
    if (command === APPEARANCE) return dressed.offers()
    if (command === MODE) return dressed.modes()
    if (command === INTERFACE || command === FONT) return dressed.sizes(command)
    return []
  },
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
  notes: {
    holding,
    where: (id) => notes.where(id),
    asking: (id) => notes.overtaken(id) !== null,
    settles: (id) => notes.settles(id),
    shuts: (id) => noted.shuts(id),
    shows: (path, title, showing) => noted.shows(path, title, showing),
  },
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
  says: (text) => (told.value = text),
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
  told.value = commands.refused(id, at)
}

/**
 * The palette is opened and put away by two keystrokes, taken on the window: it
 * belongs to no pane.
 */
const asked = (event: KeyboardEvent) => {
  // A pane that has answered this keystroke keeps it.
  if (event.defaultPrevented) return
  if (event.altKey || !(event.metaKey || event.ctrlKey)) return
  const key = event.key.toLowerCase()
  if (key === 'k') {
    event.preventDefault()
    commands.shows(false)
    palette.shows(!palette.open.value)
    return
  }
  if (key !== 'p') return
  // The window takes this keystroke.
  event.preventDefault()
  palette.shows(false)
  commands.shows(!commands.open.value)
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
        placeholder: commands.placeholder.value,
      }
    : {
        open: palette.open.value,
        typed: palette.typed.value,
        bands: offering(palette.bands.value, palette.typed.value, words, where()),
        crumb: '',
        step: '',
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
  await lands(landing, {
    travel: (path) => plexes.travel(path),
    opensAt: (path, run) => read.opensAt(path, run),
    shows: (path, title) => noted.shows(path, title),
    entersAt: (path, line) => noted.entersAt(path, line),
  })
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

/** The window opens with a plex holding the room and an agent along the edge. */
const starts = async () => {
  const plex = await held.opens(PLEX)
  const talk = await held.opens(AGENT)
  layout.value = opening(plex, talk)
}

onMounted(async () => {
  globalThis.addEventListener('keydown', asked)
  // The layout the window opens with stands before anything the vault says can
  // open a tab of its own.
  await starts()
  void listing()
  void window.start()
  void going.start()
  void dressed.start()
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
    <Trouble
      :unwatched="unwatched"
      :trouble="trouble"
      :warning="warning"
      :unmade="unmade"
      :failure="failure"
      :indexing="indexing"
      :holds="holds"
    />

    <Workspace
      v-model="layout"
      class="below"
      :tabs="held.tabs.value"
      :new-tab="words.newTab"
      @close="shut"
      @show="held.shown"
      @open="held.blanked"
    >
      <template #tab="{ id }">
        <component
          :is="held.heldIn(id)!.kind.draws"
          v-if="held.heldIn(id)"
          :held="held.heldIn(id)!.held"
        />

        <BlankTab
          v-else-if="blanks.includes(id)"
          :becomes="becomes"
          @choose="(kind: string) => void held.becomeIt(id, kind)"
        />

        <div v-else />
      </template>
    </Workspace>

    <Notices :notices="notices" :name="words.working" :put-away="words.putAway" />

    <Leaving :questions="going.questions.value" :called="noted.titled" />

    <Palette
      :model-value="field.typed"
      :bands="field.bands"
      :open="field.open"
      :placeholder="field.placeholder"
      :crumb="field.crumb"
      :step="field.step"
      :name="words.find"
      :actions-name="words.actions"
      :actions-placeholder="words.findAction"
      :actions-silence="words.noAction"
      @update:model-value="typing"
      @choose="went"
      @lit="commands.lights"
      @back="back"
      @dismiss="dismissed"
    />
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

</style>
