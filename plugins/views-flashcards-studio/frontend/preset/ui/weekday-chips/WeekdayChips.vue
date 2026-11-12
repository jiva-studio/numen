<script setup lang="ts">
/**
 * The days of the week as seven chips, each at a level. Pressing a day
 * opens the levels, and the day it was chosen for is returned with the choice.
 *
 * A disabled row keeps keyboard focus, is announced as disabled, and opens no menu.
 */
import { computed, ref, type HTMLAttributes } from 'vue'
import { RovingFocusGroup, RovingFocusItem } from 'reka-ui'
import { cn, measureRect } from '@numen/ui'
import { Menu } from '@numen/ui'
import type { Position } from '@numen/ui'
import { getFill, getFillPercent, getMenuLevels, type Day } from './week'

defineOptions({ inheritAttrs: false })

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    days: readonly Day[]
    levels: readonly number[]
    isDisabled?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { isDisabled: false, class: undefined },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  choose: [day: string, level: number]
}>()

/* ---------------------------------- State --------------------------------- */
/**
 * The most of the accent a day short of the whole is tinted with. Past it the
 * ink on the chip falls below 4.5:1 on either set of tokens, and the accent's
 * own ink does not reach it until the chip is nearly all accent, so a day
 * between is a tint under the ink and only the whole of it is the accent.
 */
const MOST_TINT = 45

const popoverAnchor = ref<{ day: string; level: number; at: Position; from: HTMLElement } | null>(
  null,
)

const availableLevels = computed(() =>
  getMenuLevels(props.levels, popoverAnchor.value?.level ?? null).map((level) => ({
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
  const box = measureRect(chip)
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
const getFillStyle = (level: number) => {
  const fill = getFill(level)
  const weight = fill >= 1 ? 100 : Math.round(fill * MOST_TINT)
  return {
    background: `color-mix(in oklab, var(--numen-raised), var(--numen-accent) ${weight}%)`,
    color: fill >= 1 ? 'var(--numen-accent-ink)' : 'var(--numen-ink)',
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
