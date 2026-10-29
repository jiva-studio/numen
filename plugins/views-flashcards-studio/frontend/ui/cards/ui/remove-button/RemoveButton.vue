<script setup lang="ts">
/**
 * A strip's control for removing what it heads: the bin icon alone, with an
 * accessible name.
 *
 * It has no background or shape of its own, and takes the colour of the
 * element it is placed on while it is hovered or focused.
 */
import { Icon } from '../icon'

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
    <Icon name="bin" />
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
  outline: none;
  outline-offset: 1px;
}

@media (prefers-reduced-motion: reduce) {
  .remove-button {
    transition: none;
  }
}
</style>
