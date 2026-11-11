<script setup lang="ts">
/**
 * The panel in which the notes joined to a deck are read.
 *
 * The notes are stacked vertically in one column for continuous reading.
 */
import { nextTick, useTemplateRef, watch, type ComponentPublicInstance } from 'vue'
import { Prose } from '@numen/ui'

import { NoteHead } from './note-head'
import { WORDS as words } from '../../lib/session/notesWords'
import type { Neighbor } from '../../api/session/notes'
import { getNoteKey, type NotesPanelState } from '../../model/session/notes'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ panel: NotesPanelState }>()

defineExpose({ scrollPage })

/* --------------------------------- State ---------------------------------- */
const NO_LINKS = Object.freeze([])

const column = useTemplateRef<HTMLElement>('column')

/**
 * Where each note was drawn, so the one a link named can be scrolled to. Read
 * only when a link is followed, so it is held outside the reactive graph.
 */
const noteElements = new Map<string, HTMLElement>()

/** How much of the panel one press of space moves it. */
const STEP = 0.85

/* --------------------------------- Hooks ---------------------------------- */
// A link pressed in the card opens the panel on the note it names, so reading
// starts at that note. The watcher waits for the notes: the panel opens while they
// are still being requested, and a note cannot be scrolled to before it is
// rendered.
watch(
  () =>
    [
      props.panel.view.open.value,
      props.panel.view.col.value,
      props.panel.content.notes.value,
    ] as const,
  async ([open, col, notes]) => {
    if (!open || !col || !notes.length) return
    await nextTick(() => {
      const found = notes.find((one: Neighbor) => one.written === col || one.path === col)
      if (found)
        noteElements.get(getNoteKey(found))?.scrollIntoView({ block: 'start', behavior: 'auto' })
      props.panel.actions.read()
    })
  },
  { immediate: true },
)

/* -------------------------------- Helpers --------------------------------- */
const registerNoteElement = (
  one: Neighbor,
  element: Element | ComponentPublicInstance | null,
): void => {
  if (element instanceof HTMLElement) noteElements.set(getNoteKey(one), element)
  else noteElements.delete(getNoteKey(one))
}

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
        v-if="!props.panel.content.notes.value.length && !props.panel.content.isWorking.value"
        class="reading__quiet"
      >
        {{ words.nothing }}
      </p>

      <article
        v-for="one in props.panel.content.notes.value"
        :key="getNoteKey(one)"
        :ref="(element) => registerNoteElement(one, element)"
        class="reading__note"
      >
        <NoteHead :note="one" />

        <p v-if="!one.path" class="reading__quiet">{{ words.dangling }}</p>
        <p v-else-if="one.isAmbiguous" class="reading__quiet">{{ words.ambiguous }}</p>
        <p v-if="one.error" class="reading__quiet">{{ one.error }}</p>

        <!-- A link in what is read leads nowhere: this window has one page. -->
        <Prose
          v-if="one.body"
          :text="one.body"
          :links="props.panel.content.links.value.get(getNoteKey(one))?.links ?? NO_LINKS"
          :unresolved="props.panel.content.links.value.get(getNoteKey(one))?.unresolved ?? NO_LINKS"
        />
      </article>

      <p v-if="props.panel.content.unread.value" class="reading__quiet">
        {{ words.named(props.panel.content.unread.value) }}
      </p>
    </div>
  </section>
</template>

<style scoped>
/* The panel occupies the card's place and looks the same, so it
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
  --quiet-margin: 0.25rem;
  margin: var(--quiet-margin) 0 0;
  overflow-wrap: anywhere;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
