<script setup lang="ts">
/**
 * The line a select shows at rest: what is in force on one line, and the mark that it opens.
 * Attributes and listeners fall through to the button.
 */
import { useTemplateRef } from 'vue'
import { ChevronDown } from '@lucide/vue'
import { cn } from '@/shared/lib/dom/classes'
import type { SelectTriggerProps } from './SelectTrigger.types'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<SelectTriggerProps>()

defineExpose({
  /** The button itself. */
  get element() {
    return element.value
  },
})

/* ---------------------------------- State --------------------------------- */
const element = useTemplateRef<HTMLButtonElement>('element')
</script>

<template>
  <button
    ref="element"
    type="button"
    data-slot="select"
    :aria-haspopup="hasPopup"
    :aria-expanded="isExpanded"
    :disabled="isDisabled"
    :title="problem || undefined"
    :aria-description="problem || undefined"
    :aria-invalid="problem ? true : undefined"
    :class="
      cn(
        'flex w-full max-w-full min-w-0 items-center justify-between gap-2',
        'h-action rounded-node border-field-rule bg-field border px-2.5',
        'text-ink text-left font-sans text-base leading-none',
        'ease-numen cursor-pointer transition-colors duration-100 outline-none',
        'hover:border-rule',
        problem && 'border-alarm hover:border-alarm',
        'focus-visible:outline-none',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
  >
    <span class="select__reading min-w-0" :class="{ 'text-hushed': isHushed }" :title="text">
      {{ text }}
    </span>
    <ChevronDown class="select__mark shrink-0" aria-hidden="true" />
  </button>
</template>

<style scoped>
/* One line, then an ellipsis. A model is named by a word and addressed by a
   line too long to read, and the name is what the line carries. */
.select__reading {
  --lead: 0.125rem;

  padding-block: var(--lead);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.select__mark {
  --mark: 0.875rem;

  inline-size: var(--mark);
  block-size: var(--mark);
  stroke-width: 1.875;
  color: var(--numen-hushed);
}
</style>
