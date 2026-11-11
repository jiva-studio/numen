<script setup lang="ts">
/**
 * The backlog plot at the position of the knob. Its axis is days and not the
 * goal's range, and it keeps its height whether or not there is a backlog to
 * draw.
 *
 * It carries the slider's `data-control` for the parts it shares with the plot
 * above; the run itself is drawn by `BacklogChart`.
 */
import { useBacklogPlot } from '../../../model/useBacklogPlot'
import { BACKLOG_PLOT_HEIGHT, PLOT_WIDTH } from '../../../lib/plot'
import type { Curve } from '../../../../lib/presets'
import { WORDS as words } from '../../../lib/words'
import { BacklogChart } from './backlog-chart'
import '../curve-slider.css'

// --- Props & Emits ---
const props = defineProps<{
  curve: Curve
  place: number
  isHonest: boolean
}>()

// --- State ---
const { line, hasBacklog, heights, ends } = useBacklogPlot(props)
const boxStyle = { aspectRatio: `${PLOT_WIDTH} / ${BACKLOG_PLOT_HEIGHT}` }
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
