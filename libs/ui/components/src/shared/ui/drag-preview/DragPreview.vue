<script setup lang="ts">
/**
 * The dragged item's label, shown next to the pointer without capturing events.
 * One line, then an ellipsis.
 */
import { computed } from 'vue'
import type { Position } from '@/shared/lib/math/geometry'

const props = defineProps<{
  at: Position
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
  --pad-block: 0.15rem;
  --pad-inline: 0.5rem;

  position: fixed;
  max-inline-size: var(--widest);
  margin: 0;
  padding: var(--pad-block) var(--pad-inline);
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
