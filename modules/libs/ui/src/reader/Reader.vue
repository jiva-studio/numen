<script setup lang="ts">
/**
 * A document read: its pages in a row, scrolled through left to right.
 *
 * This is the room and the hand in it — how far the row is scrolled, which
 * pages that puts in view, and how wide they are asked for. Where a page stands
 * is `strip.ts`, one page is `Sheet.vue`, and what turns and zooms it is
 * `Controls.vue`.
 *
 * It fills whatever it is put in, and says nothing about where that is.
 */
import { computed, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue'
import Controls from './Controls.vue'
import Sheet from './Sheet.vue'
import {
  GAP,
  drawn,
  inFront,
  row,
  standAt,
  within,
  type Sheet as Paper,
} from './strip'

/** Where something sits on the page, in fractions of it. */
interface Lit {
  readonly minX: number
  readonly minY: number
  readonly maxX: number
  readonly maxY: number
}

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
    /** What is lit on one page, in fractions of it. */
    lit?: (page: number) => readonly Lit[]
    /** What turning back a page is called, and turning on. */
    back?: string
    next?: string
    /** What the field the page is typed in is called. */
    page?: string
    /** What drawing the page larger is called, and smaller. */
    closer?: string
    further?: string
    /** What is said where a page would not come. */
    undrawn?: string
  }>(),
  {
    pages: 0,
    sheets: () => [],
    at: 0,
    picture: () => '',
    lit: () => [],
    back: 'Previous page',
    next: 'Next page',
    page: 'Page',
    closer: 'Closer',
    further: 'Further',
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

/** The room the pages are read in, in CSS pixels. */
const room = ref({ wide: 0, high: 0 })
/** How far the row has been scrolled, in CSS pixels. */
const along = ref(0)

const laid = computed(() => row(props.sheets, props.pages, room.value, zoom.value))
const shown = computed(() => within(laid.value, room.value, along.value))
const middle = computed(() => inFront(laid.value, room.value, along.value))

/**
 * The widths a page is asked for. Dragging the edge of a pane crosses a few of
 * them, and a page drawn wider than its box is drawn down into it.
 */
const STAGE = 128
const staged = (pixels: number) => Math.ceil(pixels / STAGE) * STAGE

/**
 * What a page is asked for at: the widest page there is, so one width serves
 * the whole document and turning a page is not a new drawing of everything.
 */
const asking = computed(() => {
  const widest = laid.value.widths.reduce((most, wide) => Math.max(most, wide), 0)
  return widest > 0 ? staged(widest * devicePixelRatio) : 0
})

/**
 * How long the width has to have stood still before a page is asked for at it.
 * A pane edge dragged across a screen crosses a dozen widths, and each one is a
 * page drawn and thrown away.
 */
const SETTLED = 150
let settling: ReturnType<typeof setTimeout> | undefined

/** What each page is asked for at, which follows the width once it has settled. */
const drawnAt = ref(0)

// The first width is asked for at once: a document opening has nothing drawn
// and nothing to wait for.
watch(asking, (pixels, before) => {
  if (!pixels) return
  if (!before) {
    drawnAt.value = pixels
    emit('wide', pixels)
    return
  }
  clearTimeout(settling)
  settling = setTimeout(() => {
    drawnAt.value = pixels
    emit('wide', pixels)
  }, SETTLED)
})

/** Where one page is drawn, once a width has been settled on. */
const drawing = (page: number) => (drawnAt.value > 0 ? props.picture(page) : '')

const area = useTemplateRef<HTMLElement>('area')
let watching: ResizeObserver | undefined

const measure = () => {
  if (!area.value) return
  room.value = { wide: area.value.clientWidth, high: area.value.clientHeight }
}

onMounted(() => {
  measure()
  if (!area.value || typeof ResizeObserver === 'undefined') return
  watching = new ResizeObserver(measure)
  watching.observe(area.value)
})

onBeforeUnmount(() => {
  watching?.disconnect()
  clearTimeout(settling)
})

/** The hand moved the row, so the page in front is whichever is under it. */
const scrolled = () => {
  if (!area.value) return
  along.value = area.value.scrollLeft
  if (middle.value !== props.at) emit('go', middle.value)
}

/** The row put where a page stands, with that page against the left edge. */
const stand = (page: number, how: ScrollBehavior) => {
  const begins = standAt(laid.value, page)
  if (!area.value || begins === undefined) return
  area.value.scrollTo({ left: begins, behavior: how })
}

// A page turned to from outside — a search hit, the agent, the field — is
// scrolled to. One reached by the hand is already there.
watch(
  () => props.at,
  (page) => {
    if (page !== middle.value) stand(page, 'smooth')
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

const drawTo = (how: number) => {
  zoom.value = drawn(zoom.value, how)
}

/** One page's box in the row. */
const boxOf = (page: number) => ({
  insetInlineStart: `${laid.value.starts[page] ?? 0}px`,
  insetBlockStart: `${GAP}px`,
  inlineSize: `${laid.value.widths[page] ?? 0}px`,
  blockSize: `${laid.value.high}px`,
})

defineExpose({
  /**
   * Take the room again. A reader drawn out of sight has none, and the caller
   * says when it is on screen.
   */
  measure,
})
</script>

<template>
  <div class="reader numen relative h-full min-h-0 font-sans text-base text-ink">
    <div
      ref="area"
      class="reader__room h-full overflow-auto overscroll-x-contain"
      @scroll.passive="scrolled"
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
          :lit="lit(page)"
          :page="props.page"
          :undrawn="undrawn"
          :style="boxOf(page)"
        />
      </div>
      <p v-else class="reader__silence grid h-full place-items-center text-small text-hushed">
        <slot name="silence" />
      </p>
    </div>

    <Controls
      v-if="pages > 0"
      :pages="pages"
      :at="at"
      :zoom="zoom"
      :back="back"
      :next="next"
      :page="props.page"
      :closer="closer"
      :further="further"
      @go="emit('go', $event)"
      @draw="drawTo"
    />
  </div>
</template>
