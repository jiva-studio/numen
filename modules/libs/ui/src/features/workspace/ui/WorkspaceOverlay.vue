<script setup lang="ts">
/** Where a dragged tab would go, drawn over the panes. */
import { computed } from 'vue'
import type { Rect } from '../lib/rect'

const props = defineProps<{
  box: Rect
  /** Drawn as a line between two tabs, for a place in a strip. */
  caret?: boolean
}>()

const overlayStyle = computed(() => ({
  left: `${props.box.x}px`,
  top: `${props.box.y}px`,
  width: `${props.box.width}px`,
  height: `${props.box.height}px`,
}))
</script>

<template>
  <div
    class="workspace__overlay"
    :data-caret="caret || undefined"
    :style="overlayStyle"
  />
</template>

<style scoped>
/* Shown over everything and catching nothing. The wash is the same colour as
   the outline, laid on thinly. */
.workspace__overlay {
  /* How heavily the wash inside the outline is laid on. */
  --wash: 0.16;

  position: absolute;
  z-index: 2;
  pointer-events: none;
  border: var(--numen-ring-width) solid var(--numen-ring);
  border-radius: var(--numen-radius);
}

.workspace__overlay::before {
  content: '';
  position: absolute;
  inset: 0;
  background: var(--numen-ring);
  opacity: var(--wash);
}

.workspace__overlay[data-caret] {
  inline-size: var(--numen-caret);
  margin-inline-start: calc(var(--numen-caret) / -2);
  border: none;
  background: var(--numen-ring);
}
</style>
