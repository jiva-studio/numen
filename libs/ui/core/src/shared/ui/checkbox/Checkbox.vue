<script setup lang="ts">
/**
 * A box that is ticked or not. It is a real checkbox input drawn by the library, so the
 * keyboard, the form value and the label behave as the platform defines them.
 *
 * Its name comes from the default slot, which is wrapped in a label, or from an `aria-label`
 * passed as an attribute. Attributes other than `class` go to the input.
 */
import type { HTMLAttributes } from 'vue'
import { Check } from '@lucide/vue'
import { cn } from '@/shared/lib/dom/classes'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    isDisabled?: boolean
    /** A tooltip over the box and its label. */
    title?: string
    class?: HTMLAttributes['class']
  }>(),
  { isDisabled: false },
)

const model = defineModel<boolean>({ default: false })
</script>

<template>
  <label
    data-slot="checkbox"
    :title="props.title"
    :class="
      cn(
        'inline-flex items-center gap-2',
        props.isDisabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer',
        props.class,
      )
    "
  >
    <span class="relative inline-flex size-4 shrink-0">
      <input
        v-bind="$attrs"
        v-model="model"
        type="checkbox"
        :disabled="props.isDisabled"
        :class="
          cn(
            'peer m-0 size-4 appearance-none border',
            'rounded-tight border-field-rule bg-field',
            'duration-hover ease-numen transition-colors outline-none',
            'checked:border-accent checked:bg-accent',
            'enabled:hover:border-rule enabled:checked:hover:border-accent',
            'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-(--numen-ring)',
            'cursor-[inherit]',
          )
        "
      />
      <Check
        :size="12"
        :stroke-width="3"
        aria-hidden="true"
        class="text-accent-ink pointer-events-none absolute inset-0 m-auto opacity-0 peer-checked:opacity-100"
      />
    </span>
    <slot />
  </label>
</template>
