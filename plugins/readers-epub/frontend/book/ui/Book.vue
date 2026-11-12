<script setup lang="ts">
/**
 * A book rendered for a screen: its text set in columns and turned a page at a
 * time.
 *
 * A spread of two columns turns as one. The reading position is a byte offset
 * into the book's text: enlarging the text lays out the columns again and the
 * offset is unchanged.
 */
import { useTemplateRef, type Directive } from 'vue'

import { getPageNavigationAction, measureRect } from '@numen/ui'
import { useBookPointer } from '../model/hand'
import { useBookColumns } from '../model/columns'
import { useBookLayout } from '../model/layout'
import { useBookTurns } from '../model/turns'
import { useBookLinks } from '../model/links'
import { BOOK_WORDS } from '../lib/words'
import type { BookProps } from '../lib/props'
import BookFoot from './BookFoot.vue'
import BookHead from './BookHead.vue'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<BookProps>(), {
  markup: '',
  path: '',
  span: () => ({ start: 0, end: 0 }),
  book: () => ({ start: 0, end: 0 }),
  at: 0,
  target: null,
  highlights: () => [],
  otherHighlights: () => [],
  textSize: 1,
  chapter: '',
  words: () => BOOK_WORDS,
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** The offset now in front, in bytes of the book's text. */
  (event: 'move', pos: number): void
  /**
   * A link led to another document of the book, named as the archive names it.
   * The reader lands on the place inside it once that document is drawn.
   */
  (event: 'follow', path: string): void
}>()

defineExpose({
  /**
   * Lays out the text again. A reader rendered off screen has no reading area, and
   * the caller signals when it is on screen.
   */
  measure: () => layout.settling.settle(turns.moves.getKeptOffset),
  /** A key the tab caught: true where it turned the page. */
  handleKey: (event: KeyboardEvent) => onKey(event),
})

/* --------------------------------- State ---------------------------------- */
const area = useTemplateRef<HTMLElement>('area')
const paper = useTemplateRef<HTMLElement>('paper')
const leaving = useTemplateRef<HTMLElement>('leaving')

const getEdge = (of: HTMLElement) => measureRect(of).left

const links = useBookLinks(props, {
  reportMove: (pos) => emit('move', pos),
  reportFollow: (path) => emit('follow', path),
})

const column = useBookColumns({ area, paper, props, getEdge })
const turns = useBookTurns({
  paper,
  props,
  flow: column.laid.flow,
  spreadCount: column.laid.spreadCount,
  marks: column.runMarks.marks,
  onMove: (pos) => emit('move', pos),
})
const layout = useBookLayout({
  area,
  paper,
  leaving,
  props,
  onMove: (pos) => emit('move', pos),
  takeLedLink: links.takeLedLink,
  column,
  turns,
})
const { isStood } = column
const { isMeasured, setting } = column.sizing
const { spreadCount, front, leftInChapter } = layout.position

const hand = useBookPointer({
  area,
  paper,
  turn: layout.turn,
  getEdge,
})

/** Writes markup that was checked before it reached this component. */
const vMarkup: Directive<HTMLElement, string> = {
  created: (element, binding) => {
    element.innerHTML = binding.value
  },
  beforeUpdate: (element, binding) => {
    if (binding.value !== binding.oldValue) element.innerHTML = binding.value
  },
}

/* -------------------------------- Handlers -------------------------------- */
const onArrival = () => layout.settling.settleKept()

const onFollow = (press: MouseEvent) => links.follow(press, column.runMarks.findFragmentOffset)

const onKey = (event: KeyboardEvent): boolean => {
  const way = getPageNavigationAction(event.key)
  if (!way) return false
  layout.turn(way)
  return true
}
</script>

<template>
  <div class="book numen text-ink relative h-full min-h-0 font-sans text-base">
    <BookHead :chapter="chapter" />

    <div class="book__margin h-full">
      <div
        ref="area"
        class="book__area relative h-full overflow-hidden"
        role="region"
        :aria-label="words.pages"
        @pointerdown="hand.handlePointerDown"
        @pointerup="hand.letGo"
      >
        <!-- The markup reaches this component already measured against what may
             be drawn. -->
        <div
          v-show="isMeasured"
          ref="paper"
          v-markup="markup"
          class="book__paper prose prose-sm prose-numen max-w-none"
          :class="{ 'book__paper--unset': !isStood }"
          :style="setting"
          @load.capture="onArrival"
          @error.capture="onArrival"
          @click="onFollow"
          @auxclick="onFollow"
        />
        <!-- A copy of the spread a turn into another document carries out. -->
        <div ref="leaving" class="book__leaving" inert />
      </div>
    </div>

    <!-- A book is turned by the hand and the keyboard, so the line under the
         text carries no control. -->
    <BookFoot
      :words="words"
      :front="front"
      :left-in-chapter="leftInChapter"
      :spread-count="spreadCount"
    >
      <template #way><slot name="way" /></template>
    </BookFoot>
  </div>
</template>

<style scoped src="./Book.css"></style>
