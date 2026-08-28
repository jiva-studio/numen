<script setup lang="ts">
/**
 * A box as tall as what it holds.
 *
 * The ground behind the box is set to the box's own text, in the same type at
 * the same measure, and the two share one cell. The ground is what the cell is
 * sized by, so the box is exactly as tall as its text and never scrolls.
 *
 * Everything a caller hands it that is not named here lands on the box: what
 * the box is called, what it is identified by, and every key it is listened to
 * for. A class and a style stand on the cell, which is what a caller lays out.
 */
import { computed, useAttrs } from 'vue'

defineOptions({ inheritAttrs: false })

defineProps<{
  /** What the box holds, as it now reads. */
  text: string
}>()

const attrs = useAttrs()

/** What the caller handed the box itself. */
const box = computed(() => {
  const held = { ...attrs }
  delete held['class']
  delete held['style']
  return held
})

const emit = defineEmits<{
  /** The text as it now reads, after something was typed into the box. */
  (event: 'write', text: string): void
}>()
</script>

<template>
  <div class="grown" :class="attrs.class" :style="attrs.style" :data-grown="text">
    <textarea
      class="grown__box"
      rows="1"
      :value="text"
      v-bind="box"
      @input="emit('write', ($event.target as HTMLTextAreaElement).value)"
    ></textarea>
  </div>
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

.grown__box {
  min-inline-size: 0;
}

.grown__box,
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

.grown__box:focus-visible {
  outline: none;
}
</style>
