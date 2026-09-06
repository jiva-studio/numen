<script setup lang="ts">
/**
 * What a strip is pressed to be rid of what it heads: the bin alone, named to
 * a reader.
 *
 * It carries no ground and no shape of its own, and comes up to the ink of what
 * it stands on under the hand about to press it.
 */
import Icon from './Icon.vue'

defineProps<{
  /** What it is called, read aloud and shown on hovering. */
  label: string
}>()

const emit = defineEmits<{ (event: 'press'): void }>()
</script>

<template>
  <button
    type="button"
    class="remove-button"
    draggable="false"
    :aria-label="label"
    :title="label"
    @click="emit('press')"
  >
    <Icon shows="bin" />
  </button>
</template>

<style scoped>
.remove-button {
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

.remove-button:hover,
.remove-button:focus-visible {
  color: var(--numen-ink);
}

.remove-button:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: 1px;
}

@media (prefers-reduced-motion: reduce) {
  .remove-button {
    transition: none;
  }
}
</style>
