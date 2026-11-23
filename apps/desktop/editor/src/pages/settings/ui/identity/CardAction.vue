<script setup lang="ts">
/**
 * An action in the header of an identity card: an icon and a word with no ground. Hover changes only
 * its colour; the primary action takes the accent.
 */

/* --------------------------------- Props ---------------------------------- */
defineProps<{ isPrimary?: boolean; isDisabled?: boolean }>()
</script>

<template>
  <button
    type="button"
    class="card-action"
    :class="{ 'card-action--primary': isPrimary }"
    :disabled="isDisabled"
  >
    <slot />
  </button>
</template>

<style scoped>
.card-action {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  box-sizing: border-box;
  block-size: calc(var(--numen-text-2) * var(--numen-line-height));
  margin: 0;
  padding: 0 0.375rem;
  border: none;
  border-radius: var(--numen-radius-tight);
  background: none;
  color: var(--numen-hushed);
  font: inherit;
  font-size: var(--numen-text-1);
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition: color var(--numen-motion-hover) var(--numen-easing);
}

.card-action:hover {
  color: var(--numen-ink);
}

.card-action--primary {
  color: var(--numen-accent);
}

.card-action--primary:hover {
  color: color-mix(in oklab, var(--numen-accent), var(--numen-ink) 25%);
}

.card-action:focus {
  outline: none;
}

.card-action:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.card-action:disabled {
  cursor: default;
  opacity: 0.5;
}

/* The header pads its ends by the card padding and its top by 0.375rem: the last action moves right
   so its box stands as far from the right edge as from the top. */
.card-action:last-child {
  margin-inline-end: calc(0.375rem - var(--numen-card-pad));
}
</style>
