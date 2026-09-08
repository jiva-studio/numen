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
import { computed, onBeforeUnmount, onMounted, ref, shallowRef, useTemplateRef, watch } from 'vue'

import { useViewport } from '@/features/reader/viewport'
import { onNextFrame } from '@/shared/lib/clock'
import { ALSO, HIGHLIGHT, highlight, unhighlight } from './highlight'
import { placeIn, pointsAway, type BookLink } from './link'
import { marksIn, offsetAt, rangesOver, runsIn, type Run } from './runs'
import {
  BOOK_WORDS,
  GAP,
  LARGEST,
  SMALLEST,
  beginsAt,
  columnHigh,
  held,
  columnWide,
  columnsIn,
  handTurn,
  holding,
  inFront,
  keyTurn,
  leftInDocument,
  pagesOf,
  spreads,
  type BookWords,
  type Flow,
  type Mark,
  type PageTurn,
  type Span,
} from './spread'

const props = withDefaults(
  defineProps<{
    /**
     * One document of the book, as it is drawn. Every run of text in it carries
     * `data-offset`, the byte offset at which that run begins in the book's text.
     * It reaches this component already measured against what may be drawn.
     */
    markup?: string
    /** The document being drawn, as the book's archive names it. */
    path?: string
    /** Where this document stands in the book, in bytes of the book's text. */
    span?: Span
    /** Where the book itself runs between, in bytes. */
    book?: Span
    /** The offset in front, in bytes of the book's text. */
    at?: number
    /** The runs marked where they stand, in bytes of the book's text. */
    marked?: readonly Span[]
    /** The other runs asked about, each of them somewhere else to look. */
    also?: readonly Span[]
    /** How large the text is set, as a multiple of the size prose is read at. */
    size?: number
    /** What the book calls the place in front, drawn over the text it names. */
    chapter?: string
    /** The words it is read with. */
    words?: BookWords
  }>(),
  {
    markup: '',
    path: '',
    span: () => ({ begins: 0, ends: 0 }),
    book: () => ({ begins: 0, ends: 0 }),
    at: 0,
    marked: () => [],
    also: () => [],
    size: 1,
    chapter: '',
    words: () => BOOK_WORDS,
  },
)

const emit = defineEmits<{
  /** The offset now in front, in bytes of the book's text. */
  (event: 'go', at: number): void
  /**
   * A link led to another document of the book, named as the archive names it.
   * The reader lands on the place inside it once that document is drawn.
   */
  (event: 'follow', path: string): void
}>()

/** How large the text is set, held inside what a book may be read at. */
const size = computed(() => held(props.size, SMALLEST, LARGEST))

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

/** Each run of the text, in the order the document sets them. */
let runs: readonly Run[] = []

/**
 * Whether the reading area has been measured. A book is turned and never
 * scrolled, so its text is drawn only against an area of a known size.
 */
const measured = computed(() => viewport.value.wide > 0 && viewport.value.high > 0)

const columns = computed(() => columnsIn(viewport.value.wide, size.value))

const flow = computed<Flow>(() => ({
  along: along.value,
  wide: viewport.value.wide,
  gap: GAP,
  columns: columns.value,
}))

const count = computed(() => spreads(flow.value))

/** The page in front and how many there are, counted in columns on the screen. */
const paged = computed(() => pagesOf(props.book, props.span, flow.value, standing.value, marks.value))

/** How much of the chapter in front is still to come, which is measured exactly. */
const left = computed(() => leftInDocument(flow.value, standing.value, marks.value))

/**
 * What the columns are set with. A column's own width and height are among
 * them: a percentage inside a column resolves against a box whose height is not
 * settled.
 */
const setting = computed(() => ({
  '--book-run': columns.value === 1 ? `calc(200% + ${GAP}px)` : '100%',
  '--book-gap': `${GAP}px`,
  '--book-column': `${columnWide(flow.value)}px`,
  '--book-high': `${viewport.value.high}px`,
  '--book-size': `calc(var(--numen-prose-size) * ${size.value})`,
}))

/**
 * How tall one line of the text is set, taken off a run of the text itself. The
 * lines the column has to end between are the ones the prose is set in, and the
 * box holding it is set in another. A line height the browser will put no
 * number to leaves the column at the height of the area.
 */
const lineOf = (text: HTMLElement): number => {
  const run = text.querySelector<HTMLElement>('p') ?? text
  const said = Number.parseFloat(getComputedStyle(run).lineHeight)
  return Number.isFinite(said) ? said : 0
}

