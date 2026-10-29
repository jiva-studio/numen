<script setup lang="ts">
/**
 * A document: its pages in a horizontal row, scrolled left to right.
 *
 * The row is laid out against the viewport, and the current page is the one
 * under the viewport's midpoint. The component fills its container and does not
 * depend on the container's position.
 */
import { computed, useTemplateRef, ref, watch } from 'vue'
import { ReaderToolbar } from './reader-toolbar'
import { Sheet } from './sheet'
import { usePageWidth } from '../model/width'
import { useViewport } from '@numen/ui'
import { onNextFrame } from '@numen/ui'
import { usePanScroll, type PanScroll } from '../model/scroll'
import {
  GAP,
  READER_WORDS,
  getPagesWithin,
  inFront,
  row,
  getScrollTarget,
  getTurnTarget,
} from '../lib/strip'
import type { ReaderProps } from './Reader.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<ReaderProps>(), {
  pages: () => [],
  at: 0,
  picture: () => '',
  highlights: () => [],
  otherHighlights: () => [],
  words: () => READER_WORDS,
  undrawn: 'Failed to load page',
  paint: onNextFrame,
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** The page to turn to, counted from the first. */
  (event: 'go', page: number): void
  /** A page is wanted this many device pixels across. */
  (event: 'measure', pixels: number): void
}>()

/* ---------------------------------- State --------------------------------- */
/** How close a page is drawn. What that may be is `strip.ts`. */
const zoom = ref(1)

const area = useTemplateRef<HTMLElement>('area')

/** The viewport the pages are read in, taken again whenever it changes. */
const { viewport, measure } = useViewport(area)

/** Row scroll state, and pointer and wheel handling. */
const {
  along: scrollOffset,
  isDragging,
  getScrollOffset,
  send,
  isStill,
  onPointerDown,
  onPointerMove,
  letGo,
  onWheel,
} = usePanScroll(area)

const pageLayout = computed(() => row(props.pages, viewport.value, zoom.value))
const visiblePages = computed(() =>
  getPagesWithin(pageLayout.value, viewport.value, scrollOffset.value),
)
const middle = computed(() => inFront(pageLayout.value, viewport.value, scrollOffset.value))

/** Width at which each page is requested. */
const { drawnWidth } = usePageWidth(
  () => pageLayout.value,
  (pixels) => emit('measure', pixels),
)

const rowStyle = computed(() => ({
  inlineSize: `${pageLayout.value.length}px`,
  blockSize: `${pageLayout.value.height + 2 * GAP}px`,
}))

defineExpose({
  /**
    * Re-measures the viewport. A reader rendered while hidden has none, so the
    * caller invokes this when the reader becomes visible.
   */
  measure,
  handleKey: (event: KeyboardEvent) => onKey(event),
  /** Focuses the pages so key events reach them. */
  focusPages: () => area.value?.focus(),
})

/* ---------------------------------- Hooks --------------------------------- */
// A page requested from outside is scrolled to. A page reached by the user's
// scrolling is already in view.
watch(
  () => props.at,
  (page) => {
    if (page !== inFront(pageLayout.value, viewport.value, getScrollOffset())) stand(page, 'smooth')
  },
)

// Zooming keeps the current page in view. Every page changes width, so the
// scroll position along the row shifts.
watch(
  () => pageLayout.value.height,
  () => {
    props.paint(() => stand(props.at, 'auto'))
  },
)

/* -------------------------------- Handlers -------------------------------- */
/**
 * The row scrolled, so the current page is the one under the viewport. While the
 * row is still animating to a target it emits nothing: pages passed on the way
 * are not navigation targets.
 */
const onScroll = () => {
  if (!isStill()) return
  if (middle.value !== props.at) emit('go', middle.value)
}

/**
 * A key handled by the tab: returns true when it turned the page. The target
 * page is emitted, so the owner of `at` performs the move.
 */
const onKey = (event: KeyboardEvent): boolean => {
  const to = getTurnTarget(event.key, props.at, props.pages.length)
  if (to === undefined) return false
  emit('go', to)
  return true
}

/* --------------------------------- Helpers -------------------------------- */
/** Where one page is drawn, once a width has been settled on. */
const getImageUrl = (page: number) => (drawnWidth.value > 0 ? props.picture(page) : '')

/** Scrolls the row so the page sits at the left edge. */
const stand = (page: number, how: Parameters<PanScroll['send']>[1]) => {
  const to = getScrollTarget(pageLayout.value, viewport.value, page)
  if (to !== undefined) send(to, how)
}

/** One page's box in the row. */
const getBoxStyle = (page: number) => ({
  insetInlineStart: `${pageLayout.value.starts[page] ?? 0}px`,
  insetBlockStart: `${GAP}px`,
  inlineSize: `${pageLayout.value.widths[page] ?? 0}px`,
  blockSize: `${pageLayout.value.height}px`,
})
</script>

<template>
  <div class="reader numen text-ink relative h-full min-h-0 font-sans text-base">
    <div
      ref="area"
      class="reader__viewport h-full overflow-auto overscroll-x-contain"
      :class="isDragging ? 'reader__viewport--held' : 'reader__viewport--takeable'"
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
          v-for="page in visiblePages"
          :key="page"
          :at="page"
          :picture="getImageUrl(page)"
          :highlights="highlights(page)"
          :other-highlights="otherHighlights(page)"
          :page="words.page"
          :undrawn="undrawn"
          :style="getBoxStyle(page)"
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
/* The row scrolls under the arrows, so the keyboard is drawn at its current position.
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
