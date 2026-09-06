<script setup lang="ts">
/**
 * A book made for a screen, read: its text set in columns and turned a page at
 * a time.
 *
 * One column stands in a narrow reading area and two in a wide one, and a
 * spread of two turns as one. Where a person is reading is a byte offset into
 * the book's text: setting the text larger sets the columns again, and the
 * offset stays where it was.
 */
import { computed, onBeforeUnmount, ref, shallowRef, useTemplateRef, watch } from 'vue'
import ReaderToolbar from '@/reader/ReaderToolbar.vue'
import { useViewport } from '@/reader/viewport'
import { onNextFrame } from '@/lib/clock'
import { highlight, unhighlight } from './highlight'
import {
  BOOK_WORDS,
  GAP,
  LARGER,
  LARGEST,
  SMALLEST,
  beginsAt,
  bytesIn,
  columnWide,
  columnsIn,
  holding,
  inFront,
  keyTurn,
  pressTurn,
  spreads,
  swipeTurn,
  unitsIn,
  type Flow,
  type Mark,
  type PageTurn,
  type Span,
} from './spread'
import type { ReaderWords } from '@/reader/strip'

const props = withDefaults(
  defineProps<{
    /**
     * One document of the book, as it is drawn. Every run of text in it carries
     * `data-at`, the byte offset at which that run begins in the book's text.
     * It reaches this component already measured against what may be drawn.
     */
    markup?: string
    /** Where this document stands in the book, in bytes of the book's text. */
    span?: Span
    /** Where the book itself runs between, in bytes. */
    book?: Span
    /** The offset in front, in bytes of the book's text. */
    at?: number
    /** The runs marked where they stand, in bytes of the book's text. */
    marked?: readonly Span[]
    /** The words it is read with. */
    words?: ReaderWords
  }>(),
  {
    markup: '',
    span: () => ({ begins: 0, ends: 0 }),
    book: () => ({ begins: 0, ends: 0 }),
    at: 0,
    marked: () => [],
    words: () => BOOK_WORDS,
  },
)

const emit = defineEmits<{
  /** The offset now in front, in bytes of the book's text. */
  (event: 'go', at: number): void
}>()

/** How large the text is set. What that may be is `spread.ts`. */
const size = ref(1)

const area = useTemplateRef<HTMLElement>('area')
const paper = useTemplateRef<HTMLElement>('paper')

/** The reading area, taken again whenever it changes. */
const { viewport, measure } = useViewport(area)

/** How far the columns run, taken once they have been laid out. */
const along = ref(0)

/** Which spread is in front, counted from the first. */
const standing = ref(0)

/** Where each run of the text stands. */
const marks = shallowRef<readonly Mark[]>([])

/** Each run of the text, and the element it is set in, in the order the text is. */
let runs: readonly { readonly at: number; readonly element: HTMLElement }[] = []

const columns = computed(() => columnsIn(viewport.value.wide, size.value))

const flow = computed<Flow>(() => ({
  along: along.value,
  wide: viewport.value.wide,
  gap: GAP,
  columns: columns.value,
}))

const count = computed(() => spreads(flow.value))

/**
 * What the columns are set with. A column's own width and height are among
 * them: a percentage inside a column resolves against a box whose height is not
 * settled.
 */
const setting = computed(() => ({
  '--book-columns': String(columns.value),
  '--book-gap': `${GAP}px`,
  '--book-column': `${columnWide(flow.value)}px`,
  '--book-high': `${viewport.value.high}px`,
  '--book-size': `calc(var(--numen-prose-size) * ${size.value})`,
}))

/**
 * Where each run of the text stands, taken off the runs themselves. The first
 * of a run's rectangles is the one that counts: a run broken over a column edge
 * has one in each column, and it begins in the first.
 */
const gather = () => {
  const box = area.value
  const text = paper.value
  if (!box || !text) return

  // The runs arrive as markup, so the element holding it is what finds them.
  const origin = box.getBoundingClientRect().left - box.scrollLeft
  const found: { at: number; element: HTMLElement }[] = []
  const placed: Mark[] = []
  for (const element of text.querySelectorAll<HTMLElement>('[data-at]')) {
    const said = Number(element.dataset['at'])
    if (!Number.isFinite(said)) continue
    found.push({ at: said, element })
    const first = element.getClientRects()[0]
    if (first) placed.push({ at: said, x: first.left - origin })
  }
  found.sort((one, two) => one.at - two.at)
  placed.sort((one, two) => one.at - two.at)
  runs = found
  marks.value = placed
}

