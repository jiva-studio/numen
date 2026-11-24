<script setup lang="ts">
import { ChevronDown } from '@lucide/vue'
import { useTemplateRef } from 'vue'
import { Tooltip, useTooltip } from '../tooltip'
import type { CardActionProps } from './CardAction.types'

/**
 * An action in the header of an island card: an icon in the default slot and the label beside it. The
 * label hides when the card gets narrow, and then names the button as its tooltip; the accessible name is always the label.
 */

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<CardActionProps>()

const tip = useTooltip()
const button = useTemplateRef<HTMLButtonElement>('button')

function getTip(): string {
  const shown = button.value?.querySelector<HTMLElement>('.card-action__label')
  return shown && getComputedStyle(shown).display !== 'none' ? '' : props.label
}
</script>

<template>
  <button
    ref="button"
    type="button"
    class="card-action"
    :class="{
      'card-action--primary': isPrimary,
      'card-action--secondary': isSecondary,
      'card-action--more': isMore,
      'card-action--kept': isLabelKept,
    }"
    :disabled="isDisabled"
    :aria-label="label"
    :aria-describedby="tip.describedBy.value"
    @mouseenter="tip.open($event, getTip)"
    @mouseleave="tip.close"
    @focus="tip.open($event, getTip)"
    @blur="tip.close"
  >
    <slot />
    <span v-if="!isMore && !isIconOnly" class="card-action__label" aria-hidden="true">{{ label }}</span>
    <ChevronDown v-if="hasMenu" :size="12" aria-hidden="true" />
    <Teleport to="body">
      <Tooltip v-if="tip.at.value" :id="tip.id" :at="tip.at.value">{{ tip.text.value }}</Tooltip>
    </Teleport>
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

/* A kept label is the last to give way: it shrinks and ends with an ellipsis. */
.card-action--kept {
  flex: 0 1 auto;
  min-inline-size: 0;
}

.card-action__label {
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.card-action:disabled {
  cursor: default;
  opacity: 0.5;
}

.card-action--more {
  display: none;
}
</style>
