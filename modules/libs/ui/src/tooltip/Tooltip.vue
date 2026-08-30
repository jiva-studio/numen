<script setup lang="ts">
/**
 * What a thing is, said beside it while a person points at it.
 *
 * It follows the pointer and never takes it: a tooltip a mouse can land on is
 * one that moves out from under the hand reaching for what it is about.
 */
defineProps<{
  /** Where it stands, in pixels from the top left of the window. */
  at: { x: number; y: number }
}>()
</script>

<template>
  <aside
    class="tooltip"
    role="tooltip"
    :style="{ insetInlineStart: `${at.x}px`, insetBlockStart: `${at.y}px` }"
  >
    <slot />
  </aside>
</template>

<style scoped>
.tooltip {
  position: fixed;
  z-index: 1;
  padding: var(--numen-inset);
  border: 1px solid var(--numen-node-border);
  border-radius: var(--numen-radius);
  background: var(--numen-node-bg);
  color: var(--numen-node-fg);
  box-shadow: var(--numen-shadow-raised, 0 2px 8px rgb(0 0 0 / 25%));
  font-size: var(--numen-text-1);
  pointer-events: none;
}
</style>
