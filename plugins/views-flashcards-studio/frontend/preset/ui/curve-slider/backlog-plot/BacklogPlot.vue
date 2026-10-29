<script setup lang="ts">
/**
 * stands. Its axis is days and not the goal's range, and it keeps its room
 * whether or not there is a backlog to draw.
 *
 * It carries the slider's `data-control` for the parts it shares with the plot
 * above; the run itself is drawn by `BacklogChart`.
 */
import { computed } from 'vue'
import { createBacklogAxisLabels } from '../../../lib/label'
import {
  BACKLOG_PLOT_HEIGHT,
  getBacklogPositions,
  getBacklogExtent,
  createLinePath,
  getBacklogRun,
  PLOT_WIDTH,
  type Extent,
} from '../../../lib/plot'
import type { Curve } from '../../../types'
import { WORDS as words } from '../../../words'
import { BacklogChart } from './backlog-chart'
import '../curve-slider.css'

// --- Props & Emits ---
const props = defineProps<{
  curve: Curve
  place: number
  /** Whether the curve on screen is an answer, and not the waiting for one. */
  isHonest: boolean
}>()

// --- State ---
/** The backlog at the place the knob stands, one figure a day. */
const backlog = computed<readonly number[]>(() => getBacklogRun(props.curve, props.place))

/**
 * The extent it is drawn against, which is the most any place of the curve ever
 * moves, and a place whose run is cut shorter than another's is drawn against
 * the same height as the rest.
 */
const extent = computed<Extent>(() =>
  getBacklogExtent(props.curve.at.flatMap((_, place) => [...getBacklogRun(props.curve, place)])),
)

const positions = computed(() => getBacklogPositions(backlog.value, extent.value))
const line = computed(() => createLinePath(positions.value))

/** Whether there is a backlog to draw at all. */
const hasBacklog = computed(() => props.isHonest && backlog.value.length > 1)

/**
 * The ends of the extent, against the lines they are the height of. Nothing
 * overdue is the foot, so a run holding nothing at all is that one number on
 * the floor it lies along.
 */
const heights = computed(() =>
  createBacklogAxisLabels(extent.value, positions.value, (value) =>
    words.formatBacklogCards(value),
  ),
)

/** The days at either end of the extent, which the grid says nothing about. */
const ends = computed(() => [
  words.formatBacklogDays(1),
  words.formatBacklogDays(backlog.value.length),
])
const boxStyle = { aspectRatio: `${PLOT_WIDTH} / ${BACKLOG_PLOT_HEIGHT}` }

// --- Handlers ---

// --- Helpers ---
</script>

<template>
  <div class="curve-slider__frame">
    <div class="curve-slider__axis">
      <p class="curve-slider__name curve-slider__name--y" data-control="name" data-axis="y">
        {{ words.backlogY }}
      </p>
    </div>

    <div class="curve-slider__over" data-control="over">
      <div class="curve-slider__room" data-control="room" :style="boxStyle">
        <BacklogChart v-if="hasBacklog" :line="line" />
      </div>

      <span
        v-for="(one, at) in hasBacklog ? heights : []"
        :key="at"
        class="curve-slider__number"
        data-control="number"
        :style="one.at"
      >
        {{ one.text }}
      </span>
    </div>
  </div>

  <div class="curve-slider__foot" data-control="foot">
    <p class="curve-slider__ends" data-control="ends">
      <span>{{ hasBacklog ? ends[0] : '' }}</span>
      <span>{{ hasBacklog ? ends[1] : '' }}</span>
    </p>

    <p class="curve-slider__name curve-slider__name--x" data-control="name" data-axis="x">
      {{ words.backlogX }}
    </p>
  </div>
</template>
