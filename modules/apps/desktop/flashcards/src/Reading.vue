<script setup lang="ts">
/**
 * The panel the notes a deck is joined to are read in.
 *
 * They are read one under another rather than picked from a list: a person who
 * came here to read is reading, and a list would make them choose first.
 */
import { nextTick, useTemplateRef, watch } from 'vue'
import { Prose } from '@numen/ui'

import { WORDS as words } from './reading/words'
import type { Neighbour } from './reading/core'
import type { Read } from './reading'

const props = defineProps<{ held: Read }>()

const column = useTemplateRef<HTMLElement>('column')

/** A note is keyed by what its article is keyed by. */
const keyOf = (one: Neighbour) => one.path || one.written

/**
 * Where each note was drawn, so the one a link named can be scrolled to. Read
 * only when a link is followed, so it is held outside the reactive graph.
 */
const drawn = new Map<string, HTMLElement>()

const holdNote = (one: Neighbour, element: unknown): void => {
  if (element) drawn.set(keyOf(one), element as HTMLElement)
  else drawn.delete(keyOf(one))
}

/** How much of the panel one press of space moves it. */
const STEP = 0.85

/** The panel scrolled by a key, because the caret is nowhere in it. */
const scrolls = (back = false) => {
  const at = column.value
  if (!at) return
  at.scrollBy({ top: at.clientHeight * STEP * (back ? -1 : 1), behavior: 'smooth' })
}

defineExpose({ scrolls })

/** A note is named by its title, and by how it was written where it has none. */
const named = (one: Neighbour) => one.title || one.written

// A link pressed in the card opens the panel on the note it names, so the
// reading starts where the person was looking. The notes are waited for: the
// panel comes in while they are still being asked for, and a note cannot be
// scrolled to before it is drawn.
watch(
  () => [props.held.open(), props.held.at.value, props.held.notes.value] as const,
  ([open, at, notes]) => {
    if (!open || !at || !notes.length) return
    void nextTick(() => {
      const found = notes.find((one) => one.written === at || one.path === at)
      if (found) drawn.get(keyOf(found))?.scrollIntoView({ block: 'start', behavior: 'auto' })
      props.held.read()
    })
  },
  { immediate: true },
)
</script>

<template>
  <section class="reading" :aria-label="words.reading">
    <div ref="column" class="reading__column">
      <p v-if="!props.held.notes.value.length && !props.held.working.value" class="reading__quiet">
        {{ words.nothing }}
      </p>

      <article
        v-for="one in props.held.notes.value"
        :key="keyOf(one)"
        :ref="(element) => holdNote(one, element)"
        class="reading__note"
      >
        <header class="reading__head">
          <h2 class="reading__name">{{ named(one) }}</h2>
          <!-- The path is here because two notes can be called the same thing. -->
          <p class="reading__quiet">
            <span v-if="one.path">{{ one.path }}</span>
            <span v-if="!one.points">{{ words.pointsHere }}</span>
            <span v-if="one.label">{{ one.label }}</span>
          </p>
        </header>

        <p v-if="!one.path" class="reading__quiet">{{ words.dangling }}</p>
        <p v-else-if="one.ambiguous" class="reading__quiet">{{ words.ambiguous }}</p>
        <p v-if="one.refusal" class="reading__quiet">{{ one.refusal }}</p>

        <!-- A link in what is read leads nowhere: this window has one page. -->
        <Prose v-if="one.body" :text="one.body" />
      </article>

      <p v-if="props.held.unread.value" class="reading__quiet">
        {{ words.named(props.held.unread.value) }}
      </p>
    </div>
  </section>
</template>

<style scoped>
/* The panel stands where the card stood and is the same thing to look at, so it
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

/* Read down, while the strip it stands in is taken across. */
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

/* The name, ruled off from the prose under it. */
.reading__head {
  margin-block-end: var(--numen-inset);
  padding-block-end: var(--numen-inset);
  border-block-end: 1px solid var(--numen-rule);
}

.reading__name {
  margin: 0;
  overflow-wrap: anywhere;
  font-size: var(--numen-title-size);
}

.reading__quiet {
  margin: 0.25rem 0 0;
  overflow-wrap: anywhere;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

/* Two things said quietly about one note stand apart on the same line. */
.reading__quiet > span + span {
  margin-inline-start: var(--numen-inset);
}
</style>
