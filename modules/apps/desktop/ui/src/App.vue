<script setup lang="ts">
/**
 * The window: one vault, and tabs to divide the screen between.
 *
 * Choosing a node asks for that note's neighbourhood and hands it back to the
 * plex, which travels there by itself. What decides when to ask is in
 * `showing.ts`; what each tab stands for is settled here and nowhere else.
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Activity, Agent, Editor, Menu, Plex, Workspace, closeTab, openTab, remainingWord } from '@numen/ui'
import type { MenuItem, Tab, WorkspaceLayout } from '@numen/ui'
import '@numen/ui/styles.css'
import { core } from './vault'
import { showing } from './showing'
import { footOf, type Phase } from './foot'
import { editing } from './editing'
import { leaving } from './leaving'
import { asPlex } from './plex'
import { core as agent } from './agent'
import { conversation } from './conversation'
import { AGENT, PLEX, TABS, opening } from './workspace'

const notes = editing(core)
const window = showing(core, undefined, undefined, notes.changed)
/** What the window answers when the application says it is going. */
const going = leaving(core)
going.holds(notes.flush)
const { neighbourhood, indexing, failure, notice, trouble, unwatched, go } = window
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
    const mark = marked(path)
    return { id: path, title: titles.get(path) ?? path, ...(mark ? { mark } : {}) }
  }),
])

/** What a note was called by the node it was opened from. */
const titles = new Map<string, string>()

const marked = (path: string): string | undefined => {
  const state = notes.shown(path).state
  return state === 'unsaved' || state === 'saving' ? 'unsaved' : state === 'stuck' ? 'stuck' : undefined
}

/** The menu on a node, and where it was asked for. */
const menu = ref<{ path: string; at: { x: number; y: number }; from: HTMLElement | SVGElement | null } | null>(null)

const items: readonly MenuItem[] = [
  { id: 'open', text: 'Open in a tab' },
  { id: 'ask', text: 'Ask the agent about this note' },
  { id: 'copy', text: 'Copy path' },
]

const askMenu = (path: string, at: { x: number; y: number }, from: HTMLElement | SVGElement | null) => {
  menu.value = { path, at, from }
}

const chose = (id: string) => {
  const asking = menu.value
  menu.value = null
  if (!asking) return
  if (id === 'open') return openNote(asking.path)
  if (id === 'copy') return void navigator.clipboard?.writeText(asking.path)
  if (id === 'ask') {
    asked.value = asking.path + ' — '
    layout.value = openTab(layout.value, AGENT)
  }
}

/** A note opens in the pane the person is in, and the tab is shown. */
const openNote = (path: string) => {
  titles.set(path, nameOf(path))
  notes.open(path)
  layout.value = openTab(layout.value, path)
}

const nameOf = (path: string): string => {
  const named = neighbourhood.value
  if (named?.focus?.path === path && named.focus.title) return named.focus.title
  const near = named?.related?.find((r) => r.note?.path === path)
  return near?.note?.title || (path.split('/').pop() ?? path).replace(/\.md$/, '')
}

/** A tab that holds a note writes what it owes before it goes. */
const shut = (id: string, hold: () => void) => {
  if (!notes.all().includes(id)) return
  hold()
  void notes.shut(id).then(() => {
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
  going.close()
  close()
})
</script>

<template>
  <main>
    <p v-if="unwatched" class="warning">not following the vault — {{ unwatched }}</p>
    <p v-if="trouble" class="warning">the vault could not be read — {{ trouble }}</p>
    <p v-if="notice" class="warning">{{ notice }}</p>

    <p v-if="notes.said" role="alert" class="warning">{{ notes.said }}</p>

    <p v-if="failure" class="failure">{{ failure }}</p>
    <p v-else-if="indexing" class="waiting">reading the vault…</p>
    <p v-else-if="!neighbourhood && trouble" class="waiting">nothing was read</p>
    <p v-else-if="!neighbourhood" class="waiting">this vault holds no notes</p>

    <Workspace v-model="layout" class="below" :tabs="tabs" @close="shut">
      <template #tab="{ id }">
        <Plex
          v-if="id === PLEX && plexed && !failure && !indexing"
          :neighbourhood="plexed!"
          :creatable="[]"
          @activate="go"
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
          <Editor
            :model-value="notes.shown(id).body"
            class="note__text"
            @update:model-value="(body: string) => notes.typed(id, body)"
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

    <Menu
      v-if="menu"
      :items="items"
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
