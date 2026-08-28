<script setup lang="ts">
/** One of the marks a stencil and a deck are drawn with. */
import { MARKS, type Mark } from './marks'

defineProps<{ shows: Mark }>()
</script>

<template>
  <svg
    class="glyph"
    viewBox="0 0 12 12"
    :data-filled="MARKS[shows].filled || undefined"
    aria-hidden="true"
    focusable="false"
  >
    <path :d="MARKS[shows].path" />
  </svg>
</template>

<style scoped>
/* A mark stands at about the height of the letter beside it, in its ink. It is
   drawn for the eye, so a press on it lands on whatever it stands in. */
.glyph {
  flex: none;
  pointer-events: none;
  inline-size: 0.875em;
  block-size: 0.875em;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.5;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.glyph[data-filled] {
  fill: currentColor;
  stroke: none;
}
</style>