/** The spread put against the near edge of the reading area. */
const stand = (spread: number, how: ScrollBehavior) => {
  const box = area.value
  if (!box) return
  standing.value = Math.min(Math.max(spread, 0), Math.max(count.value - 1, 0))
  box.scrollTo({ left: beginsAt(flow.value, standing.value), behavior: how })
}

/** What stands in front now, said once, and nothing while nothing does. */
const said = () => {
  const now = inFront(marks.value, flow.value, standing.value)
  if (now !== undefined && now !== props.at) emit('go', now)
}

const goTo = (spread: number) => {
  stand(spread, 'smooth')
  said()
}

/** What the person is reading now, to be kept in front while the text is set again. */
const keeping = () => inFront(marks.value, flow.value, standing.value) ?? props.at

/**
 * The text set in columns again, with one offset kept in front. The columns are
 * measured after the browser has laid them out, and only an area that overflows
 * has a length to measure.
 */
const settle = (keep: number) => {
  onNextFrame(() => {
    const box = area.value
    if (!box) return
    measure()
    along.value = box.scrollWidth
    gather()
    stand(holding(marks.value, flow.value, keep), 'auto')
    marking()
  })
}

const turn = (way: PageTurn) => {
  if (way === 'first') return goTo(0)
  if (way === 'last') return goTo(count.value - 1)

  const to = standing.value + (way === 'next' ? 1 : -1)
  if (to >= 0 && to < count.value) return goTo(to)

  // Past either end of this document stands the next one, which the caller
  // hands over.
  const asked = way === 'next' ? props.span.ends : props.span.begins - 1
  if (asked >= props.book.begins && asked < props.book.ends) emit('go', asked)
}

const pressed = (event: KeyboardEvent) => {
  const way = keyTurn(event.key)
  if (!way) return
  event.preventDefault()
  turn(way)
}

/** Where the hand went down, while it is down. */
let hand: number | undefined

const took = (event: PointerEvent) => {
  hand = event.button === 0 ? event.clientX : undefined
}

/**
 * The hand lifted: the page follows a swipe, and a press near either edge turns
 * it that way. A link is followed and turns nothing.
 */
const letGo = (event: PointerEvent) => {
  const from = hand
  hand = undefined
  const box = area.value
  if (from === undefined || !box) return
  if ((event.target as HTMLElement | null)?.closest?.('a')) return

  const swipe = swipeTurn(event.clientX - from)
  if (swipe) return turn(swipe)

  const press = pressTurn(event.clientX - box.getBoundingClientRect().left, box.clientWidth)
  if (press) turn(press)
}

/**
 * Where an offset stands inside a run: the text node it falls in, and how far
 * into that node it reaches. The distance from the run's own offset is a number
 * of bytes, and only `unitsIn` turns one of those into a place in a string.
 */
const inside = (run: HTMLElement, into: number): { node: Text; offset: number } | undefined => {
  const walk = document.createTreeWalker(run, NodeFilter.SHOW_TEXT)
  let counted = 0
  let node = walk.nextNode() as Text | null
  while (node) {
    const bytes = bytesIn(node.data)
    if (counted + bytes >= into) return { node, offset: unitsIn(node.data, into - counted) }
    counted += bytes
    node = walk.nextNode() as Text | null
  }
  return undefined
}

/** The run an offset falls in: the last one beginning at or before it. */
const runAt = (at: number) => {
  let found: (typeof runs)[number] | undefined
  for (const run of runs) {
    if (run.at > at) break
    found = run
  }
  return found
}

