<script lang="ts">
/**
 * The one word a tab carries beside its title, and what a screen reader reads
 * out. A state with no word carries no mark; `stateOf` holds a tab in one
 * state, so the precedence the words are declared in is the precedence drawn.
 */
import type { State } from './tab'

const MARKS: Record<State, string | undefined> = {
  stuck: 'stuck',
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
 * `showing.ts`; what each tab stands for is settled here and nowhere else.
 */
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { Activity, Agent, Editor, Menu, Plex, Workspace, closeTab, openTab, remainingWord } from '@numen/ui'
import type { PlexRelatedSeat, Tab, WorkspaceLayout } from '@numen/ui'
import '@numen/ui/styles.css'
import { core } from './vault'
import { showing } from './showing'
import { footOf, type Phase } from './foot'
import { editing } from './editing'
import { drawn } from './drawn'
import { creating, CREATABLE } from './creating'
import { ITEMS, chose as carry } from './menu'
import { leaving } from './leaving'
import { asPlex } from './plex'
import { core as agent } from './agent'
import { conversation } from './conversation'
import { AGENT, PLEX, TABS, opening } from './workspace'

const drawings = drawn()
const notes = editing(core, undefined, drawings.arrived)
const making = creating(core)
const window = showing(core, undefined, undefined, notes.changed, drawings.told)
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
const { neighbourhood, here, indexing, failure, notice, trouble, unwatched, go } = window
/** What could not be made or joined, in words a person reads. */
const unmade = computed(() => making.said.value)
/** The plex reads one value, so what it is given changes when the vault does. */
const plexed = computed(() => (neighbourhood.value ? asPlex(neighbourhood.value) : null))
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
  stopped: 'The agent stopped here',
  reading: 'Reading',
  learning: 'Preparing search by meaning',
  words: 'Searching by words only — no model set',
  overtaken: 'The file changed on disk, so this note stopped saving.',
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

const { turns, working, ask, close } = conversation(agent, words)
const asked = ref('')
const layout = ref<WorkspaceLayout>(opening())

const send = (text: string) => {
  asked.value = ''
  void ask(text, neighbourhood.value?.focus?.path ?? '')
}

/** Every tab the window holds: the two it opens with, and one per open note. */
const tabs = computed<readonly Tab[]>(() => [
  ...TABS,
  ...notes.all().map((path): Tab => {
    const mark = markOf(notes.shown(path).state)
    return { id: path, title: titles.get(path) ?? path, ...(mark ? { mark } : {}) }
  }),
])

/** What a note was called by the node it was opened from. */
const titles = new Map<string, string>()

/** The menu on a node, and where it was asked for. */
const menu = ref<{ path: string; at: { x: number; y: number }; from: HTMLElement | SVGElement | null } | null>(null)

const askMenu = (path: string, at: { x: number; y: number }, from: HTMLElement | SVGElement | null) => {
  menu.value = { path, at, from }
}

const chose = (id: string) => {
  const asking = menu.value
  menu.value = null
  if (!asking) return
  carry(id, asking.path, {
    open: (path) => openNote(path),
    child: (path) => void made(path, 'child'),
    ask: (path) => {
      asked.value = path + ' — '
      layout.value = openTab(layout.value, AGENT)
    },
    copy: (path) => void navigator.clipboard?.writeText(path),
  })
}

/** A note opens in the pane the person is in, and the tab is shown. */
const openNote = (path: string, title = nameOf(path)) => {
  titles.set(path, title)
  notes.open(path)
  layout.value = openTab(layout.value, path)
}

/**
 * A note made in a seat of another one. It is in the index by the time the
 * answer arrives, so the picture is asked for again and it is drawn in it.
 */
const made = async (from: string, seat: PlexRelatedSeat) => {
  if ((await making.make(from, seat)) && here.value) await go(here.value)
}

/** Two notes the person drew a line between. */
const joined = async (from: string, to: string, seat: PlexRelatedSeat) => {
  if ((await making.join(from, to, seat)) && here.value) await go(here.value)
}

const nameOf = (path: string): string => {
  const named = neighbourhood.value
  if (named?.focus?.path === path && named.focus.title) return named.focus.title
  const near = named?.related?.find((r) => r.note?.path === path)
  return near?.note?.title || (path.split('/').pop() ?? path).replace(/\.md$/, '')
}

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
const shown = (id: string) => editors.get(id)?.measure()

/** A tab that holds a note writes what it owes before it goes. */
const shut = (id: string, hold: () => void) => {
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
  close()
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
    <p v-else-if="!neighbourhood && trouble" class="waiting">nothing was read</p>
    <p v-else-if="!neighbourhood" class="waiting">this vault holds no notes</p>

    <Workspace v-model="layout" class="below" :tabs="tabs" @close="shut" @show="shown">
      <template #tab="{ id }">
        <Plex
          v-if="id === PLEX && plexed && !failure && !indexing"
          :neighbourhood="plexed!"
          :creatable="CREATABLE"
          @activate="go"
          @create="(from: string, seat: PlexRelatedSeat) => void made(from, seat)"
          @link="(from: string, to: string, seat: PlexRelatedSeat) => void joined(from, to, seat)"
          @menu="askMenu"
          @dismiss="menu = null"
        />

        <Agent
          v-else-if="id === AGENT"
          v-model="asked"
          :turns="turns"
          :working="working"
          :placeholder="words.ask"
          @submit="send"
        >
          <template #failure="{ turn }">
            {{ turn.voice === 'asked' ? words.unsent : words.stopped }}
          </template>
        </Agent>

        <div v-else-if="notes.all().includes(id)" class="note">
          <p v-if="notes.saying(id)" role="alert" class="warning">{{ notes.saying(id) }}</p>

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
  background: light-dark(#fff4e5, #3a2e1c);
  color: light-dark(#7a4b00, #f0c890);
  box-shadow: 0 6px 24px light-dark(#00000022, #00000066);
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
  background: light-dark(#fff4e5, #3a2e1c);
  color: light-dark(#7a4b00, #f0c890);
}

.failure {
  color: #b3261e;
  opacity: 1;
  max-width: 40rem;
  text-align: center;
}
</style>
