<script setup lang="ts">
/**
 * Two to four choices side by side, one of them chosen. The chosen segment is
 * filled with the accent and the rest are quiet.
 *
 * The whole control is one stop on the way round the screen, and the arrow
 * keys move between the segments and choose as they go. Home and End go to
 * the ends.
 */
import type { HTMLAttributes } from 'vue'
import { RadioGroupItem, RadioGroupRoot } from 'reka-ui'
import { cn } from '@/classes'
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

/** Home and End go to the ends of the row, and the choice follows the keyboard. */
const onKey = (event: KeyboardEvent) => {
  if (props.disabled) return
  const choice =
    event.key === 'Home'
      ? props.choices[0]
      : event.key === 'End'
        ? props.choices[props.choices.length - 1]
        : undefined
  if (choice) model.value = choice.id
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
        // One row tall, which every control standing on a row is drawn at.
        'inline-flex h-action items-stretch gap-px rounded-tight border border-rule bg-raised p-px',
        props.class,
      )
    "
    @update:model-value="chose"
    @keydown="onKey"
  >
    <RadioGroupItem
      v-for="choice in choices"
      :key="choice.id"
      :value="choice.id"
      :class="
        cn(
          'inline-flex shrink-0 items-center justify-center whitespace-nowrap rounded-tight',
          // A segment fills the height of the row and keeps its own clearance
          // at the ends, which is measured to the ink the screen paints.
          'px-1.5',
          'font-sans text-base font-medium leading-none text-ink',
          'cursor-pointer transition-[background-color,color] duration-hover ease-numen',
          'hover:bg-[color-mix(in_oklab,var(--numen-node-bg),var(--numen-node-fg)_8%)]',
          'data-[state=checked]:bg-accent data-[state=checked]:text-accent-ink',
          'data-[state=checked]:hover:bg-[color-mix(in_oklab,var(--numen-focus-bg),var(--numen-focus-fg)_8%)]',
          'outline-none ring-numen',
          'disabled:cursor-not-allowed disabled:opacity-50',
        )
      "
    >
      {{ choice.text }}
    </RadioGroupItem>
  </RadioGroupRoot>
</template>
