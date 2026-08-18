<script lang="ts">
/**
 * The one word a tab carries beside its title, and what a screen reader reads
 * out. A state with no word carries no mark; `stateOf` holds a tab in one
 * state, so the precedence the words are declared in is the precedence drawn.
 */
import type { State } from './tab'

const MARKS: Record<State, string | undefined> = {
  stuck: 'stuck',
  gone: 'gone',
  overtaken: 'overtaken',
  unsaved: 'unsaved',
  saving: 'unsaved',
  loading: undefined,
  clean: undefined,
}

export const markOf = (state: State): string | undefined => MARKS[state]
</script>

<script setup lang="ts">
/**
 * The window: one vault, and tabs to divide the screen between.
 *
 * Choosing a node asks for that note's neighbourhood and hands it back to the
 * plex, which travels there by itself. What decides when to ask is in
 * `showing.ts`, what a tab holds is in `holding.ts`, and what is drawn from
 * either is here.
 */
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import {
  Activity,
  Agent,
  Editor,
  Menu,
  Plex,
  Workspace,
  closeTab,
  openTab,
  openTabBeside,
  remainingWord,
} from '@numen/ui'
import type { MenuOpening, PlexNeighbourhood, PlexRelatedSeat, PlexShowing, Tab } from '@numen/ui'
import '@numen/ui/styles.css'
import { core } from './vault'
import { showing } from './showing'
import { footOf, type Phase } from './foot'
import { editing } from './editing'
import { drawn } from './drawn'
import { creating, CREATABLE } from './creating'
import { holding, type Talk } from './holding'
import { ITEMS, chose as carry } from './menu'
import { leaving } from './leaving'
import { core as agent } from './agent'
import { conversation } from './conversation'
import { AGENT, NOTE, PLEX, plexCalled, shortened } from './workspace'

const drawings = drawn()
const notes = editing(core, undefined, drawings.arrived)
const making = creating(core)
const window = showing(core, undefined, undefined, notes.changed, drawings.told, (path) =>
  held.shows(path),
)
/** What the window answers when the application says it is going. */
const going = leaving(core)
going.holds(notes.flush)

/**
 * The notes whose text the file has moved past, told to the quit.
 *
 * A question is raised while the tab stands overtaken and dropped when it
 * stops, so the window waits on exactly what is still to be answered.
 */
const raised = new Map<string, () => void>()
watch(
  () => notes.all().filter((path) => notes.shown(path).state === 'overtaken'),
  (standing) => {
    for (const path of standing) {
      if (raised.has(path)) continue
      raised.set(
        path,
        going.raise({ path, keep: async () => notes.keep(path), take: async () => notes.take(path) }),
      )
    }
    for (const [path, drop] of raised) {
      if (standing.includes(path)) continue
      drop()
      raised.delete(path)
    }
  },
  { deep: true },
)
const { indexing, failure, notice, trouble, unwatched, holds, looking } = window
/** What could not be made or joined, in words a person reads. */
const unmade = computed(() => making.said.value)
const { chunks, embedded, reading, embedding, books, booksRead, learning, rate } = window
/** What the vault says about having work in hand. The counts do not say it. */
const { working: reads } = window

/** Everything this window says in its own voice. */
const words = {
  ask: 'Ask about this note',
  thinking: 'Thinking',
  unreachable: 'The agent could not be reached.',
  nothing: 'The agent finished without saying anything.',
  unsent: 'Did not send',
  send: 'Send',
  stop: 'Stop the agent',
  newTab: 'New tab',
  choose: 'What goes in this tab',
  newNote: 'New note',
  newPlex: 'New plex',
  newAgent: 'New agent',
  plex: 'Plex',
  agent: 'Agent',
  stopped: 'The agent stopped here',
  reading: 'Reading',
  learning: 'Preparing search by meaning',
  words: 'Searching by words only — no model set',
  overtaken: 'The file changed on disk, so this note stopped saving.',
  gone: 'This note is no longer in the vault, so saving stopped. What is here is still yours.',
  makeAgain: 'make it again',
  keep: 'Keep mine',
  take: "Take the file's",
  going: 'These notes stopped saving because their files changed. The window waits.',
  later: 'Not yet',
}

/** What the foot of the window says, one sentence per phase. */
const saying: Record<Phase, string> = {
  reading: words.reading,
  learning: words.learning,
  wordsOnly: words.words,
  idle: '',
}

