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

import { keyTurn } from '@/shared/lib/turn'
import { useBookHand } from '../model/hand'
import { useBookLayout } from '../model/layout'
import { createBookLinks } from '../model/links'
import { BOOK_WORDS } from '../lib/words'
import type { BookProps } from '../lib/props'
import BookFoot from './BookFoot.vue'
import BookHead from './BookHead.vue'

const props = withDefaults(defineProps<BookProps>(), {
  markup: '',
  path: '',
  span: () => ({ from: 0, to: 0 }),
  book: () => ({ from: 0, to: 0 }),
  at: 0,
  highlights: () => [],
  otherHighlights: () => [],
  textSize: 1,
  chapter: '',
  words: () => BOOK_WORDS,
})

const emit = defineEmits<{
  /** The offset now in front, in bytes of the book's text. */
  (event: 'move', at: number): void
  /**
   * A link led to another document of the book, named as the archive names it.
   * The reader lands on the place inside it once that document is drawn.
   */
  (event: 'follow', path: string): void
}>()

const area = useTemplateRef<HTMLElement>('area')
const paper = useTemplateRef<HTMLElement>('paper')

/** The near edge of a box, which every place across the columns is read from. */
const edgeOf = (of: HTMLElement) => of.getBoundingClientRect().left

const links = createBookLinks(props, {
  reportMove: (at) => emit('move', at),
  reportFollow: (path) => emit('follow', path),
})

const layout = useBookLayout(area, paper, props, (at) => emit('move', at), links.takeLed, edgeOf)
const { measured, setting, spreadCount, front, leftInChapter } = layout

const hand = useBookHand(area, paper, layout.turn, edgeOf)

const follow = (press: MouseEvent) => links.follow(press, layout.placeAt)

/**
 * A key the tab caught. Turning belongs to whatever holds the book, so the
 * page it turns is answered for and the key is not listened for here.
 */
const handleKey = (event: KeyboardEvent): boolean => {
  const way = keyTurn(event.key)
  if (!way) return false
  layout.turn(way)
  return true
}

defineExpose({
  /**
   * Set the text again. A reader drawn out of sight has no reading area, and
   * the caller says when it is on screen.
   */
  measure: () => layout.settle(layout.getKeptOffset()),
  /** A key the tab caught: true where it turned the page. */
  handleKey,
})
</script>

<template>
  <div class="book numen text-ink relative h-full min-h-0 font-sans text-base">
    <BookHead :chapter="chapter" />

    <div class="book__margin h-full">
      <div
        ref="area"
        class="book__area h-full overflow-hidden"
        role="region"
        :aria-label="words.pages"
        @pointerdown="hand.takeDown"
        @pointerup="hand.letGo"
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

    <!-- A book is turned by the hand and the keyboard, so the line under the
         text carries no control. -->
    <BookFoot
      :words="words"
      :front="front"
      :left-in-chapter="leftInChapter"
      :spread-count="spreadCount"
    >
      <template #way><slot name="way" /></template>
    </BookFoot>
  </div>
</template>

<style scoped>
.book {
  /* The margin over the text, which the running head stands in the middle of. */
  --book-head: 4rem;
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

/* The page a person is reading carries nothing drawn around it. */
.book__area:focus-visible {
  outline: none;
}

/* The columns run sideways out of the area, and how far they run is read off
   it. */
.book__paper {
  /* How tall a column is set: a whole number of lines, which the reader works
     out once the text is laid out, and the whole of the area until it has. */
  --book-paper: var(--book-height);

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
:global(::highlight(numen-book-other-highlight)) {
  background-color: color-mix(in srgb, var(--numen-highlight) 35%, transparent);
}
</style>
