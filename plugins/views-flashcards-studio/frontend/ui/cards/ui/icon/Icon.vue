<script setup lang="ts">
/** One of the icons a card model and a deck are drawn with. */
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
.icon {
  --icon-inline-size: 0.875em;
  --icon-block-size: 0.875em;
  flex: none;
  pointer-events: none;
  inline-size: var(--icon-inline-size);
  block-size: var(--icon-block-size);
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
