<script setup lang="ts">
/**
 * What stands overdue at the end of each day ahead, at the place the knob
 * stands. Its axis is days and not the goal's range, and it keeps its room
 * whether or not there is a backlog to draw.
 *
 * It carries the slider's `data-control` for the parts it shares with the plot
 * above, and `data-backlog` for the two of its own: `picture` and `line`.
 */
import { computed } from 'vue'
import {
  against,
  BACKLOG_HIGH,
  BACKLOG_PLOT,
  backlogPositionsOf,
  clearAt,
  extentOfBacklog,
  LEFT,
  lineOf,
  RIGHT,
  runAt,
  WIDE,
  type Extent,
} from '../../../lib/plot'
import type { Curve } from '../../../types'
import { WORDS as words } from '../../../words'
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
  const said = (value: number) => words.backlogHeightAt(value)
  const fits = (y: number, lift: string, value: number) =>
    clearAt(y, positions.value, [])
      ? [{ at: against(y, lift, BACKLOG_PLOT.high), text: said(value) }]
      : []
  if (most === least) {
    return [{ at: against(BACKLOG_PLOT.foot, '0', BACKLOG_PLOT.high), text: said(least) }]
  }
  return [
    ...fits(BACKLOG_PLOT.top, '-100%', most),
    ...fits(BACKLOG_PLOT.foot, '0', least),
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
        <svg
          v-if="drawn"
          class="curve-slider__picture backlog__picture"
          data-backlog="picture"
          aria-hidden="true"
          :viewBox="`0 0 ${WIDE} ${BACKLOG_HIGH}`"
        >
          <!-- The foot is nothing overdue, which is what the extent is read up from. -->
          <line
            class="curve-slider__rule"
            data-control="rule"
            :x1="LEFT"
            :x2="LEFT"
            :y1="BACKLOG_PLOT.top"
            :y2="BACKLOG_PLOT.foot"
          />
          <line
            class="curve-slider__rule"
            data-control="rule"
            :x1="LEFT"
            :x2="RIGHT"
            :y1="BACKLOG_PLOT.foot"
            :y2="BACKLOG_PLOT.foot"
          />

          <path class="backlog__line" data-backlog="line" :d="line" />
        </svg>
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

<style scoped>
/* The backlog is read and not dragged, so no pointer is offered over it. */
.backlog__picture {
  cursor: default;
}

.backlog__line {
  fill: none;
  stroke: var(--numen-caution-fg);
  stroke-width: 1.75;
  stroke-linecap: round;
  stroke-linejoin: round;
}
</style>
