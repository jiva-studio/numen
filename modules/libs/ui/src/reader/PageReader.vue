<script setup lang="ts">
/**
 * A document read: its pages in a row, scrolled through left to right.
 *
 * The row is laid out against the viewport it is read in, and the page in front
 * is the one under the middle of that viewport. It fills whatever it is put in,
 * and says nothing about where that is.
 */
import { computed, useTemplateRef, ref, watch } from 'vue'
import ReaderToolbar from './ReaderToolbar.vue'
import Sheet from './Sheet.vue'
import { usePageWidth } from './width'
import { useViewport } from './viewport'
import { useHandScroll } from './scroll'
import {
  GAP,
  READER_WORDS,
  inFront,
  row,
  standAt,
  within,
  type Rect,
  type ReaderWords,
  type Sheet as Paper,
} from './strip'

const props = withDefaults(
  defineProps<{
    /** How many pages the document has. */
    pages?: number
    /** How big each page is, in its own units. */
    sheets?: readonly Paper[]
    /** Which page is in front, counted from the first. */
    at?: number
    /** Where one page is drawn, as an address to point a picture at. */
    picture?: (page: number) => string
    /** What is highlighted on one page, in fractions of it. */
    highlights?: (page: number) => readonly Rect[]
    /** The other places on one page, each of them somewhere else to look. */
    also?: (page: number) => readonly Rect[]
    /** The words it is read with. */
    words?: ReaderWords
    /** What is said where a page would not come. */
    undrawn?: string
  }>(),
  {
    pages: 0,
    sheets: () => [],
    at: 0,
    picture: () => '',
    highlights: () => [],
    also: () => [],
    words: () => READER_WORDS,
    undrawn: 'This page would not come.',
  },
)

const emit = defineEmits<{
  /** The page to turn to, counted from the first. */
  (event: 'go', page: number): void
  /** A page is wanted this many device pixels across. */
  (event: 'wide', pixels: number): void
}>()

/** How close a page is drawn. What that may be is `strip.ts`. */
const zoom = ref(1)

const area = useTemplateRef<HTMLElement>('area')

/** The viewport the pages are read in, taken again whenever it changes. */
const { viewport, measure } = useViewport(area)

/** Where the row stands, and what a hand or a wheel does to it. */
const { along, dragging, whereabouts, send, stands, took, pulled, letGo, turned } =
  useHandScroll(area)

const laid = computed(() => row(props.sheets, props.pages, viewport.value, zoom.value))
const shown = computed(() => within(laid.value, viewport.value, along.value))
const middle = computed(() => inFront(laid.value, viewport.value, along.value))

/** What each page is asked for at. */
const { drawnAt } = usePageWidth(
  () => laid.value,
  (pixels) => emit('wide', pixels),
)

/** Where one page is drawn, once a width has been settled on. */
const drawing = (page: number) => (drawnAt.value > 0 ? props.picture(page) : '')

/**
 * The row moved, so the page in front is whichever is under the viewport now. A
 * row still on its way to where it was sent says nothing: the pages it passes
 * over are pages nobody turned to.
 */
const scrolled = () => {
  if (!stands()) return
  if (middle.value !== props.at) emit('go', middle.value)
}

/** The row put where a page stands, with that page against the left edge. */
const stand = (page: number, how: ScrollBehavior) => {
  const begins = standAt(laid.value, page)
  if (begins === undefined) return

  // As far as the row goes: the last page cannot be brought any further left
  // than the end of it.
  const furthest = Math.max(laid.value.length - viewport.value.wide, 0)
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
  () => laid.value.high,
  () => {
    requestAnimationFrame(() => stand(props.at, 'auto'))
  },
)

/** One page's box in the row. */
const boxOf = (page: number) => ({
  insetInlineStart: `${laid.value.starts[page] ?? 0}px`,
  insetBlockStart: `${GAP}px`,
  inlineSize: `${laid.value.widths[page] ?? 0}px`,
  blockSize: `${laid.value.high}px`,
})

defineExpose({
  /**
   * Take the viewport again. A reader drawn out of sight has none, and the
   * caller says when it is on screen.
   */
  measure,
})
</script>

<template>
  <div class="reader numen relative h-full min-h-0 font-sans text-base text-ink">
    <div
      ref="area"
      class="reader__viewport h-full overflow-auto overscroll-x-contain"
      :class="dragging ? 'reader__viewport--held' : 'reader__viewport--takeable'"
      tabindex="0"
      role="region"
      :aria-label="words.pages"
      @scroll.passive="scrolled"
      @pointerdown="took"
      @pointermove="pulled"
      @pointerup="letGo"
      @pointercancel="letGo"
      @wheel="turned"
    >
      <div
        v-if="pages > 0"
        class="reader__row relative"
        :style="{ inlineSize: `${laid.length}px`, blockSize: `${laid.high + 2 * GAP}px` }"
      >
        <Sheet
          v-for="page in shown"
          :key="page"
          :at="page"
          :picture="drawing(page)"
          :highlights="highlights(page)"
          :also="also(page)"
          :page="words.page"
          :undrawn="undrawn"
          :style="boxOf(page)"
        />
      </div>
      <p v-else class="grid h-full place-items-center text-small text-hushed">
        <slot name="silence" />
      </p>
    </div>

    <ReaderToolbar
      v-if="pages > 0"
      v-model:zoom="zoom"
      :pages="pages"
      :at="at"
      :words="words"
      @update:at="emit('go', $event)"
      @back="emit('go', at - 1)"
      @next="emit('go', at + 1)"
    />
  </div>
</template>

<style scoped>
/* The row scrolls under the arrows, so the keyboard is drawn where it stands.
   Inside, because the row fills the reader to its edges. */
.reader__viewport:focus-visible {
  outline: none;
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
