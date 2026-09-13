<script setup lang="ts">
/**
 * A document read: its pages in a row, scrolled through left to right.
 *
 * The row is laid out against the viewport it is read in, and the page in front
 * is the one under the middle of that viewport. It fills whatever it is put in,
 * and says nothing about where that is.
 */
import { computed, useTemplateRef, ref, watch } from 'vue'
import { ReaderToolbar } from './reader-toolbar'
import { Sheet } from './sheet'
import { usePageWidth } from '../model/width'
import { useViewport } from '@/shared/lib/viewport'
import { keyTurn } from '@/shared/lib/turn'
import { useHandScroll } from '../model/scroll'
import {
  GAP,
  READER_WORDS,
  getPagesWithin,
  inFront,
  row,
  standAt,
  type Rect,
  type ReaderWords,
  type Page,
} from '../lib/strip'

const props = withDefaults(
  defineProps<{
    /** How big each page is, in its own units. */
    pages?: readonly Page[]
    /** Which page is in front, counted from the first. */
    at?: number
    /** Where one page is drawn, as an address to point a picture at. */
    picture?: (page: number) => string
    /** What is highlighted on one page, in fractions of it. */
    highlights?: (page: number) => readonly Rect[]
    /** The other places named on one page, apart from the one opened at. */
    otherHighlights?: (page: number) => readonly Rect[]
    /** The words it is read with. */
    words?: ReaderWords
    /** What is said where a page would not come. */
    undrawn?: string
  }>(),
  {
    pages: () => [],
    at: 0,
    picture: () => '',
    highlights: () => [],
    otherHighlights: () => [],
    words: () => READER_WORDS,
    undrawn: 'This page would not come.',
  },
)

const emit = defineEmits<{
  /** The page to turn to, counted from the first. */
  (event: 'go', page: number): void
  /** A page is wanted this many device pixels across. */
  (event: 'measure', pixels: number): void
}>()

/** How close a page is drawn. What that may be is `strip.ts`. */
const zoom = ref(1)

const area = useTemplateRef<HTMLElement>('area')

/** The viewport the pages are read in, taken again whenever it changes. */
const { viewport, measure } = useViewport(area)

/** Where the row stands, and what a hand or a wheel does to it. */
const {
  along,
  dragging,
  whereabouts,
  send,
  isStill,
  onPointerDown,
  onPointerMove,
  letGo,
  onWheel,
} = useHandScroll(area)

const laid = computed(() => row(props.pages, viewport.value, zoom.value))
const shown = computed(() => getPagesWithin(laid.value, viewport.value, along.value))
const middle = computed(() => inFront(laid.value, viewport.value, along.value))

/** What each page is asked for at. */
const { drawnWidth } = usePageWidth(
  () => laid.value,
  (pixels) => emit('measure', pixels),
)

/** Where one page is drawn, once a width has been settled on. */
const drawing = (page: number) => (drawnWidth.value > 0 ? props.picture(page) : '')

/**
 * The row moved, so the page in front is whichever is under the viewport now. A
 * row still on its way to where it was sent says nothing: the pages it passes
 * over are pages nobody turned to.
 */
const onScroll = () => {
  if (!isStill()) return
  if (middle.value !== props.at) emit('go', middle.value)
}

/** The row put where a page stands, with that page against the left edge. */
const stand = (page: number, how: ScrollBehavior) => {
  const begins = standAt(laid.value, page)
  if (begins === undefined) return

  // As far as the row goes: the last page cannot be brought any further left
  // than the end of it.
  const furthest = Math.max(laid.value.length - viewport.value.width, 0)
  send(Math.min(Math.max(begins, 0), furthest), how)
}

// A page asked for from outside is scrolled to. One reached by the hand is
// already there.
watch(
  () => props.at,
  (page) => {
    if (page !== inFront(laid.value, viewport.value, whereabouts())) stand(page, 'smooth')
  },
)

// Zooming keeps the page in front in front. Every page changes width, so the
// place along the row the hand was looking at has moved.
watch(
  () => laid.value.height,
  () => {
    requestAnimationFrame(() => stand(props.at, 'auto'))
  },
)

/** One page's box in the row. */
const boxOf = (page: number) => ({
  insetInlineStart: `${laid.value.starts[page] ?? 0}px`,
  insetBlockStart: `${GAP}px`,
  inlineSize: `${laid.value.widths[page] ?? 0}px`,
  blockSize: `${laid.value.height}px`,
})

const rowStyle = computed(() => ({
  inlineSize: `${laid.value.length}px`,
  blockSize: `${laid.value.height + 2 * GAP}px`,
}))

/**
 * A key the tab caught: true where it turned the page. The page asked for is
 * emitted, so whoever holds `at` is the one that moves it.
 */
const handleKey = (event: KeyboardEvent): boolean => {
  const way = keyTurn(event.key)
  if (!way || props.pages.length === 0) return false

  const last = props.pages.length - 1
  const to = way === 'first' ? 0 : way === 'last' ? last : props.at + (way === 'next' ? 1 : -1)

  if (to < 0 || to > last) return false
  emit('go', to)
  return true
}

defineExpose({
  /**
   * Take the viewport again. A reader drawn out of sight has none, and the
   * caller says when it is on screen.
   */
  measure,
  handleKey,
  /** The pages take the keyboard, so that a key struck reaches them. */
  focusPages: () => area.value?.focus(),
})
</script>

<template>
  <div class="reader numen text-ink relative h-full min-h-0 font-sans text-base">
    <div
      ref="area"
      class="reader__viewport h-full overflow-auto overscroll-x-contain"
      :class="dragging ? 'reader__viewport--held' : 'reader__viewport--takeable'"
      tabindex="0"
      role="region"
      :aria-label="words.pages"
      @scroll.passive="onScroll"
      @pointerdown="onPointerDown"
      @pointermove="onPointerMove"
      @pointerup="letGo"
      @pointercancel="letGo"
      @wheel="onWheel"
    >
      <div v-if="pages.length > 0" class="reader__row relative" :style="rowStyle">
        <Sheet
          v-for="page in shown"
          :key="page"
          :at="page"
          :picture="drawing(page)"
          :highlights="highlights(page)"
          :other-highlights="otherHighlights(page)"
          :page="words.page"
          :undrawn="undrawn"
          :style="boxOf(page)"
        />
      </div>
      <p v-else class="text-small text-hushed grid h-full place-items-center">
        <slot name="silence" />
      </p>
    </div>

    <ReaderToolbar
      v-if="pages.length > 0"
      v-model:zoom="zoom"
      :page-count="pages.length"
      :at="at"
      :words="words"
      @update:at="emit('go', $event)"
    />
  </div>
</template>

<style scoped>
/* The row scrolls under the arrows, so the keyboard is drawn where it stands.
   Inside, because the row fills the reader to its edges. */
.reader__viewport:focus-visible {
  outline: none;
  box-shadow: inset 0 0 0 var(--numen-ring-width) var(--numen-ring);
}

/* The row is taken hold of and pulled, so the hand says so before it is put
   down and while it is holding. */
.reader__viewport--takeable {
  cursor: grab;
}

.reader__viewport--held {
  cursor: grabbing;
  user-select: none;
  -webkit-user-select: none;
}
</style>
