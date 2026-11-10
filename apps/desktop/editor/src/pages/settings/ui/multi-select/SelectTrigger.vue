<script setup lang="ts">
/**
 * The line a select shows at rest: what is in force on one line, and the mark that it opens.
 * Attributes and listeners fall through to the button.
 */
import { useTemplateRef, type HTMLAttributes } from 'vue'
import { ChevronDown } from '@lucide/vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  text: string
  isHushed: boolean
  isDisabled: boolean
  isExpanded: boolean
  hasPopup: 'menu' | 'listbox'
  class?: HTMLAttributes['class']
}>()

/* ---------------------------------- State --------------------------------- */
const element = useTemplateRef<HTMLButtonElement>('element')

defineExpose({
  /** The button itself. */
  element,
})
</script>

<template>
  <button
    ref="element"
    type="button"
    data-slot="select"
    class="select-trigger"
    :aria-haspopup="hasPopup"
    :aria-expanded="isExpanded"
    :disabled="isDisabled"
    :class="props.class"
  >
    <span
      class="select-trigger__reading"
      :class="{ 'select-trigger__reading--hushed': isHushed }"
      :title="text"
    >
      {{ text }}
    </span>
    <ChevronDown class="select-trigger__mark" aria-hidden="true" />
  </button>
</template>

<style scoped>
.select-trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  inline-size: 100%;
  max-inline-size: 100%;
  min-inline-size: 0;
  block-size: var(--numen-action-size);
  padding-inline: 0.625rem;
  border: 1px solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  background: var(--numen-field-bg);
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-font-size);
  line-height: 1;
  text-align: start;
  cursor: pointer;
  outline: none;
  transition-property: color, background-color, border-color;
  transition-duration: 100ms;
  transition-timing-function: var(--numen-easing);
}

.select-trigger:hover {
  border-color: var(--numen-rule);
}

.select-trigger:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

/* One line, then an ellipsis. A model is named by a word and addressed by a
   line too long to read, and the name is what the line carries. */
.select-trigger__reading {
  min-inline-size: 0;
  padding-block: 0.125rem;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.select-trigger__reading--hushed {
  color: var(--numen-hushed);
}

/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.select-trigger__mark {
  flex-shrink: 0;
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.875;
  color: var(--numen-hushed);
}
</style>
