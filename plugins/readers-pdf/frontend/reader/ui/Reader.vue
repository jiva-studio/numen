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
  getAnchor,
  getAnchorScroll,
  getAcrossScroll,
  getAcrossShare,
  type Anchor,
  SHARP,
  getCurrentPage,
  row,
  getScrollTarget,
  getTurnTarget,
  getFitZoom,
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
  fit: 'page',
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** The page to turn to, counted from the first. */
  (event: 'go', page: number): void
  /** A page is wanted this many device pixels across. */
  (event: 'measure', pixels: number): void
}>()

/* ---------------------------------- State --------------------------------- */
/** The zoom the person chose with the toolbar. Until one is chosen the document is drawn at its fit. */
const chosenZoom = ref<number | null>(null)

const area = useTemplateRef<HTMLElement>('area')

/** The viewport the pages are read in, taken again whenever it changes. */
const { viewport, measure: measureViewport } = useViewport(area)

/** Row scroll state, and pointer and wheel handling. */
const {
  along: scrollOffset,
  isDragging,
  getScrollOffset,
  send,
  restore,
  isMoved,
  isStill,
  onPointerDown,
  onPointerMove,
  letGo,
  onWheel,
} = usePanScroll(area)

/** How close a page is drawn. What that may be is `strip.ts`. */
const zoom = computed({
  get: () => chosenZoom.value ?? getFitZoom(props.fit, props.pages, viewport.value),
  set: (value: number) => {
    chosenZoom.value = value
  },
})

const pageLayout = computed(() => row(props.pages, viewport.value, zoom.value))
const visiblePages = computed(() =>
  getPagesWithin(pageLayout.value, viewport.value, scrollOffset.value),
)
/** The pages that are requested at the settled width. The rest keep what they show. */
const sharpPages = computed(
  () => new Set(getPagesWithin(pageLayout.value, viewport.value, scrollOffset.value, SHARP)),
)
const middle = computed(() => getCurrentPage(pageLayout.value, viewport.value, scrollOffset.value))

/** Width at which each page is requested. */
const { drawnWidth } = usePageWidth(
  () => pageLayout.value,
  (pixels) => emit('measure', pixels),
)

const rowStyle = computed(() => ({
  inlineSize: `${pageLayout.value.length}px`,
  blockSize: `${pageLayout.value.height + 2 * GAP}px`,
}))

/**
 * How many puts of the row back wait for a frame. The browser queues a scroll event when the row
 * scrolls and delivers it with the next frame. An event queued before the row was hidden or moved
 * reaches the handler after the browser has scrolled the row to its start, and reports the start as
 * a place the person scrolled to.
 */
let pendingPuts = 0

/**
 * Puts the row back with the next frame. A row the browser moved takes no scroll event as the
 * person's until then.
 */
function putRowBack(): void {
  if (!isMoved()) {
    props.paint(restore)
    return
  }
  pendingPuts += 1
  props.paint(() => {
    restore()
    pendingPuts -= 1
  })
}

/**
 * Re-measures the viewport and puts the row back where it was. A reader
 * rendered while hidden has no viewport, and a row that was hidden or moved
 * in the document is scrolled to its start by the browser, so the caller
 * invokes this when the reader becomes visible.
 */
const measure = (): void => {
  measureViewport()
  putRowBack()
}

defineExpose({
  measure,
  handleKey: (event: KeyboardEvent) => onKey(event),
  /** Focuses the pages so key events reach them. */
  focusPages: () => area.value?.focus(),
})

/* ---------------------------------- Hooks --------------------------------- */
/** The point kept through zooms whose redraw has not run yet, so a second zoom keeps the first one's. */
let pendingZoom: { anchor: Anchor | undefined; across: number } | undefined

// A page requested from outside is scrolled to. A page reached by the user's
// scrolling is already in view.
watch(
  () => props.at,
  (page) => {
    if (page === getCurrentPage(pageLayout.value, viewport.value, getScrollOffset())) return
    // A redraw that has not run yet holds the place the row was at when the layout changed, and
    // would send the row back there once the turn has started. The turn is where the row goes.
    if (pendingZoom) pendingZoom.anchor = undefined
    stand(page, 'smooth')
  },
)

// Zooming keeps the point under the centre of the viewport where it is. Every
// page changes width, so the scroll position shifts. A row with no pages before
// it has no point to keep, and the current page is shown instead.
watch(pageLayout, (now, was) => {
  if (now.height === was.height) return
  pendingZoom ??= {
    anchor: getAnchor(was, viewport.value, getScrollOffset()),
    across: getAcrossShare(was, viewport.value, area.value?.scrollTop ?? 0),
  }
  props.paint(() => {
    const point = pendingZoom
    pendingZoom = undefined
    if (!point) return
    const to = point.anchor && getAnchorScroll(pageLayout.value, viewport.value, point.anchor)
    if (to === undefined) stand(props.at, 'auto')
    else send(to, 'auto')
    if (area.value && point.anchor) {
      area.value.scrollTop = getAcrossScroll(pageLayout.value, viewport.value, point.across)
    }
  })
})

// A viewport that is reported again was hidden or moved, and the browser
// scrolled its row to the start without a scroll event.
watch(viewport, putRowBack)

/* -------------------------------- Handlers -------------------------------- */
/**
 * The row scrolled, so the current page is the one under the viewport. While the
 * row is still animating to a target it emits nothing: pages passed on the way
 * are not navigation targets.
 */
const onScroll = () => {
  if (pendingPuts > 0 || !isStill()) return
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
/** Where one page is drawn, once a width has been settled on. A page past the margin has none. */
const getImageUrl = (page: number) =>
  drawnWidth.value > 0 && sharpPages.value.has(page) ? props.picture(page) : ''

/** Scrolls the row to show the page, so that it becomes the current page. */
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
  <div class="reader numen text-ink flex h-full min-h-0 flex-col font-sans text-base">
    <ReaderToolbar
      v-if="pages.length > 0"
      v-model:zoom="zoom"
      :page-count="pages.length"
      :at="at"
      :words="words"
      @update:at="emit('go', $event)"
    />

    <div
      ref="area"
      class="reader__viewport min-h-0 flex-1 overflow-auto overscroll-x-contain"
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
  </div>
</template>

<style scoped>
.reader__viewport:focus-visible {
  outline: none;
}

/* The row is draggable: the grab cursor shows before and while it is dragged. */
.reader__viewport--takeable {
  cursor: grab;
}

.reader__viewport--held {
  cursor: grabbing;
  user-select: none;
  -webkit-user-select: none;
}
</style>
