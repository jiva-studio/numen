<script setup lang="ts">
/**
 * The window: one vault, and tabs to divide the screen between.
 *
 * Choosing a node asks for that note's neighbourhood and hands it back to the
 * plex, which travels there by itself. What decides when to ask is in
 * `showing.ts`, what a tab holds is in `holding.ts`, and what is drawn from
 * either is here.
 */
import { computed, onMounted, onUnmounted, watch } from 'vue'
import { Notices, Palette, Workspace, closeTab, openTab, openTabBeside } from '@numen/ui'
import type { Notice, PlexShowing, Tab } from '@numen/ui'
import '@numen/ui/styles.css'
import { core, documents } from './vault'
import { showing } from './showing'
import { reading, type Run } from './reading'
import { cornerOf } from './corner'
import { editing } from './editing'
import { drawn } from './drawn'
import { creating } from './creating'
import { holding, type Talk } from './holding'
import { finding } from './finding'
import { leaving } from './leaving'
import AgentTab from './agent/AgentTab.vue'
import { talking } from './agent/kind'
import DocumentTab from './document/DocumentTab.vue'
import { documenting } from './document/kind'
import NoteTab from './note/NoteTab.vue'
import { noting } from './note/kind'
import PlexTab from './plex/PlexTab.vue'
import { plexing } from './plex/kind'
import { core as agent } from './agent'
import { conversation } from './conversation'
import { WORDS as words } from './words'
import { AGENT, NOTE, PLEX, plexCalled, shortened } from './workspace'

