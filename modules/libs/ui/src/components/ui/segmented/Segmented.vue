<script setup lang="ts">
/**
 * Two to four choices side by side, one of them chosen. The chosen segment is
 * filled with the accent and the rest are quiet.
 *
 * The whole control is one stop on the way round the screen, and the arrow
 * keys move between the segments and choose as they go.
 */
import type { HTMLAttributes } from 'vue'
import { RadioGroupItem, RadioGroupRoot } from 'reka-ui'
import { cn } from '@/lib/utils'
import { type SegmentedChoice } from '.'

const props = withDefaults(
  defineProps<{
    /** The choices, in the order they are offered. */
    choices: readonly SegmentedChoice[]
    disabled?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { disabled: false },
)

/** Which choice is in force, by the identifier the caller gave it. */
const model = defineModel<string>({ default: '' })

const chose = (value: unknown) => {
  if (typeof value === 'string') model.value = value
}
</script>

<template>
  <RadioGroupRoot
    data-slot="segmented"
    orientation="horizontal"
    :model-value="model"
    :disabled="disabled"
    :loop="true"
    :class="
      cn(
        'inline-flex items-center gap-px rounded-node border border-rule bg-raised p-px',
        props.class,
      )
    "
    @update:model-value="chose"
  >
    <RadioGroupItem
      v-for="choice in choices"
      :key="choice.id"
      :value="choice.id"
      :class="
        cn(
          'inline-flex h-7 shrink-0 items-center justify-center whitespace-nowrap rounded-node px-3',
          'font-sans text-base font-medium text-ink',
          'cursor-pointer transition-[background-color,color] duration-100 ease-numen',
          'hover:bg-[color-mix(in_oklab,var(--numen-node-bg),var(--numen-node-fg)_8%)]',
          'data-[state=checked]:bg-accent data-[state=checked]:text-accent-ink',
          'data-[state=checked]:hover:bg-[color-mix(in_oklab,var(--numen-focus-bg),var(--numen-focus-fg)_8%)]',
          'outline-none focus-visible:ring-(length:--numen-ring-width) focus-visible:ring-ring',
          'disabled:cursor-not-allowed disabled:opacity-50',
        )
      "
    >
      {{ choice.text }}
    </RadioGroupItem>
  </RadioGroupRoot>
</template>
