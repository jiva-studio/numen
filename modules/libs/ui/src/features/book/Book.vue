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
import { useTemplateRef } from 'vue'

import { placeIn, pointsAway, type BookLink } from './link'
import { handTurn, keyTurn } from './turn'
import { useBookLayout } from './layout'
import { BOOK_WORDS } from './spread'
import type { BookProps } from './props'

const props = withDefaults(defineProps<BookProps>(), {
  markup: '',
  path: '',
  span: () => ({ begins: 0, ends: 0 }),
  book: () => ({ begins: 0, ends: 0 }),
  at: 0,
  highlights: () => [],
  elsewhere: () => [],
  textSize: 1,
  chapter: '',
  words: () => BOOK_WORDS,
})

const emit = defineEmits<{
  /** The offset now in front, in bytes of the book's text. */
  (event: 'moved', at: number): void
  /**
   * A link led to another document of the book, named as the archive names it.
   * The reader lands on the place inside it once that document is drawn.
   */
  (event: 'followed', path: string): void
}>()

const area = useTemplateRef<HTMLElement>('area')
const paper = useTemplateRef<HTMLElement>('paper')

/** Where a link led, held until the document holding that place is drawn. */
let led: BookLink | undefined
const takeLed = (): BookLink | undefined => {
  const place = led
  led = undefined
  return place
}

const layout = useBookLayout(area, paper, props, (at) => emit('moved', at), takeLed, (of) =>
  of.getBoundingClientRect().left,
)
const { measured, setting, spreadCount, front, leftInChapter } = layout

/**
 * A key the tab caught. Turning belongs to whatever holds the book, so the
 * page it turns is answered for and the key is not listened for here.
 */
const pressed = (event: KeyboardEvent): boolean => {
  const way = keyTurn(event.key)
  if (!way) return false
  layout.turn(way)
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
  if (way) layout.turn(way)
}

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
    emit('followed', place.path)
    return
  }
  emit('moved', layout.placeAt(place.fragment) ?? props.span.begins)
}

defineExpose({
  /**
   * Set the text again. A reader drawn out of sight has no reading area, and
   * the caller says when it is on screen.
   */
  measure: () => layout.settle(layout.keeping()),
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
      <template v-if="spreadCount > 0">
        <span class="book__count">{{ words.of(front.page, front.pages) }}</span>
        <span class="book__left">{{ words.left(leftInChapter) }}</span>
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
  /* The gutter a book keeps beside its text stands inside the columns and not
     around them: the page a turn carries off goes off the edge of the pane
     rather than being cut where the text begins. */
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

  box-sizing: border-box;
  block-size: var(--book-paper);
  /* Two columns, always. A single-column box is not broken into columns at all
     by WebKit: the text past the first column is cut off and never reached. A
     spread of one column is two set across a box twice as wide, and the area
     around it shows one of them. */
  inline-size: var(--book-run);
  /* Half a gap beside the outermost column at either edge, so a column stands
     the same distance from the edge as two columns stand from each other and a
     spread begins one whole area along. */
  padding-inline: calc(var(--book-gap) / 2);
  column-count: 2;
  column-gap: var(--book-gap);
  /* The spread in front is carried here, and the turn is that being carried. */
  transition: translate var(--numen-motion) var(--numen-easing);
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
  /* A pre in a book is a title page or a verse, and stands on the paper the
     rest of the text does. The typography dresses one as a block of code. */
  padding: 0;
  border-radius: 0;
  background: none;
  color: inherit;
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
:global(::highlight(numen-book-elsewhere)) {
  background-color: color-mix(in srgb, var(--numen-highlight) 35%, transparent);
}
</style>