const activity = computed(() => {
  const foot = footOf({
    busy: reads.value,
    learning: learning.value,
    reading: reading.value,
    books: books.value,
    booksRead: booksRead.value,
    chunks: chunks.value,
    embedded: embedded.value,
    embedding: embedding.value,
    rate: rate.value,
  })
  return {
    says: saying[foot.phase],
    about: foot.about,
    working: foot.working,
    left: remainingWord(foot.left, foot.perSecond),
    tally: foot.tally,
  }
})

/** What a note was called by the node it was opened from. */
const titles = new Map<string, string>()

/** What each tab of the window holds, and what it lets go of when it closes. */
const held = holding({
  plex: window.plex,
  talk: (name) => conversation(agent, words, name),
  note: async () => {
    const made = await making.start()
    if (!made) return ''
    titles.set(made.path, made.title)
    notes.open(made.path)
    return made.path
  },
})
const { layout, blanks } = held

/** A question is about the note the person is looking at. */
const send = (id: string, text: string) => {
  const talk = held.agents.value.get(id)
  if (!talk) return
  talk.asked.value = ''
  void talk.ask(text, looking.value)
}

/** The identity a pane made by a split is filed under. */
const naming = () => crypto.randomUUID()

/** What the window can be told to put in a blank tab. */
const becomes: readonly Tab[] = [
  { id: NOTE, title: words.newNote },
  { id: PLEX, title: words.newPlex },
  { id: AGENT, title: words.newAgent },
]

/** An agent is called by the first thing asked of it. */
const agentCalled = (talk: Talk): string =>
  shortened(talk.turns.value.find((turn) => turn.voice === 'asked')?.text ?? '') || words.agent

/** Every tab the window holds, and what each is called. */
const tabs = computed<readonly Tab[]>(() => [
  ...[...held.plexes.value].map(
    ([id, one]): Tab => ({
      id,
      title: plexCalled(words.plex, one.view.neighbourhood.value?.focus?.title ?? ''),
    }),
  ),
  ...[...held.agents.value].map(([id, talk]): Tab => ({ id, title: agentCalled(talk) })),
  ...blanks.value.map((id): Tab => ({ id, title: words.newTab })),
  ...notes.all().map((path): Tab => {
    const mark = markOf(notes.shown(path).state)
    return { id: path, title: titles.get(path) ?? path, ...(mark ? { mark } : {}) }
  }),
])

/** The menu on a node: the plex it was asked in, where, and by what. */
const menu = ref<{
  tab: string
  path: string
  at: { x: number; y: number }
  from: HTMLElement | SVGElement | null
  opening: MenuOpening
} | null>(null)

const askMenu = (
  tab: string,
  path: string,
  at: { x: number; y: number },
  from: HTMLElement | SVGElement | null,
  opening: MenuOpening,
) => {
  menu.value = { tab, path, at, from, opening }
}

const chose = (id: string) => {
  const asking = menu.value
  menu.value = null
  if (!asking) return
  carry(id, asking.path, {
    open: (path) => openNote(asking.tab, path),
    child: (path) => void made(asking.tab, path, 'child'),
    ask: (path) => held.askAbout(`${path} — `),
    copy: (path) => void navigator.clipboard?.writeText(path),
  })
}

/** A note opens where the person asked for it, and the tab is shown. */
const openNote = (tab: string, path: string, showing: PlexShowing = 'here') => {
  titles.set(path, nameOf(tab, path))
  notes.open(path)
  layout.value =
    showing === 'beside'
      ? openTabBeside(layout.value, path, 'right', naming)
      : openTab(layout.value, path)
}

/**
 * A note made in a seat of another one. It is in the index by the time the
 * answer arrives, so the picture is asked for again and it is drawn in it.
 */
const made = async (tab: string, from: string, seat: PlexRelatedSeat) => {
  const view = plexIn(tab)
  if ((await making.make(from, seat)) && view?.here.value) await view.go(view.here.value)
}

/** Two notes the person drew a line between. */
const joined = async (tab: string, from: string, to: string, seat: PlexRelatedSeat) => {
  const view = plexIn(tab)
  if ((await making.join(from, to, seat)) && view?.here.value) await view.go(view.here.value)
}

const nameOf = (tab: string, path: string): string => {
  const around = plexIn(tab)?.neighbourhood.value
  if (around?.focus?.path === path && around.focus.title) return around.focus.title
  const near = around?.related?.find((r) => r.note?.path === path)
  return near?.note?.title || (path.split('/').pop() ?? path).replace(/\.md$/, '')
}

