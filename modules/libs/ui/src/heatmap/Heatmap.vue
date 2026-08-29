<script setup lang="ts">
/**
 * What a person did on each day, as a grid of weeks.
 *
 * A column is a week and today stands in the last of them. How many weeks are
 * drawn is how many fit the room there is: a wide window shows more of the year
 * rather than the same weeks drawn larger, and a narrow one shows fewer rather
 * than a grid marooned in the middle of empty room.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

import { days, fits, ROWS } from './heatmap'
import type { Day } from './heatmap'

const props = withDefaults(
  defineProps<{
    /** How much was done on each day, by the day it was done on. */
    did: ReadonlyMap<string, number>
    /** When it is. */
    now?: Date
    /** How large one cell is drawn, at most, in pixels. */
    cell?: number
    /** How much room stands between two cells, at least. */
    gap?: number
  }>(),
  { now: () => new Date(), cell: 11, gap: 3 },
)

const emit = defineEmits<{ (event: 'reaches', day: Day): void }>()

/** The room there is, measured, so the grid is laid out to what it has. */
const held = ref<HTMLElement | null>(null)
const room = ref(0)

let watching: ResizeObserver | null = null

onMounted(() => {
  if (!held.value) return
  room.value = held.value.clientWidth
  if (typeof ResizeObserver === 'undefined') return
  watching = new ResizeObserver(([one]) => {
    room.value = one?.contentRect.width ?? 0
  })
  watching.observe(held.value)
})

onBeforeUnmount(() => {
  watching?.disconnect()
  watching = null
})

const laid = computed(() => fits({ width: room.value, cell: props.cell, gap: props.gap }))
const shown = computed(() => days(laid.value.columns, props.now, props.did))

const step = computed(() => laid.value.cell + laid.value.gap)
const height = computed(() => ROWS * step.value - laid.value.gap)

const xOf = (at: number) => Math.floor(at / ROWS) * step.value
const yOf = (at: number) => (at % ROWS) * step.value

/** What a day says when a person rests on it. */
const told = (day: Day) =>
  day.did > 0 ? `${day.day}: ${day.did} answered` : `${day.day}: nothing answered`

watch(shown, (now) => {
  const today = now.find((one) => one.today)
  if (today) emit('reaches', today)
})
</script>

<template>
  <div ref="held" class="heatmap">
    <svg
      v-if="room > 0"
      class="heatmap__grid"
      :viewBox="`0 0 ${Math.max(1, room)} ${height}`"
      :width="room"
      :height="height"
      role="img"
      aria-label="What was answered on each day"
    >
      <rect
        v-for="(day, at) in shown"
        :key="day.day"
        :x="xOf(at)"
        :y="yOf(at)"
        :width="laid.cell"
        :height="laid.cell"
        :rx="2"
        :ry="2"
        class="heatmap__day"
        :data-weight="day.weight"
        :data-today="day.today ? 'yes' : undefined"
      >
        <title>{{ told(day) }}</title>
      </rect>
    </svg>
  </div>
</template>

<style scoped>
.heatmap {
  inline-size: 100%;
}

.heatmap__grid {
  display: block;
  /* The day standing now is drawn with a ring around it, and a ring is centred
     on the edge it is drawn at, so it stands half outside the grid. */
  overflow: visible;
}

/* A day nobody answered on is the ground the grid is drawn on, and a day
   answered on is the accent, at the weight of what was done. */
.heatmap__day {
  fill: var(--numen-node-bg);
  stroke: var(--numen-node-border);
  stroke-width: 1;
}

.heatmap__day[data-weight='1'] {
  fill: color-mix(in oklab, var(--numen-focus-bg) 25%, var(--numen-node-bg));
  stroke: none;
}

.heatmap__day[data-weight='2'] {
  fill: color-mix(in oklab, var(--numen-focus-bg) 45%, var(--numen-node-bg));
  stroke: none;
}

.heatmap__day[data-weight='3'] {
  fill: color-mix(in oklab, var(--numen-focus-bg) 70%, var(--numen-node-bg));
  stroke: none;
}

.heatmap__day[data-weight='4'] {
  fill: var(--numen-focus-bg);
  stroke: none;
}

/* Today is where a person's eye goes first, so it is ringed whatever it holds. */
.heatmap__day[data-today] {
  stroke: var(--numen-node-fg);
  stroke-width: 1.5;
}
</style>
