<script setup lang="ts">
/**
 * The line under a book's text: the way into its contents, the page in front,
 * and what is left of the chapter.
 */
import type { Pages } from '../lib/spread'
import type { BookWords } from '../lib/words'

defineProps<{
  /** The words it is read with. */
  words: BookWords
  /** The spread in front, as a page of the book, and how many there are. */
  front: Pages
  /** How many columns of this chapter stand after the spread in front. */
  leftInChapter: number
  /** How many spreads the document is read in. */
  spreadCount: number
}>()
</script>

<template>
  <!-- Nothing on this line is pressed, so it lets a press through to the page
       behind. The count is carried over the book and what is left of the
       chapter is measured on the page in front. -->
  <footer class="book__foot text-small text-hushed">
    <span class="book__way"><slot name="way" /></span>
    <template v-if="spreadCount > 0">
      <span class="book__count">{{ words.of(front.page, front.pages) }}</span>
      <span class="book__left">{{ words.left(leftInChapter) }}</span>
    </template>
  </footer>
</template>

<style scoped>
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

/* The way into the contents carries nothing drawn around it. */
.book__way {
  grid-column: 1;
  justify-self: start;
  pointer-events: auto;
}
</style>
