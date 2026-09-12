<script setup lang="ts">
/**
 * The line over a book's text: what the book calls the place in front.
 */
defineProps<{
  /** What the book calls the place in front. */
  chapter: string
}>()
</script>

<template>
  <header class="book__head text-small text-hushed">{{ chapter }}</header>
</template>

<style scoped>
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
</style>
