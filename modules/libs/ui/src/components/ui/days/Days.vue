<script setup lang="ts">
/**
 * The days of the week as seven chips, each on or off on its own. A chip that
 * is on is filled with the accent.
 *
 * The row is one stop on the way round the screen; the arrow keys move along
 * it and the space bar turns a day on or off. Each chip says its whole name to
 * a screen reader and draws a letter or two.
 */
import type { HTMLAttributes } from 'vue'
import { ToggleGroupItem, ToggleGroupRoot } from 'reka-ui'
import { cn } from '@/lib/utils'
import { lit, WEEK, type Day } from './week'

const props = withDefaults(
  defineProps<{
    /** The days, in the order they are drawn. */
    days?: readonly Day[]
    disabled?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { days: () => WEEK, disabled: false },
)

/** The days that are on, by the identifiers the caller gave them. */
const model = defineModel<readonly string[]>({ default: () => [] })

const chose = (value: unknown) => {
  const among = Array.isArray(value) ? value.filter((one) => typeof one === 'string') : []
  model.value = lit(among, props.days)
}
</script>

<template>
  <ToggleGroupRoot
    type="multiple"
    data-slot="days"
    orientation="horizontal"
    :model-value="[...model]"
    :disabled="disabled"
    :loop="true"
    :class="cn('inline-flex items-center gap-1', props.class)"
    @update:model-value="chose"
  >
    <ToggleGroupItem
      v-for="day in days"
      :key="day.id"
      :value="day.id"
      :aria-label="day.long"
      :class="
        cn(
          'inline-flex size-7 shrink-0 items-center justify-center rounded-pill',
          'border border-rule bg-raised font-sans text-base font-medium text-ink',
          'cursor-pointer transition-[background-color,color] duration-100 ease-numen',
          'hover:bg-[color-mix(in_oklab,var(--numen-node-bg),var(--numen-node-fg)_8%)]',
          'data-[state=on]:border-transparent data-[state=on]:bg-accent data-[state=on]:text-accent-ink',
          'data-[state=on]:hover:bg-[color-mix(in_oklab,var(--numen-focus-bg),var(--numen-focus-fg)_8%)]',
          'outline-none focus-visible:ring-(length:--numen-ring-width) focus-visible:ring-ring',
          'disabled:cursor-not-allowed disabled:opacity-50',
        )
      "
    >
      {{ day.short }}
    </ToggleGroupItem>
  </ToggleGroupRoot>
</template>