/** One marked run, as a range over the nodes the markup carries. */
const rangeOver = (span: Span): Range | undefined => {
  const opens = runAt(span.begins)
  const closes = runAt(span.ends)
  if (!opens || !closes) return undefined
  const from = inside(opens.element, span.begins - opens.at)
  const to = inside(closes.element, span.ends - closes.at)
  if (!from || !to) return undefined
  const range = document.createRange()
  range.setStart(from.node, from.offset)
  range.setEnd(to.node, to.offset)
  return range
}

const marking = () => {
  const drawn: Range[] = []
  for (const span of props.marked) {
    const range = rangeOver(span)
    if (range) drawn.push(range)
  }
  highlight(props, drawn)
}

// A reading area of another size, or a text of another size, is another set of
// columns.
watch([() => viewport.value.wide, () => viewport.value.high, size], () => {
  settle(keeping())
})

// Another document is opened at the offset asked for, and there is nothing to
// keep in front.
watch(
  () => props.markup,
  () => {
    standing.value = 0
    settle(props.at)
  },
)

watch(() => props.marked, marking)

// An offset asked for from outside is turned to. One reached by the hand is
// already in front.
watch(
  () => props.at,
  (at) => {
    if (at < props.span.begins || at >= props.span.ends) return
    const want = holding(marks.value, flow.value, at)
    if (want !== standing.value) stand(want, 'smooth')
  },
)

onBeforeUnmount(() => {
  unhighlight(props)
})

defineExpose({
  /**
   * Set the text again. A reader drawn out of sight has no reading area, and
   * the caller says when it is on screen.
   */
  measure: () => settle(keeping()),
})
</script>

<template>
  <div class="book numen relative h-full min-h-0 font-sans text-base text-ink">
    <div class="book__margin h-full">
      <div
        ref="area"
        class="book__area h-full overflow-hidden"
        tabindex="0"
        role="region"
        :aria-label="words.pages"
        @keydown="pressed"
        @pointerdown="took"
        @pointerup="letGo"
      >
        <!-- The markup reaches this component already measured against what may
             be drawn. -->
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div
          ref="paper"
          class="book__paper prose prose-sm prose-numen max-w-none"
          :style="setting"
          v-html="markup"
        />
      </div>
    </div>

    <ReaderToolbar
      v-if="count > 0"
      v-model:zoom="size"
      :at="standing"
      :pages="count"
      :words="words"
      :least="SMALLEST"
      :most="LARGEST"
      :step="LARGER"
      @update:at="goTo"
    />
  </div>
</template>

<style scoped>
/* The clearance the text keeps from the pane, and the room the controls stand
   in below it. The area itself carries none of it: what it measures across is
   the width of a spread. */
.book__margin {
  box-sizing: border-box;
  padding-inline: var(--numen-inset-wide);
  padding-block-end: 3rem;
}

/* The area is moved by script alone: a book is turned a page at a time. */
.book__area:focus-visible {
  outline: none;
  box-shadow: inset 0 0 0 var(--numen-ring-width) var(--numen-ring);
}

/* The columns run sideways out of the area, and how far they run is read off
   it. */
.book__paper {
  block-size: 100%;
  inline-size: 100%;
  column-count: var(--book-columns);
  column-gap: var(--book-gap);
  column-fill: auto;
  font-size: var(--book-size);
  /* A word longer than the column is broken inside itself. */
  overflow-wrap: anywhere;
  hyphens: auto;
}

/* A picture is set to its column's width and no taller than the column. */
.book__paper :deep(img),
.book__paper :deep(svg),
.book__paper :deep(figure) {
  box-sizing: border-box;
  display: block;
  max-inline-size: var(--book-column);
  max-block-size: var(--book-high);
  object-fit: contain;
}

/* A table is set to the column and carries its own scrollbars where it will not
   break. */
.book__paper :deep(table),
.book__paper :deep(pre) {
  display: block;
  box-sizing: border-box;
  max-inline-size: var(--book-column);
  max-block-size: var(--book-high);
  overflow: auto;
}

/* A heading stands in the column its text does. */
.book__paper :deep(h1),
.book__paper :deep(h2),
.book__paper :deep(h3),
.book__paper :deep(h4) {
  break-after: avoid;
}

/* A marked run is drawn where it stands, over however many columns it runs. */
:global(::highlight(numen-book)) {
  background-color: var(--numen-highlight);
}
</style>
