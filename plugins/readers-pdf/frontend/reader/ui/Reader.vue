<script setup lang="ts">
/**
 * A document: its pages in a horizontal row, scrolled left to right.
 *
 * The row is laid out against the viewport, and the current page is the one
 * under the viewport's midpoint. The component fills its container and does not
 * depend on the container's position.
 */
import { computed, useTemplateRef, ref } from 'vue'
import { ReaderToolbar } from './reader-toolbar'
import { Sheet } from './sheet'
import { usePageWidth } from '../model/width'
import { useViewport } from '@numen/ui'
import { onNextFrame } from '@numen/ui'
import { usePanScroll } from '../model/scroll'
import { useRowLayout } from '../model/useRowLayout'
import { usePageImageUrl } from '../model/usePageImageUrl'
import { useRowRestore } from '../model/useRowRestore'
import { READER_WORDS, getFitZoom, getTurnTarget } from '../lib/strip'
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

defineExpose({
  measure: () => measure(),
  handleKey: (event: KeyboardEvent) => onKey(event),
  /** Focuses the pages so key events reach them. */
  focusPages: () => area.value?.focus(),
})

/* ---------------------------------- State --------------------------------- */
/** The zoom the person chose with the toolbar. Until one is chosen the document is drawn at its fit. */
const chosenZoom = ref<number | null>(null)

const area = useTemplateRef<HTMLElement>('area')

/** The viewport the pages are read in, taken again whenever it changes. */
const { viewport, measure: measureViewport } = useViewport(area)

/** Row scroll state, and pointer and wheel handling. */
const scroll = usePanScroll(area)
const scrollOffset = scroll.state.offset
const { isDragging } = scroll.state
const { onPointerDown, onPointerMove, letGo } = scroll.drag
const { onWheel } = scroll

/** How close a page is drawn. What that may be is `strip.ts`. */
const zoom = computed({
  get: () => chosenZoom.value ?? getFitZoom(props.fit, props.pages, viewport.value),
  set: (value: number) => {
    chosenZoom.value = value
  },
})

const { pageLayout, visiblePages, sharpPages, middle, rowStyle, getBoxStyle } = useRowLayout({
  getPages: () => props.pages,
  viewport,
  zoom,
  offset: scrollOffset,
})

/** Width at which each page is requested. */
const { drawnWidth } = usePageWidth(
  () => pageLayout.value,
  (pixels) => emit('measure', pixels),
)

const { measure, isPutting } = useRowRestore({
  area,
  viewport,
  measureViewport,
  pageLayout,
  getAt: () => props.at,
  getPaint: () => props.paint,
  scroll: scroll.moves,
})

const getImageUrl = usePageImageUrl({ getPicture: () => props.picture, drawnWidth, sharpPages })

/* -------------------------------- Handlers -------------------------------- */
const onScroll = () => {
  if (isPutting() || !scroll.moves.isStill()) return
  if (middle.value !== props.at) emit('go', middle.value)
}

const onKey = (event: KeyboardEvent): boolean => {
  const to = getTurnTarget(event.key, props.at, props.pages.length)
  if (to === undefined) return false
  emit('go', to)
  return true
}
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
