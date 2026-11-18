<script setup lang="ts">
/**
 * A strip's control for removing what it heads: the bin icon alone, with an
 * accessible name.
 *
 * It has no background or shape of its own, and takes the colour of the
 * element it is placed on while it is hovered or focused.
 */
import { Tooltip, useTooltip } from '@numen/ui'
import { Icon } from '../icon'

defineProps<{
  label: string
}>()

const emit = defineEmits<{ (event: 'press'): void }>()

const tip = useTooltip()
</script>

<template>
  <button
    type="button"
    class="remove-button"
    draggable="false"
    :aria-label="label"
    :aria-describedby="tip.describedBy.value"
    @mouseenter="tip.open"
    @mouseleave="tip.close"
    @focus="tip.open"
    @blur="tip.close"
    @click="emit('press')"
  >
    <Icon name="bin" />
    <Teleport to="body">
      <Tooltip v-if="tip.at.value" :id="tip.id" :at="tip.at.value">{{ label }}</Tooltip>
    </Teleport>
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
