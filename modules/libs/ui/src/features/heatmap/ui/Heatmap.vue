<script setup lang="ts">
/**
 * What a person did on each day and what is still coming to them, as a grid of
 * weeks.
 *
 * A column is a week: the weeks behind run up to the one they are in, and a few
 * of what is still to come stand after it. How many weeks are drawn is how many
 * fit the room there is.
 *
 * Each cell carries `data-heatmap-day`, the day it stands for.
 */
import { computed, ref, useTemplateRef } from 'vue'

import { Tooltip } from './tooltip'
import type { Box } from '@/shared/lib/place'
import { DaySummary } from './day-summary'
import { getDays, measureGrid, ROWS } from '../lib/heatmap'
import type { Day, Tally } from '../lib/heatmap'
import { useWidth } from '../model/width'
import type { Words } from '../lib/words'

const props = withDefaults(
  defineProps<{
    /** What was answered on each day, by the day it was answered on. */
    did: ReadonlyMap<string, Tally>
    /** How much falls on each day still to come, by the day it falls on. */
    due?: ReadonlyMap<string, number>
    /** What a day's account is called, in the person's own language. */
    words: Words
    /** When it is. */
    now?: Date
    /** How large one cell is drawn, at most, in pixels. */
    cell?: number
    /** How much room stands between two cells, at least. */
    gap?: number
  }>(),
  { due: () => new Map(), now: () => new Date(), cell: 11, gap: 3 },
)

const root = useTemplateRef<HTMLElement>('root')
const room = useWidth(root)

const laid = computed(() =>
  measureGrid({ width: room.value, cell: props.cell, gap: props.gap }),
)
const shown = computed(() => getDays(laid.value.columns, props.now, props.did, props.due))

const step = computed(() => laid.value.cell + laid.value.gap)

const height = computed(() => ROWS * step.value - laid.value.gap)

const xOf = (at: number) => Math.floor(at / ROWS) * step.value
const yOf = (at: number) => (at % ROWS) * step.value

/** The day a person is pointing at, and the cell on the page it is drawn in. */
const pointed = ref<{ day: Day; at: Box } | null>(null)

const setPointed = (day: Day, press: MouseEvent) => {
  const cell = (press.target as SVGRectElement).getBoundingClientRect()
  pointed.value = {
    day,
    at: { x: cell.x, y: cell.y, width: cell.width, height: cell.height },
  }
}
</script>

<template>
  <div ref="root" class="heatmap">
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
        :data-heatmap-day="day.day"
        :data-weight="day.weight"
        :data-ahead="day.ahead ? 'yes' : undefined"
        :data-today="day.today ? 'yes' : undefined"
        @mouseenter="setPointed(day, $event)"
        @mouseleave="pointed = null"
      />
    </svg>

    <Tooltip v-if="pointed" :at="pointed.at">
      <DaySummary :day="pointed.day" :words="words" />
    </Tooltip>
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
  fill: var(--numen-raised);
  stroke: var(--numen-rule);
  stroke-width: 1;
}

.heatmap__day[data-weight='1'] {
  fill: color-mix(in oklab, var(--numen-accent) 25%, var(--numen-raised));
  stroke: none;
}

.heatmap__day[data-weight='2'] {
  fill: color-mix(in oklab, var(--numen-accent) 45%, var(--numen-raised));
  stroke: none;
}

.heatmap__day[data-weight='3'] {
  fill: color-mix(in oklab, var(--numen-accent) 70%, var(--numen-raised));
  stroke: none;
}

.heatmap__day[data-weight='4'] {
  fill: var(--numen-accent);
  stroke: none;
}

/* A day still to come is said quietly: what is done stands at full strength,
   and what is coming is the same weight drawn dim. It carries its weight, so a
   heavy week ahead reads as a heavy week. */
.heatmap__day[data-ahead][data-weight='1'] {
  fill: color-mix(in oklab, var(--numen-accent) 8%, var(--numen-raised));
  stroke: none;
}

.heatmap__day[data-ahead][data-weight='2'] {
  fill: color-mix(in oklab, var(--numen-accent) 15%, var(--numen-raised));
  stroke: none;
}

.heatmap__day[data-ahead][data-weight='3'] {
  fill: color-mix(in oklab, var(--numen-accent) 23%, var(--numen-raised));
  stroke: none;
}

.heatmap__day[data-ahead][data-weight='4'] {
  fill: color-mix(in oklab, var(--numen-accent) 32%, var(--numen-raised));
  stroke: none;
}

/* Today is where a person's eye goes first, so it is ringed whatever it holds. */
.heatmap__day[data-today] {
  stroke: var(--numen-ink);
  stroke-width: 1.5;
}
</style>
