<script setup lang="ts">
/**
 * The line over a book's text: what the book calls the place in front.
 */
/* --------------------------------- Props ---------------------------------- */
defineProps<{
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
  --inset-min: 1rem;
  --inset-max: 2.5rem;
  position: absolute;
  inset-block-start: 0;
  block-size: var(--book-head);
  inset-inline: clamp(var(--inset-min), 3%, var(--inset-max));
  display: grid;
  place-items: center;
  overflow: hidden;
  white-space: nowrap;
  text-align: center;
  text-overflow: ellipsis;
  pointer-events: none;
}
</style>