/** The picture one plex tab draws, or nothing while it has none. */
const picture = (id: string): PlexNeighbourhood | null => {
  if (failure.value || indexing.value) return null
  return held.plexes.value.get(id)?.picture.value ?? null
}

/** Where one plex tab is standing. */
const plexIn = (id: string) => held.plexes.value.get(id)?.view ?? null

/** The talk one agent tab holds. */
const talkIn = (id: string) => held.agents.value.get(id) ?? null

/** The editor of each open note, for as long as its tab is drawn. */
const editors = new Map<string, { measure: () => void }>()

const drew = (path: string, editor: unknown) => {
  if (editor) editors.set(path, editor as { measure: () => void })
  else editors.delete(path)
}

/**
 * A tab is drawn while it is out of sight, where an editor has nothing to
 * measure. The editor of the tab now on screen takes its measurements again.
 */
const shown = (id: string) => {
  held.shown(id)
  editors.get(id)?.measure()
}

/** A tab lets go of what it held. A tab that holds a note writes what it owes. */
const shut = (id: string, hold: () => void) => {
  if (held.shut(id)) return
  if (!notes.all().includes(id)) return
  hold()
  drawings.shut(id)
  void notes.shut(id).then((gone) => {
    if (!gone) return
    titles.delete(id)
    layout.value = closeTab(layout.value, id)
  })
}

onMounted(() => {
  void window.start()
  void going.start()
})
onUnmounted(() => {
  window.close()
  drawings.close()
  going.close()
  held.close()
})
</script>

<template>
  <main>
    <p v-if="unwatched" class="warning">not following the vault — {{ unwatched }}</p>
    <p v-if="trouble" class="warning">the vault could not be read — {{ trouble }}</p>
    <p v-if="notice" class="warning">{{ notice }}</p>

    <p v-if="unmade" role="alert" class="warning">{{ unmade }}</p>

    <p v-if="failure" class="failure">{{ failure }}</p>
    <p v-else-if="indexing" class="waiting">reading the vault…</p>
    <p v-else-if="!holds && trouble" class="waiting">nothing was read</p>
    <p v-else-if="!holds" class="waiting">this vault holds no notes</p>

    <Workspace
      v-model="layout"
      class="below"
      :tabs="tabs"
      :new-tab="words.newTab"
      @close="shut"
      @show="shown"
      @open="held.blanked"
    >
      <template #tab="{ id }">
        <Plex
          v-if="picture(id)"
          :neighbourhood="picture(id)!"
          :creatable="CREATABLE"
          @activate="(path: string) => void plexIn(id)?.go(path)"
          @create="(from: string, seat: PlexRelatedSeat) => void made(id, from, seat)"
          @link="
            (from: string, to: string, seat: PlexRelatedSeat) => void joined(id, from, to, seat)
          "
          @menu="
            (
              path: string,
              at: { x: number; y: number },
              from: SVGGElement,
              opening: MenuOpening,
            ) => askMenu(id, path, at, from, opening)
          "
          @show="(path: string, how: PlexShowing) => openNote(id, path, how)"
          @dismiss="menu = null"
        />

        <Agent
          v-else-if="talkIn(id)"
          :model-value="talkIn(id)!.asked.value"
          :turns="talkIn(id)!.turns.value"
          :working="talkIn(id)!.working.value"
          :placeholder="words.ask"
          :sends="words.send"
          :stops="words.stop"
          @update:model-value="(text: string) => held.writing(id, text)"
          @submit="(text: string) => send(id, text)"
          @stop="talkIn(id)?.stop()"
        >
          <template #failure="{ turn }">
            {{ turn.voice === 'asked' ? words.unsent : words.stopped }}
          </template>
        </Agent>

        <section v-else-if="blanks.includes(id)" class="blank">
          <p class="blank__says">{{ words.choose }}</p>
          <ul class="blank__choices">
            <li v-for="one in becomes" :key="one.id">
              <button type="button" class="blank__choice" @click="void held.becomeIt(id, one.id)">
                {{ one.title }}
              </button>
            </li>
          </ul>
        </section>

        <div v-else-if="notes.all().includes(id)" class="note">
          <p v-if="notes.saying(id)" role="alert" class="warning">{{ notes.saying(id) }}</p>


          <p v-if="notes.shown(id).state === 'gone'" role="status" class="warning overtaken">
            {{ words.gone }}
            <button type="button" class="overtaken__answer" @click="notes.keep(id)">
              {{ words.makeAgain }}
            </button>
          </p>
          <p
            v-if="notes.shown(id).state === 'overtaken'"
            role="status"
            class="warning overtaken"
          >
            {{ words.overtaken }}
            <button type="button" class="overtaken__answer" @click="notes.keep(id)">
              {{ words.keep }}
            </button>
            <button type="button" class="overtaken__answer" @click="notes.take(id)">
              {{ words.take }}
            </button>
          </p>

          <Editor
            :ref="(editor: unknown) => drew(id, editor)"
            :model-value="notes.shown(id).body"
            :change="drawings.shown(id)"
            class="note__text"
            @update:model-value="(body: string) => notes.typed(id, body)"
            @save="notes.save(id)"
          />
        </div>

        <div v-else />
      </template>
    </Workspace>

    <Activity
      class="activity"
      :says="activity.says"
      :about="activity.about"
      :working="activity.working"
      :left="activity.left"
      :tally="activity.tally"
    />

    <section v-if="going.questions.value.length" role="alertdialog" class="leaving">
      <p class="leaving__says">{{ words.going }}</p>
      <ul class="leaving__notes">
        <li v-for="one in going.questions.value" :key="one.path" class="leaving__note">
          <span class="leaving__title">{{ titles.get(one.path) ?? one.path }}</span>
          <button type="button" class="overtaken__answer" @click="void one.keep()">
            {{ words.keep }}
          </button>
          <button type="button" class="overtaken__answer" @click="void one.take()">
            {{ words.take }}
          </button>
          <button type="button" class="overtaken__answer" @click="one.later()">
            {{ words.later }}
          </button>
        </li>
      </ul>
    </section>

    <Menu
      v-if="menu"
      :items="ITEMS"
      :at="menu.at"
      :from="menu.from"
      :opening="menu.opening"
      open
      @choose="chose"
      @dismiss="menu = null"
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

