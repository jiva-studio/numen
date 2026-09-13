<script setup lang="ts">
/**
 * What stands overdue at the end of each day ahead, at the place the knob
 * stands. Its axis is days and not the goal's range, and it keeps its room
 * whether or not there is a backlog to draw.
 *
 * It carries the slider's `data-control` for the parts it shares with the plot
 * above; the run itself is drawn by `BacklogPicture`.
 */
import { computed } from 'vue'
import { clearAt, getAxisNumber } from '../../../lib/label'
import {
  BACKLOG_HIGH,
  BACKLOG_PLOT,
  backlogPositionsOf,
  extentOfBacklog,
  lineOf,
  runAt,
  WIDE,
  type Extent,
} from '../../../lib/plot'
import type { Curve } from '../../../types'
import { WORDS as words } from '../../../words'
import { BacklogPicture } from './backlog-picture'
import '../curve-slider.css'

// --- Props & Emits ---
const props = defineProps<{
  curve: Curve
  /** The place of the grid the knob stands at. */
  place: number
  /** Whether the curve on screen is an answer, and not the waiting for one. */
  honest: boolean
}>()

// --- State ---
/** The backlog at the place the knob stands, one figure a day. */
const backlog = computed<readonly number[]>(() => runAt(props.curve, props.place))

/**
 * The extent it is drawn against, which is the most any place of the curve ever
 * stands at. One extent for every place keeps the picture still while the knob
 * moves, and a place whose run is cut shorter than another's is drawn against
 * the same height as the rest.
 */
const extent = computed<Extent>(() =>
  extentOfBacklog(props.curve.at.flatMap((_, place) => [...runAt(props.curve, place)])),
)

const positions = computed(() => backlogPositionsOf(backlog.value, extent.value))
const line = computed(() => lineOf(positions.value))

/** Whether there is a backlog to draw at all. */
const drawn = computed(() => props.honest && backlog.value.length > 1)

/**
 * The ends of the extent, against the lines they are the height of. Nothing
 * overdue is the foot, so a run holding nothing at all is that one number on
 * the floor it lies along.
 */
const heights = computed(() => {
  const { least, most } = extent.value
  const formatValue = (value: number) => words.backlogHeightAt(value)
  const heightsAt = (y: number, lift: string, value: number) =>
    clearAt(y, positions.value, [])
      ? [{ at: getAxisNumber(y, lift, BACKLOG_PLOT.high), text: formatValue(value) }]
      : []
  if (most === least) {
    return [
      { at: getAxisNumber(BACKLOG_PLOT.foot, '0', BACKLOG_PLOT.high), text: formatValue(least) },
    ]
  }
  return [
    ...heightsAt(BACKLOG_PLOT.top, '-100%', most),
    ...heightsAt(BACKLOG_PLOT.foot, '0', least),
  ]
})

/** The days at either end of the extent, which the grid says nothing about. */
const ends = computed(() => [words.backlogWidthAt(1), words.backlogWidthAt(backlog.value.length)])

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
      <div
        class="curve-slider__room"
        data-control="room"
        :style="{ aspectRatio: `${WIDE} / ${BACKLOG_HIGH}` }"
      >
        <BacklogPicture v-if="drawn" :line="line" />
      </div>

      <span
        v-for="(one, at) in drawn ? heights : []"
        :key="at"
        class="curve-slider__number"
        data-control="number"
        :style="one.at"
        >{{ one.text }}</span
      >
    </div>
  </div>

  <div class="curve-slider__foot" data-control="foot">
    <p class="curve-slider__ends" data-control="ends">
      <span>{{ drawn ? ends[0] : '' }}</span>
      <span>{{ drawn ? ends[1] : '' }}</span>
    </p>

    <p class="curve-slider__name curve-slider__name--x" data-control="name" data-axis="x">
      {{ words.backlogX }}
    </p>
  </div>
</template>
