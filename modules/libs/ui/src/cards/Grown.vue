<script setup lang="ts">
/**
 * A box as tall as what it holds.
 *
 * The ground behind the box is set to the box's own text, in the same type at
 * the same measure, and the two share one cell. The ground is what the cell is
 * sized by, so the box is exactly as tall as its text and never scrolls.
 */
defineProps<{
  /** What the box holds, as it now reads. */
  text: string
}>()
</script>

<template>
  <div class="grown" :data-grown="text"><slot /></div>
</template>

<style scoped>
/* A box holding nothing stands as tall as a box holding one line. */
.grown {
  display: grid;
  min-block-size: calc(
    var(--grown-lines, 1) * var(--numen-line-height) * 1em + 2 * var(--box-air, 0.5rem)
  );
}

/* The space at the end keeps a line for text ending in a newline, and for no
   text at all. */
.grown::after {
  content: attr(data-grown) ' ';
  visibility: hidden;
}

.grown > :deep(textarea),
.grown::after {
  grid-area: 1 / 1;
  padding: var(--box-air, 0.5rem) var(--box-pad-inline, 0.625rem);
  border: none;
  background: none;
  color: var(--numen-node-fg);
  font: inherit;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  overflow: hidden;
  resize: none;
}

.grown > :deep(textarea:focus-visible) {
  outline: none;
}
</style>
