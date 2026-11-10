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
  --remove-button-inline-size: 1.5rem;
  --remove-button-block-size: 1.5rem;
  display: grid;
  place-items: center;
  inline-size: var(--remove-button-inline-size);
  block-size: var(--remove-button-block-size);
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
  --remove-button-focus-visible-outline-offset: 0.0625rem;
  outline: none;
  outline-offset: var(--remove-button-focus-visible-outline-offset);
}

@media (prefers-reduced-motion: reduce) {
  .remove-button {
    transition: none;
  }
}
</style>
