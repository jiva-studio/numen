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
import { cn } from '@/shared/lib/classes'
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

const onChoose = (value: unknown) => {
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
        'h-action rounded-tight border-rule bg-raised inline-flex items-stretch gap-px border p-px',
        props.class,
      )
    "
    @update:model-value="onChoose"
    @keydown="onKey"
  >
    <RadioGroupItem
      v-for="choice in choices"
      :key="choice.id"
      :value="choice.id"
      :class="
        cn(
          'rounded-tight inline-flex shrink-0 items-center justify-center whitespace-nowrap',
          // A segment fills the height of the row and keeps its own clearance
          // at the ends, which is measured to the ink the screen paints.
          'px-1.5',
          'text-ink font-sans text-base leading-none font-medium',
          'duration-hover ease-numen cursor-pointer transition-[background-color,color]',
          'hover:bg-[color-mix(in_oklab,var(--numen-raised),var(--numen-ink)_8%)]',
          'data-[state=checked]:bg-accent data-[state=checked]:text-accent-ink',
          'data-[state=checked]:hover:bg-[color-mix(in_oklab,var(--numen-accent),var(--numen-accent-ink)_8%)]',
          'ring-numen outline-none',
          'disabled:cursor-not-allowed disabled:opacity-50',
        )
      "
    >
      {{ choice.text }}
    </RadioGroupItem>
  </RadioGroupRoot>
</template>