/** The runs of the drawn document, and where the columns put each of them. */
const gather = () => {
  const box = area.value
  const text = paper.value
  if (!box || !text) return

  runs = runsIn(text)
  marks.value = marksIn(runs, box.getBoundingClientRect().left - box.scrollLeft)
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

/** The offset a place named inside the drawn document stands at. */
const placeAt = (fragment: string): number | undefined => {
  const text = paper.value
  return text ? offsetAt(text, runs, fragment) : undefined
}

/**
 * The text set in columns again, with one offset kept in front. The columns are
 * measured after the browser has laid them out, and only an area that overflows
 * has a length to measure.
 *
 * A place a link led to is found here, in markup that has only now been drawn.
 */
const settle = (keep: number, led?: BookLink) => {
  measure()
  if (!measured.value) return
  onNextFrame(() => {
    const box = area.value
    const text = paper.value
    if (!box || !text) return
    text.style.setProperty('--book-paper', `${columnHigh(box.clientHeight, lineOf(text))}px`)
    // How far the columns run is asked of the box they are set in. The area
    // around it clips what overflows, and a box that clips is not asked how far
    // what it clipped reaches.
    along.value = text.scrollWidth
    gather()
    const landed =
      led && (led.path === '' || led.path === props.path) ? placeAt(led.fragment) : undefined
    stand(holding(marks.value, flow.value, landed ?? keep), 'auto')
    if (landed !== undefined) emit('go', landed)
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

/**
 * A key the tab caught. Turning belongs to whatever holds the book, so the
 * page it turns is answered for and the key is not listened for here.
 */
const pressed = (event: KeyboardEvent): boolean => {
  const way = keyTurn(event.key)
  if (!way) return false
  turn(way)
  return true
}

/** Where the hand went down, while it is down. */
let hand: number | undefined

const took = (event: PointerEvent) => {
  hand = event.button === 0 ? event.clientX : undefined
}

/** Whether words of the text stand taken up. */
const selecting = (): boolean => {
  const taken = window.getSelection()
  if (!taken || taken.isCollapsed || taken.toString().trim() === '') return false
  const text = paper.value
  return !!text && !!taken.anchorNode && text.contains(taken.anchorNode)
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

  const edge = box.getBoundingClientRect().left
  const way = handTurn(from - edge, event.clientX - edge, box.clientWidth, selecting())
  if (way) turn(way)
}

/** Where a link led, held until the document holding that place is drawn. */
let led: BookLink | undefined

/**
 * A link pressed in the text. Nothing a book contains navigates the window: a
 * link inside the book is a move within the book, and one leading out of it is
 * the window's own to hand on.
 */
const follow = (press: MouseEvent) => {
  const link = (press.target as Element | null)?.closest?.('a[href]')
  const href = link?.getAttribute('href')
  if (href === null || href === undefined) return

  press.preventDefault()
  if (pointsAway(href)) return

  const place = placeIn(href)
  if (place.path !== '' && place.path !== props.path) {
    led = place
    emit('follow', place.path)
    return
  }
  emit('go', placeAt(place.fragment) ?? props.span.begins)
}

/** The runs asked about marked where they stand, and the rest more faintly. */
const marking = () => {
  highlight(HIGHLIGHT, props, rangesOver(runs, props.marked))
  highlight(ALSO, props, rangesOver(runs, props.also))
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
    const place = led
    led = undefined
    standing.value = 0
    settle(props.at, place)
  },
)

watch([() => props.marked, () => props.also], marking)

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

onMounted(() => {
  settle(props.at)
  // The columns are counted over the type the book is set in, which arrives
  // after the markup does.
  void document.fonts?.ready.then(() => settle(keeping()))
})

onBeforeUnmount(() => {
  unhighlight(props)
})

defineExpose({
  /**
   * Set the text again. A reader drawn out of sight has no reading area, and
   * the caller says when it is on screen.
   */
  measure: () => settle(keeping()),
  /** A key the tab caught: true where it turned the page. */
  pressed,
})
</script>

<template>
  <div class="book numen relative h-full min-h-0 font-sans text-base text-ink">
    <!-- The line over the text: the way into what the book divides into, and
         what it calls the place in front. -->
    <header class="book__head text-small text-hushed">{{ chapter }}</header>

    <div class="book__margin h-full">
      <div
        ref="area"
        class="book__area h-full overflow-hidden"
        role="region"
        :aria-label="words.pages"
        @pointerdown="took"
        @pointerup="letGo"
      >
        <!-- The markup reaches this component already measured against what may
             be drawn. -->
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div
          v-show="measured"
          ref="paper"
          class="book__paper prose prose-sm prose-numen max-w-none"
          :style="setting"
          @click="follow"
          @auxclick="follow"
          v-html="markup"
        />
      </div>
    </div>

    <!-- One line under the text, and nothing to press on it: a book is turned
         by the hand and the keyboard. The count is carried over the book and
         what is left of the chapter is measured on the page in front. -->
    <footer class="book__foot text-small text-hushed">
      <span class="book__way"><slot name="way" /></span>
      <template v-if="count > 0">
        <span class="book__count">{{ words.of(paged.page, paged.pages) }}</span>
        <span class="book__left">{{ words.left(left) }}</span>
      </template>
    </footer>
  </div>
</template>

<style scoped>
.book {
  /* The margin over the text, which the running head stands in the middle of. */
  --book-head: 4rem;
}

/* One line under the text: the count in the middle of it and what is left of
   the chapter at the end, as a book has them. Nothing on it is pressed, so it
   lets a press through to the page behind. */
.book__foot {
  position: absolute;
  inset-block-end: var(--numen-inset-wide);
  inset-inline: var(--numen-inset-wide);
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: baseline;
  pointer-events: none;
}

.book__count {
  grid-column: 2;
}

.book__left {
  grid-column: 3;
  justify-self: end;
}

/* The clearance the text keeps from the pane, and the room the controls stand
   in below it. The area itself carries none of it: what it measures across is
   the width of a spread. */
.book__margin {
  box-sizing: border-box;
  /* The gutter a book keeps beside its text, which is wide: a column runs to
     the measure it is set at and the room left over is margin. */
  padding-inline: clamp(1rem, 3%, 2.5rem);
  padding-block-start: var(--book-head);
  padding-block-end: 3rem;
}

/* What the book calls the place in front, standing in the margin over the text
   it names, midway between the top of the window and the top of that text. */
.book__head {
  position: absolute;
  inset-block-start: 0;
  block-size: var(--book-head);
  inset-inline: clamp(1rem, 3%, 2.5rem);
  display: grid;
  place-items: center;
  overflow: hidden;
  white-space: nowrap;
  text-align: center;
  text-overflow: ellipsis;
  pointer-events: none;
}

/* The way into the contents stands on the line under the text and carries
   nothing drawn around it. */
.book__way {
  grid-column: 1;
  justify-self: start;
  pointer-events: auto;
}

/* The page a person is reading carries nothing drawn around it. */
.book__area:focus-visible {
  outline: none;
}

/* The columns run sideways out of the area, and how far they run is read off
   it. */
.book__paper {
  /* How tall a column is set: a whole number of lines, which the reader works
     out once the text is laid out, and the whole of the area until it has. */
  --book-paper: var(--book-high);

  block-size: var(--book-paper);
  /* Two columns, always. A single-column box is not broken into columns at all
     by WebKit: the text past the first column is cut off and never reached. A
     spread of one column is two set across a box twice as wide, and the area
     around it shows one of them. */
  inline-size: var(--book-run);
  column-count: 2;
  column-gap: var(--book-gap);
  column-fill: auto;
  font-size: var(--book-size);
  /* Set to the measure, broken at the syllable, as a book is. */
  text-align: justify;
  hyphens: auto;
  /* A word longer than the column is broken inside itself. */
  overflow-wrap: anywhere;
  /* The window takes selection away from everything and gives it back to what
     is there to be read. A book is there to be read. */
  user-select: text;
  -webkit-user-select: text;
}

/* A picture is set to its column's width and no taller than the column. */
.book__paper :deep(img),
.book__paper :deep(svg),
.book__paper :deep(figure) {
  box-sizing: border-box;
  display: block;
  max-inline-size: var(--book-column);
  max-block-size: var(--book-paper);
  object-fit: contain;
}

/* A table is set to the column and carries its own scrollbars where it will not
   break. */
.book__paper :deep(table) {
  display: block;
  box-sizing: border-box;
  max-inline-size: var(--book-column);
  max-block-size: var(--book-paper);
  overflow: auto;
}

/* Verse, a title page and whatever else a book sets in a pre element keep the
   lines they were written on and are read across the columns as the rest of the
   text is. A line too long for the column is wrapped. */
.book__paper :deep(pre) {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  max-inline-size: var(--book-column);
  /* A box that clips its own overflow cannot be cut between two columns, and
     the typography this paper carries gives one to every pre. */
  overflow: visible;
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

/* A place the person was not sent to is drawn faintly: it says there is
   something here, and the place they were sent to is the one drawn full. */
:global(::highlight(numen-book-also)) {
  background-color: color-mix(in srgb, var(--numen-highlight) 35%, transparent);
}
</style>
