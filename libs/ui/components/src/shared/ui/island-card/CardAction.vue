<script setup lang="ts">
/**
 * An action in the header of an island card: an icon in the default slot and the label beside it. The
 * label hides when the card gets narrow, and then names the button as its tooltip and accessible name.
 */

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  label: string
  isPrimary?: boolean
  isDisabled?: boolean
  /** Hidden when the card is too narrow even for icons: the overflow menu carries it. */
  isSecondary?: boolean
  /** Shows the icon at every width. */
  isIconOnly?: boolean
  /** Shown only where the secondary actions are hidden. */
  isMore?: boolean
}>()
</script>

<template>
  <button
    type="button"
    class="card-action"
    :class="{
      'card-action--primary': isPrimary,
      'card-action--secondary': isSecondary,
      'card-action--more': isMore,
    }"
    :disabled="isDisabled"
    :aria-label="label"
    :title="label"
  >
    <slot />
    <span v-if="!isMore && !isIconOnly" class="card-action__label" aria-hidden="true">{{ label }}</span>
  </button>
</template>

<style scoped>
.card-action {
  display: inline-flex;
  flex: none;
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
  font-size: var(--numen-text-2);
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

.card-action--more {
  display: none;
}
</style>