const drawings = drawn()
const notes = editing(core, undefined, drawings.arrived)
const making = creating(core)
const window = showing(
  core,
  undefined,
  notes.changed,
  drawings.told,
  (path) => held.shows(path),
  (path, runs) => opensAt(path, ...runs),
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
const { indexing, failure, warning, trouble, unwatched, unreachable, holds, looking } = window
/** What could not be made or joined, in words a person reads. */
const unmade = computed(() => making.said.value)
const { chunks, embedding, tasks } = window

/** Everything running behind the window, as the corner draws it. */
const notices = computed<readonly Notice[]>(() =>
  cornerOf(tasks.value, { chunks: chunks.value, embedding: embedding.value }, words),
)

/** The notes the window has open: what each is called, and what each tab of one holds. */
const noted = noting(core, notes, drawings, {
  closes: (path) => {
    layout.value = closeTab(layout.value, path)
  },
})
const titles = noted.titles

/** What each tab of the window holds, and what it lets go of when it closes. */
const held = holding({
  plex: (at) =>
    plexing(window.plex(at), {
      makes: making,
      ready: () => !failure.value && !indexing.value,
      opens: (path, title, showing) => openNote(path, title, showing),
      asks: (text) => held.askAbout(text),
    }),
  talk: (name) =>
    talking(conversation(agent, words, name), {
      looking: () => looking.value,
      opens: (path, ...runs) => opensAt(path, ...runs),
    }),
  note: async () => {
    const made = await making.start()
    if (!made) return ''
    noted.opens(made.path, made.title)
    return made.path
  },
  document: (path) => documenting(reading(documents, path)),
})
const { layout, blanks } = held

/** The palette: one keystroke, and everything the words typed turn up. */
const palette = finding(core, words)

/**
 * The palette is opened and put away by one keystroke, taken on the window: it
 * belongs to no pane.
 */
const asked = (event: KeyboardEvent) => {
  // A pane that has already answered this keystroke has answered it: an editor
  // binds Ctrl-K to a cut of its own.
  if (event.defaultPrevented) return
  if (event.key.toLowerCase() !== 'k' || event.altKey || !(event.metaKey || event.ctrlKey)) return
  event.preventDefault()
  palette.shows(!palette.open.value)
}

/**
 * A document put in front of the person, opened at a stretch of its own text.
 * What stands there is lit, and the tab turns to the first page of it. The
 * places named after it are lit where they fall, each of them somewhere else to
 * look.
 */
const opensAt = (path: string, ...runs: readonly Run[]) => {
  held.reads(path)
  void held.documents.value.get(path)?.reach(...runs)
}

/**
 * Somewhere the palette was asked to go. A name is a thing and travels in the
 * plex the person is looking at; a heading and a passage are places in a note,
 * and open it where they stand. A passage from a source that is not a note
 * opens that source where it stands.
 */
const went = (item: string, action: string) => {
  const landing = palette.chose(item, action)
  palette.shows(false)
  if (!landing) return

  if (landing.at === 'plex') {
    void window.travel(landing.path)
    return
  }
  if (landing.at === 'document') {
    opensAt(landing.path, { start: landing.start ?? 0, length: landing.length ?? 0 })
    return
  }
  noted.opens(landing.path, landing.title || landing.path, landing.line ?? undefined)
  layout.value = openTab(layout.value, landing.path)
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

/** A document is called by the file it is read out of. */
const documentCalled = (path: string): string => path.split('/').pop() ?? path

/** Every tab the window holds, and what each is called. */
const tabs = computed<readonly Tab[]>(() => [
  ...[...held.plexes.value].map(
    ([id, one]): Tab => ({
      id,
      title: plexCalled(words.plex, one.view.neighbourhood.value?.focus?.title ?? ''),
    }),
  ),
  ...[...held.agents.value].map(([id, talk]): Tab => ({ id, title: agentCalled(talk) })),
  ...[...held.documents.value.keys()].map((id): Tab => ({ id, title: documentCalled(id) })),
  ...blanks.value.map((id): Tab => ({ id, title: words.newTab })),
  ...notes.all().map((path): Tab => {
    const mark = noted.marked(path)
    return { id: path, title: noted.called(path), ...(mark ? { mark } : {}) }
  }),
])

/** A note opens where the person asked for it, and the tab is shown. */
const openNote = (path: string, title: string, showing: PlexShowing = 'here') => {
  noted.opens(path, title)
  layout.value =
    showing === 'beside'
      ? openTabBeside(layout.value, path, 'right', naming)
      : openTab(layout.value, path)
}

/** What one plex tab holds, or nothing where the tab holds no plex. */
const plexIn = (id: string) => held.plexes.value.get(id) ?? null

/** The talk one agent tab holds. */
const talkIn = (id: string) => held.agents.value.get(id) ?? null

/** The document one document tab is reading. */
const documentIn = (id: string) => held.documents.value.get(id) ?? null

/** What one note tab holds, or nothing where the tab holds no note. */
const noteIn = (id: string) => (notes.all().includes(id) ? noted.held(id) : null)

/**
 * A tab is drawn while it is out of sight, where an editor and a page have no
 * room to measure. Whatever the tab now on screen holds measures again.
 */
const shown = (id: string) => {
  held.shown(id)
  noteIn(id)?.measure()
  documentIn(id)?.measure()
}

/** A tab lets go of what it held. A tab that holds a note writes what it owes. */
const shut = (id: string, hold: () => void) => {
  if (held.shut(id)) return
  const note = noteIn(id)
  if (!note) return
  hold()
  note.shuts()
}

onMounted(() => {
  globalThis.addEventListener('keydown', asked)
  void window.start()
  void going.start()
})
onUnmounted(() => {
  globalThis.removeEventListener('keydown', asked)
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
    <p v-if="warning" class="warning">{{ warning }}</p>

    <p v-if="unmade" role="alert" class="warning">{{ unmade }}</p>

    <p v-if="failure" class="failure">{{ failure }}</p>
    <p v-else-if="indexing" class="waiting">reading the vault…</p>
    <p v-else-if="!holds && trouble" class="waiting">nothing was read</p>

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
        <PlexTab v-if="plexIn(id)" :held="plexIn(id)!" />

        <AgentTab v-else-if="talkIn(id)" :held="talkIn(id)!" :unreachable="unreachable" />

        <DocumentTab v-else-if="documentIn(id)" :held="documentIn(id)!" />

        <div v-else-if="blanks.includes(id)" class="blank">
          <p class="blank__says">{{ words.choose }}</p>
          <ul class="blank__choices">
            <li v-for="one in becomes" :key="one.id">
              <button type="button" class="blank__choice" @click="void held.becomeIt(id, one.id)">
                {{ one.title }}
              </button>
            </li>
          </ul>
        </div>

        <NoteTab v-else-if="noteIn(id)" :held="noteIn(id)!" />

        <div v-else />
      </template>
    </Workspace>

    <Notices :notices="notices" :name="words.working" :put-away="words.putAway" />

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

    <Palette
      :model-value="palette.typed.value"
      :bands="palette.bands.value"
      :open="palette.open.value"
      :placeholder="words.find"
      :name="words.find"
      @update:model-value="(text: string) => void palette.typing(text)"
      @choose="went"
      @dismiss="palette.shows(false)"
    >
      <template #silence>{{ words.typeToFind }}</template>
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
  font-family: var(--numen-font-sans);
  font-size: 0.85rem;
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
  z-index: var(--numen-lift-going);
  padding: 0.8rem 1rem;
  border-radius: var(--numen-radius);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  box-shadow: var(--numen-panel-shadow);
  font-family: var(--numen-font-sans);
  font-size: 0.85rem;
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

.waiting,
.failure {
  margin: auto;
  font-family: var(--numen-font-sans);
  font-size: 0.9rem;
  opacity: 0.6;
}

/* A warning and a failure carry filesystem paths, and a long one breaks where
   it stands. */
.warning {
  margin: 0;
  padding: 0.4rem 1rem;
  font-family: var(--numen-font-sans);
  font-size: 0.8rem;
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  overflow-wrap: break-word;
}

.failure {
  color: var(--numen-alarm);
  opacity: 1;
  max-width: 40rem;
  text-align: center;
  overflow-wrap: break-word;
}
</style>
