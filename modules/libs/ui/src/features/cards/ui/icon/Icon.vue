<script setup lang="ts">
/** One of the icons a stencil and a deck are drawn with. */
import { ICONS, type Icon } from './icons'

defineProps<{ name: Icon }>()
</script>

<template>
  <svg
    class="icon"
    viewBox="0 0 12 12"
    :data-filled="ICONS[name].isFilled || undefined"
    aria-hidden="true"
    focusable="false"
  >
    <path :d="ICONS[name].path" />
  </svg>
</template>

<style scoped>
/* An icon stands at about the height of the letter beside it, in its ink. It is
   drawn for the eye, so a press on it lands on whatever it stands in. */
.icon {
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

.icon[data-filled] {
  fill: currentColor;
  stroke: none;
}
</style>
