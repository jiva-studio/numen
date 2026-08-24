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
import { computed, onMounted, onUnmounted } from 'vue'
import { Notices, Palette, Workspace } from '@numen/ui'
import type { Notice } from '@numen/ui'
import '@numen/ui/styles.css'
import { core, documents } from './vault'
import { showing } from './showing'
import { standing } from './standing'
import { reading } from './reading'
import { cornerOf } from './corner'
import { editing } from './editing'
import { drawn } from './drawn'
import { creating } from './creating'
import { finding } from './finding'
import { leaving } from './leaving'
import { raising } from './raising'
import { windowing } from './windowing'
import BlankTab from './BlankTab.vue'
import Leaving from './Leaving.vue'
import { agentKind, talking } from './agent/kind'
import { documentKind, documenting } from './document/kind'
import { noting } from './note/kind'
import { plexKind, plexing } from './plex/kind'
import { core as agent } from './agent'
import { conversation } from './conversation'
import { WORDS as talk } from './agent/words'
import { WORDS as words } from './words'
import { AGENT, CONVERSATION, PLEX, named, opening } from './workspace'

const drawings = drawn()
const notes = editing(core, undefined, drawings.arrived)
const making = creating(core)
const window = showing(
  core,
  undefined,
  async (paths, renamed) => {
    notes.changed(paths, renamed)
    await plexes.again()
  },
  drawings.told,
  (path) => plexes.travel(path),
  (path, runs) => void read.opensAt(path, ...runs),
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
/** What could not be made or joined, in words a person reads. */
const unmade = computed(() => making.said.value)
const { chunks, embedding, tasks } = window

/** Everything running behind the window, as the corner draws it. */
const notices = computed<readonly Notice[]>(() =>
  cornerOf(tasks.value, { chunks: chunks.value, embedding: embedding.value }, words),
)

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
  opening: () => window.opening.value,
  first: () => window.first(),
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
 * Somewhere the palette was asked to go. A name is a thing and travels in the
 * plex the person is looking at; a heading and a passage are places in a note,
 * and open it where they stand. A passage from a source that is not a note
 * opens that source where it stands.
 */
const went = async (item: string, action: string) => {
  const landing = palette.chose(item, action)
  palette.shows(false)
  if (!landing) return

  if (landing.at === 'plex') {
    void plexes.travel(landing.path)
    return
  }
  if (landing.at === 'document') {
    await read.opensAt(landing.path, { start: landing.start ?? 0, length: landing.length ?? 0 })
    return
  }
  noted.shows(landing.path, landing.title || landing.path)
  if (landing.line !== undefined) noted.entersAt(landing.path, landing.line)
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

    <Leaving :questions="going.questions.value" :called="noted.called" />

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
