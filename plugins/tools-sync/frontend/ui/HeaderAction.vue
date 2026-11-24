<script setup lang="ts">
/**
 * An action in the header of a card: an icon and a word with no ground, the same as the actions in the
 * headers of the identity cards. Hover changes only its colour.
 */

/* --------------------------------- Props ---------------------------------- */
defineProps<{ isDisabled?: boolean }>()
</script>

<template>
  <button type="button" class="header-action" :disabled="isDisabled">
    <slot />
  </button>
</template>

<style scoped>
.header-action {
  display: inline-flex;
  align-items: center;
  gap: calc(0.375 * var(--numen-chrome-unit));
  box-sizing: border-box;
  block-size: calc(var(--numen-chrome-text-2) * var(--numen-line-height));
  margin: 0;
  padding: 0 calc(0.375 * var(--numen-chrome-unit));
  border: none;
  border-radius: var(--numen-chrome-radius-tight);
  background: none;
  color: var(--numen-accent);
  font: inherit;
  font-size: var(--numen-chrome-text-1);
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition: color var(--numen-motion-hover) var(--numen-easing);
}

.header-action:hover {
  color: color-mix(in oklab, var(--numen-accent), var(--numen-ink) 25%);
}

.header-action:focus {
  outline: none;
}

.header-action:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.header-action:disabled {
  cursor: default;
  opacity: 0.5;
}

.header-action:last-child {
  margin-inline-end: calc(0.375 * var(--numen-chrome-unit) - var(--numen-chrome-unit));
}
</style>
