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
import { Hand, wheeled } from './hand'
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
    /** The other places on one page, each of them somewhere else to look. */
    also?: (page: number) => readonly Lit[]
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
    also: () => [],
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

/**
 * The room, taken again.
 *
 * A room with no size is not a measurement. Every tab of a pane is mounted while
 * it is out of sight, and one out of sight has no room; laying the row out on
 * nothing asks for every page again at a width nothing will ever draw at, and
 * asks for them all a second time when the tab comes back.
 */
const measure = () => {
  if (!area.value) return
  const wide = area.value.clientWidth
  const high = area.value.clientHeight
  if (wide <= 0 || high <= 0) return
  room.value = { wide, high }
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

/** Where the row was told to stand, while it is on its way there. */
let heading: number | undefined
/** How near the row has to be to count as standing there, in CSS pixels. */
const THERE = 1

/**
 * The row moved, so the page in front is whichever is under the room now.
 *
 * A row travelling to where it was told to stand says nothing until it gets
 * there. The pages it passes over on the way are pages nobody turned to, and
 * the answer to one of them is a scroll back to it.
 */
const scrolled = () => {
  if (!area.value) return
  along.value = area.value.scrollLeft
  if (heading !== undefined) {
    if (Math.abs(along.value - heading) > THERE) return
    heading = undefined
  }
  if (middle.value !== props.at) emit('go', middle.value)
}

/**
 * The row taken hold of and pulled. A book on a table is moved by putting a
 * hand on it, and a row five hundred pages long is a long way to travel by a
 * scrollbar.
 */
const hand = new Hand()
/** Whether the hand is dragging, which is what the room is drawn as. */
const dragging = ref(false)

const took = (event: PointerEvent) => {
  // The controls sit over the room and are pressed, not dragged.
  if (!area.value || event.button !== 0) return
  hand.take(
    { x: event.clientX, y: event.clientY },
    { x: area.value.scrollLeft, y: area.value.scrollTop },
  )
}

const pulled = (event: PointerEvent) => {
  if (!area.value || !hand.holding) return
  const stands = hand.to({ x: event.clientX, y: event.clientY })
  if (!stands) return
  dragging.value = true
  // The hand has the row now, wherever it was being taken.
  heading = undefined
  area.value.scrollLeft = stands.x
  area.value.scrollTop = stands.y
  follow(event)
}

/**
 * The pointer followed where it leaves the room, so a hand that runs off the
 * edge still carries the row. A pointer the window is not holding is one this
 * cannot be asked about, and the drag then lasts as long as the pointer is over
 * the room.
 */
const follow = (event: PointerEvent) => {
  if (!area.value || area.value.hasPointerCapture(event.pointerId)) return
  try {
    area.value.setPointerCapture(event.pointerId)
  } catch {
    // The row is carried by the pointer while it is over the room.
  }
}

const letGo = (event: PointerEvent) => {
  hand.release()
  dragging.value = false
  if (area.value?.hasPointerCapture(event.pointerId)) {
    area.value.releasePointerCapture(event.pointerId)
  }
}

/**
 * A wheel turned. A row at rest has one axis and a wheel turned down means the
 * next page; drawn closer the room has both, and then down means down.
 */
const turned = (event: WheelEvent) => {
  if (!area.value) return
  const hasBelow = area.value.scrollHeight > area.value.clientHeight
  const by = wheeled({ x: event.deltaX, y: event.deltaY }, hasBelow)
  if (by.x === 0 && by.y === 0) return
  event.preventDefault()
  // The wheel has the row now, wherever it was being taken.
  heading = undefined
  area.value.scrollLeft += by.x
  area.value.scrollTop += by.y
}

/** The row put where a page stands, with that page against the left edge. */
const stand = (page: number, how: ScrollBehavior) => {
  const begins = standAt(laid.value, page)
  if (!area.value || begins === undefined) return

  // As far as the row goes: the last page cannot be brought any further left
  // than the end of it.
  const furthest = Math.max(laid.value.length - room.value.wide, 0)
  const target = Math.min(Math.max(begins, 0), furthest)
  if (Math.abs(area.value.scrollLeft - target) <= THERE) return

  heading = target
  area.value.scrollTo({ left: target, behavior: how })
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
      :class="dragging ? 'reader__room--held' : 'reader__room--takeable'"
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
          :lit="lit(page)"
          :also="also(page)"
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

<style scoped>
/* The row is taken hold of and pulled, so the hand says so before it is put
   down and while it is holding. */
.reader__room--takeable {
  cursor: grab;
}

.reader__room--held {
  cursor: grabbing;
  user-select: none;
  -webkit-user-select: none;
}
</style>
