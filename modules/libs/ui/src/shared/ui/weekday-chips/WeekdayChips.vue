<script setup lang="ts">
/**
 * The days of the week as seven chips, each standing at a level. Pressing a day
 * offers the levels, and the day it was chosen for is handed back with it.
 *
 * keeps the keyboard, says so, and offers nothing.
 */
import { computed, ref, type HTMLAttributes } from 'vue'
import { RovingFocusGroup, RovingFocusItem } from 'reka-ui'
import { cn } from '@/shared/lib/dom/classes'
import { Menu } from '../menu'
import type { Position } from '@/shared/lib/math/geometry'
import { getFill, getFillPercent, getOfferedLevels, type Day } from './week'

defineOptions({ inheritAttrs: false })

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    days: readonly Day[]
    /** The levels offered, in the order they are offered. */
    levels: readonly number[]
    isDisabled?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { isDisabled: false },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** A day put at a level, which is the day as it was given and the level chosen. */
  choose: [day: string, level: number]
}>()

/* ---------------------------------- State --------------------------------- */
const popoverAnchor = ref<{ day: string; level: number; at: Position; from: HTMLElement } | null>(
  null,
)

const availableLevels = computed(() =>
  getOfferedLevels(props.levels, popoverAnchor.value?.level ?? null).map((level) => ({
    id: `${level}`,
    text: getFillPercent(level),
  })),
)

/** Which of the levels on offer is the one in force, as the menu names it. */
const current = computed(() =>
  popoverAnchor.value === null ? null : `${popoverAnchor.value.level}`,
)

/* -------------------------------- Handlers -------------------------------- */
const onOpenLevels = (day: Day, event: Event) => {
  if (props.isDisabled) return
  const chip = event.currentTarget
  if (!(chip instanceof HTMLElement)) return
  const box = chip.getBoundingClientRect()
  popoverAnchor.value = {
    day: day.id,
    level: getFill(day.level),
    at: { x: box.left, y: box.bottom },
    from: chip,
  }
}

const onChoose = (id: string) => {
  const day = popoverAnchor.value?.day
  const level = Number(id)
  if (day === undefined || Number.isNaN(level)) return
  emit('choose', day, level)
}

/* --------------------------------- Helpers -------------------------------- */
/**
 * How strongly a day is filled: the whole of the accent at a day standing at
 * the whole of it, and no colour at all at a day standing at nothing.
 */
const getFillStyle = (level: number) => {
  const weight = getFill(level) * 100
  return {
    background: `color-mix(in oklab, var(--numen-raised), var(--numen-accent) ${weight}%)`,
    color: weight > 50 ? 'var(--numen-accent-ink)' : 'var(--numen-ink)',
  }
}
</script>

<template>
  <RovingFocusGroup
    data-slot="weekday-chips"
    role="toolbar"
    orientation="horizontal"
    v-bind="$attrs"
    loop
    :class="cn('inline-flex items-center gap-1', props.class)"
  >
    <RovingFocusItem v-for="day in days" :key="day.id" as-child>
      <button
        type="button"
        :aria-disabled="isDisabled || undefined"
        :aria-label="`${day.long}, ${getFillPercent(day.level)}`"
        aria-haspopup="menu"
        :aria-expanded="popoverAnchor?.day === day.id"
        :style="getFillStyle(day.level)"
        :class="
          cn(
            'rounded-pill inline-flex size-7 shrink-0 items-center justify-center',
            'border-rule border font-sans text-base font-medium',
            'duration-hover ease-numen cursor-pointer transition-[background-color,color]',
            'ring-numen outline-none',
            'aria-disabled:cursor-not-allowed aria-disabled:opacity-50',
          )
        "
        @click="onOpenLevels(day, $event)"
      >
        {{ day.short }}
      </button>
    </RovingFocusItem>
  </RovingFocusGroup>

  <Menu
    v-if="popoverAnchor"
    :items="availableLevels"
    :at="popoverAnchor.at"
    :from="popoverAnchor.from"
    :current="current"
    is-open
    trigger="keyboard"
    @choose="onChoose"
    @dismiss="popoverAnchor = null"
  />
</template>
