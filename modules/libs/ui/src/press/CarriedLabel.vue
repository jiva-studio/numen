<script setup lang="ts">
/**
 * What is being carried, said beside the pointer and catching nothing. One
 * line, then an ellipsis.
 */
import type { Point } from '../lib/geometry'

defineProps<{
  /** Where the pointer is, in pixels from the top left of the window. */
  at: Point
  /** What is being carried, in the caller's own words. */
  says: string
}>()
</script>

<template>
  <p
    class="carried font-sans text-small"
    :style="{ left: `${at.x}px`, top: `${at.y}px` }"
  >
    {{ says }}
  </p>
</template>

<style scoped>
.carried {
  /* How far it stands clear of the pointer, and how far it reaches before what
     it says is cut. */
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
