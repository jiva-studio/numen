<script setup lang="ts">
/**
 * A book made for a screen, read: its text set in columns and turned a page at
 * a time.
 *
 * spread of two turns as one. Where a person is reading is a byte offset into
 * the book's text: setting the text larger sets the columns again, and the
 * offset stays where it was.
 */
import { useTemplateRef } from 'vue'

import { getPageNavigationAction } from '@/shared/lib/dom/pageNavigation'
import { createBookHand } from '../model/hand'
import { useBookLayout } from '../model/layout'
import { createBookLinks } from '../model/links'
import { BOOK_WORDS } from '../lib/words'
import type { BookProps } from '../lib/props'
import BookFoot from './BookFoot.vue'
import BookHead from './BookHead.vue'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<BookProps>(), {
  markup: '',
  path: '',
  span: () => ({ from: 0, to: 0 }),
  book: () => ({ from: 0, to: 0 }),
  at: 0,
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

/* --------------------------------- State ---------------------------------- */
const area = useTemplateRef<HTMLElement>('area')
const paper = useTemplateRef<HTMLElement>('paper')

/** The near edge of a box, which every place across the columns is read from. */
const getEdge = (of: HTMLElement) => of.getBoundingClientRect().left

const links = createBookLinks(props, {
  reportMove: (pos) => emit('move', pos),
  reportFollow: (path) => emit('follow', path),
})

const layout = useBookLayout({
  area,
  paper,
  props,
  onMove: (pos) => emit('move', pos),
  takeLed: links.takeLed,
  getEdge,
})
const { isMeasured, isStood, setting, spreadCount, front, leftInChapter } = layout

const hand = createBookHand({
  area,
  paper,
  turn: layout.turn,
  getEdge,
})

/* -------------------------------- Handlers -------------------------------- */
const onFollow = (press: MouseEvent) => links.follow(press, layout.findFragmentOffset)

/**
 * A key the tab caught. Turning belongs to whatever holds the book, so the
 * page it turns is answered for and the key is not listened for here.
 */
const onKey = (event: KeyboardEvent): boolean => {
  const way = getPageNavigationAction(event.key)
  if (!way) return false
  layout.turn(way)
  return true
}

/* -------------------------------- Helpers --------------------------------- */
defineExpose({
  /**
   * Set the text again. A reader drawn out of sight has no reading area, and
   * the caller says when it is on screen.
   */
  measure: () => layout.settle(layout.getKeptOffset()),
  /** A key the tab caught: true where it turned the page. */
  handleKey: onKey,
})
</script>

<template>
  <div class="book numen text-ink relative h-full min-h-0 font-sans text-base">
    <BookHead :chapter="chapter" />

    <div class="book__margin h-full">
      <div
        ref="area"
        class="book__area h-full overflow-hidden"
        role="region"
        :aria-label="words.pages"
        @pointerdown="hand.takeDown"
        @pointerup="hand.letGo"
      >
        <!-- The markup reaches this component already measured against what may
             be drawn. -->
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div
          v-show="isMeasured"
          ref="paper"
          class="book__paper prose prose-sm prose-numen max-w-none"
          :class="{ 'book__paper--unset': !isStood }"
          :style="setting"
          @click="onFollow"
          @auxclick="onFollow"
          v-html="markup"
        />
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