/* A note fills the pane it is in: the editor scrolls, and the line it says
   something is wrong on stays where it is. */
.note {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
}

.note__text {
  flex: 1;
  min-block-size: 0;
}

/* A tab with nothing in it yet, and the few things it can be told to be. */
.blank {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
  block-size: 100%;
  padding: var(--numen-gutter);
  font: 0.85rem system-ui, sans-serif;
}

.blank__says {
  margin: 0;
  color: var(--numen-edge-label);
}

.blank__choices {
  display: flex;
  flex-direction: column;
  align-items: start;
  gap: 0.25rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.blank__choice {
  padding: 0.3rem 0.6rem;
  border: 0;
  border-radius: var(--numen-radius);
  background: none;
  color: var(--numen-node-fg);
  font: inherit;
  cursor: pointer;
}

.blank__choice:hover {
  background: var(--numen-bubble-bg);
}

.blank__choice:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: 1px;
}

/* The question a note holds: the band a refusal is said in, with the two
   answers on the same line as the sentence, so the band stands one line high. */
/* The window is going and these notes are not written. It sits over the work
   because nothing else the person does can end it. */
.leaving {
  position: fixed;
  inset-block-end: 1rem;
  inset-inline: 1rem;
  z-index: 20;
  padding: 0.8rem 1rem;
  border-radius: var(--numen-radius);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  box-shadow: var(--numen-panel-shadow);
  font: 0.85rem system-ui, sans-serif;
}

.leaving__says {
  margin: 0 0 0.5rem;
}

.leaving__notes {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.leaving__note {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.leaving__title {
  flex: 1;
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.overtaken {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0 0.9rem;
}

.overtaken__answer {
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-decoration: underline;
  text-underline-offset: 0.15em;
  cursor: pointer;
}

.overtaken__answer:hover {
  text-decoration-thickness: 2px;
}

.overtaken__answer:focus-visible {
  outline: 1px solid currentColor;
  outline-offset: 2px;
}

/* The foot of the window: clear of the plex, quiet when there is no work. */
.activity {
  flex: none;
  padding: 0.3rem 1rem;
}

.waiting,
.failure {
  margin: auto;
  font: 0.9rem system-ui, sans-serif;
  opacity: 0.6;
}

.warning {
  margin: 0;
  padding: 0.4rem 1rem;
  font: 0.8rem system-ui, sans-serif;
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
}

.failure {
  color: var(--numen-alarm);
  opacity: 1;
  max-width: 40rem;
  text-align: center;
}
</style>
