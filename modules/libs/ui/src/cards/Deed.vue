<script setup lang="ts">
/**
 * One deed a strip is pressed for: the glyph alone, named to a reader.
 *
 * It carries no ground and no shape of its own, because the strip does not draw
 * it until the strip is reached for. It is set as quietly as what a value is
 * called, and comes up to the ink of what it stands on under the hand about to
 * press it.
 */
import Glyph from './Glyph.vue'
import type { Mark } from './marks'

defineProps<{
  /** The glyph it is drawn as. */
  shows: Mark
  /** What it is called, read aloud and shown on hovering. */
  label: string
}>()

const emit = defineEmits<{ (event: 'press'): void }>()
</script>

<template>
  <button
    type="button"
    class="deed"
    draggable="false"
    :aria-label="label"
    :title="label"
    @click="emit('press')"
  >
    <Glyph :shows="shows" />
  </button>
</template>

<style scoped>
.deed {
  display: grid;
  place-items: center;
  inline-size: 1.5rem;
  block-size: 1.5rem;
  border: 0;
  background: transparent;
  color: var(--numen-edge-label);
  cursor: pointer;
  transition: color var(--numen-motion-hover) var(--numen-easing);
}

.deed:hover,
.deed:focus-visible {
  color: var(--numen-node-fg);
}

.deed:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: 1px;
}

@media (prefers-reduced-motion: reduce) {
  .deed {
    transition: none;
  }
}
</style>
