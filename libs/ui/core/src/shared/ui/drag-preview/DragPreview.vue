<script setup lang="ts">
/**
 * The dragged item's label, shown next to the pointer without capturing events.
 * One line, then an ellipsis.
 */
import { computed } from 'vue'
import type { Position } from '@/shared/lib/math/geometry'

const props = defineProps<{
  /** Where the pointer is, in pixels from the top left of the window. */
  at: Position
  /** What is being dragged, in the caller's own words. */
  label: string
}>()

const previewStyle = computed(() => ({
  left: `${props.at.x}px`,
  top: `${props.at.y}px`,
}))
</script>

<template>
  <p class="drag-preview text-small font-sans" :style="previewStyle">
    {{ label }}
  </p>
</template>

<style scoped>
.drag-preview {
  /* The offset of the preview from the pointer, and the maximum width before its
     text is truncated. */
  --gap: 0.75rem;
  --widest: 15rem;

  position: fixed;
  max-inline-size: var(--widest);
  margin: 0;
  padding: 0.15rem 0.5rem;
  translate: var(--gap) var(--gap);
  pointer-events: none;
  overflow: hidden;
  border-radius: var(--numen-radius);
  background: var(--numen-accent);
  color: var(--numen-accent-ink);
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
