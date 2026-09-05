<script setup lang="ts">
/**
 * The days of the week as seven chips, each standing at a level. Pressing a day
 * offers the levels, and the day it was chosen for is handed back with it.
 *
 * A chip is filled in step with the level it stands at. A row nobody may turn
 * keeps the keyboard, says so, and offers nothing.
 */
import { computed, ref, type HTMLAttributes } from 'vue'
import { RovingFocusGroup, RovingFocusItem } from 'reka-ui'
import { cn } from '@/classes'
import Menu from '@/menu/Menu.vue'
import type { Point } from '@/lib/geometry'
import { filled, offering, percent, type Day } from './week'

// The row and the levels it offers are two things drawn, so what a caller
// names the row by is put on the row itself.
defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** The days, in the order they are drawn, each at the level it stands at. */
    days: readonly Day[]
    /** The levels offered, in the order they are offered. */
    levels: readonly number[]
    disabled?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { disabled: false },
)

const raises = defineEmits<{
  /** A day put at a level, which is the day as it was given and the level chosen. */
  chooses: [day: string, level: number]
}>()

/** Which day is being given a level, what it stands at, and where its chip is. */
const asking = ref<{ day: string; level: number; at: Point; from: HTMLElement } | null>(null)

const offered = computed(() =>
  offering(props.levels, asking.value?.level ?? null).map((level) => ({
    id: `${level}`,
    text: percent(level),
  })),
)

/** Which of the levels on offer is the one in force, as the menu names it. */
const current = computed(() => (asking.value === null ? null : `${asking.value.level}`))

const asks = (day: Day, event: Event) => {
  if (props.disabled) return
  const chip = event.currentTarget
  if (!(chip instanceof HTMLElement)) return
  const box = chip.getBoundingClientRect()
  asking.value = {
    day: day.id,
    level: filled(day.level),
    at: { x: box.left, y: box.bottom },
    from: chip,
  }
}

const chose = (said: string) => {
  const day = asking.value?.day
  const level = Number(said)
  if (day === undefined || Number.isNaN(level)) return
  raises('chooses', day, level)
}

/**
 * How strongly a day is filled: the whole of the accent at a day standing at
 * the whole of it, and no colour at all at a day standing at nothing.
 */
const filling = (level: number) => {
  const weight = filled(level) * 100
  return {
    background: `color-mix(in oklab, var(--numen-raised), var(--numen-accent) ${weight}%)`,
    color: weight > 50 ? 'var(--numen-accent-ink)' : 'var(--numen-ink)',
  }
}
</script>

<template>
  <RovingFocusGroup
    data-slot="days"
    role="toolbar"
    orientation="horizontal"
    v-bind="$attrs"
    :loop="true"
    :class="cn('inline-flex items-center gap-1', props.class)"
  >
    <RovingFocusItem v-for="day in days" :key="day.id" as-child>
      <button
        type="button"
        :aria-disabled="disabled || undefined"
        :aria-label="`${day.long}, ${percent(day.level)}`"
        aria-haspopup="menu"
        :aria-expanded="asking?.day === day.id"
        :style="filling(day.level)"
        :class="
          cn(
            'inline-flex size-7 shrink-0 items-center justify-center rounded-pill',
            'border border-rule font-sans text-base font-medium',
            'cursor-pointer transition-[background-color,color] duration-hover ease-numen',
            'outline-none ring-numen',
            'aria-disabled:cursor-not-allowed aria-disabled:opacity-50',
          )
        "
        @click="asks(day, $event)"
      >
        {{ day.short }}
      </button>
    </RovingFocusItem>
  </RovingFocusGroup>

  <Menu
    v-if="asking"
    :items="offered"
    :at="asking.at"
    :from="asking.from"
    :current="current"
    open
    opening="keyboard"
    @choose="chose"
    @dismiss="asking = null"
  />
</template>
