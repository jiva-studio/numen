<script setup lang="ts">
/**
 * The panel the notes a deck is joined to are read in.
 *
 * They are read one under another: a person who came here to read is reading.
 */
import { nextTick, useTemplateRef, watch } from 'vue'
import { Prose } from '@numen/ui'

import { NoteHead } from './note-head'
import { WORDS as words } from '../lib/notesWords'
import type { Neighbour } from '../api/notes'
import type { NotesPanelState } from '../model/notes'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ panel: NotesPanelState }>()

/* --------------------------------- State ---------------------------------- */
const column = useTemplateRef<HTMLElement>('column')

/* --------------------------------- Hooks ---------------------------------- */
defineExpose({ scrollPage })

// A link pressed in the card opens the panel on the note it names, so the
// reading starts where the person was looking. The notes are waited for: the
// panel comes in while they are still being asked for, and a note cannot be
// scrolled to before it is drawn.
watch(
  () => [props.panel.open.value, props.panel.col.value, props.panel.notes.value] as const,
  ([open, col, notes]) => {
    if (!open || !col || !notes.length) return
    void nextTick(() => {
      const found = notes.find((one: Neighbour) => one.written === col || one.path === col)
      if (found)
        noteElements.get(getNoteKey(found))?.scrollIntoView({ block: 'start', behavior: 'auto' })
      props.panel.read()
    })
  },
  { immediate: true },
)

/* -------------------------------- Helpers --------------------------------- */
/** A note is keyed by what its article is keyed by. */
const getNoteKey = (one: Neighbour) => one.path || one.written

/**
 * Where each note was drawn, so the one a link named can be scrolled to. Read
 * only when a link is followed, so it is held outside the reactive graph.
 */
const noteElements = new Map<string, HTMLElement>()

const registerNoteElement = (one: Neighbour, element: unknown): void => {
  if (element) noteElements.set(getNoteKey(one), element as HTMLElement)
  else noteElements.delete(getNoteKey(one))
}

/** How much of the panel one press of space moves it. */
const STEP = 0.85

/** The panel scrolled by a key, because the caret is nowhere in it. */
function scrollPage(back = false) {
  const col = column.value
  if (!col) return
  col.scrollBy({ top: col.clientHeight * STEP * (back ? -1 : 1), behavior: 'smooth' })
}
</script>

<template>
  <section class="reading" :aria-label="words.reading">
    <div ref="column" class="reading__column">
      <p
        v-if="!props.panel.notes.value.length && !props.panel.isWorking.value"
        class="reading__quiet"
      >
        {{ words.nothing }}
      </p>

      <article
        v-for="one in props.panel.notes.value"
        :key="getNoteKey(one)"
        :ref="(element) => registerNoteElement(one, element)"
        class="reading__note"
      >
        <NoteHead :note="one" />

        <p v-if="!one.path" class="reading__quiet">{{ words.dangling }}</p>
        <p v-else-if="one.isAmbiguous" class="reading__quiet">{{ words.ambiguous }}</p>
        <p v-if="one.error" class="reading__quiet">{{ one.error }}</p>

        <!-- A link in what is read leads nowhere: this window has one page. -->
        <Prose v-if="one.body" :text="one.body" />
      </article>

      <p v-if="props.panel.unread.value" class="reading__quiet">
        {{ words.named(props.panel.unread.value) }}
      </p>
    </div>
  </section>
</template>

<style scoped>
/* The panel stands where the card stood and is the same thing to look col, so it
   takes the card's ground. */
.reading {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-inline-size: 0;
  min-block-size: 0;
  padding: var(--numen-inset-wide);
  border: 1px solid var(--numen-rule);
  border-radius: var(--numen-radius);
  overflow: hidden;
  background: var(--numen-raised);
}

.reading__column {
  flex: 1;
  min-inline-size: 0;
  min-block-size: 0;
  overflow-y: auto;
  scrollbar-width: none;
  scroll-behavior: smooth;
  overscroll-behavior-y: contain;
}

.reading__column::-webkit-scrollbar {
  display: none;
}

.reading__note + .reading__note {
  margin-block-start: calc(var(--numen-inset-wide) * 2);
}

.reading__quiet {
  margin: 0.25rem 0 0;
  overflow-wrap: anywhere;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
