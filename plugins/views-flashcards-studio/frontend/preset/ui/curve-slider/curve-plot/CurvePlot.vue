<script setup lang="ts">
/**
 * The area the curve is drawn in, with the axis it is read against, the labels
 * and heights beside it, and the callout at the knob.
 */
import { PLOT_HEIGHT, PLOT_WIDTH } from '../../../lib/plot'
import type { Goal } from '../../../types'
import { WORDS as words } from '../../../words'
import type { CurveSliderState } from '../useCurveSlider'
import { CurveCallout } from './curve-callout'
import { CurveChart } from './curve-chart'
import { CurveWaiting } from './curve-waiting'
import '../curve-slider.css'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  state: CurveSliderState
  /** What the curve is scheduled by, which names both axes. */
  goal: Goal
  /** Whether an answer is being waited for. */
  isWaiting: boolean
  valueText: string
}>()

/* --------------------------------- State ---------------------------------- */
const { isHonest, labels } = props.state.annotations
const { named, heights } = labels
const boxStyle = { aspectRatio: `${PLOT_WIDTH} / ${PLOT_HEIGHT}` }
</script>

<template>
  <div class="curve-slider__frame">
    <div class="curve-slider__axis">
      <p class="curve-slider__name curve-slider__name--y" data-control="name" data-axis="y">
        {{ words.axisY(props.goal) }}
      </p>
    </div>

    <div class="curve-slider__over" data-control="over">
      <div class="curve-slider__room" data-control="room" :style="boxStyle">
        <CurveWaiting v-if="!isHonest && props.isWaiting" />

        <CurveChart v-else-if="isHonest" :state="props.state" :value-text="props.valueText" />
      </div>

      <span
        v-for="one in named"
        :key="one.key"
        class="curve-slider__label"
        data-control="label"
        :style="one.at"
      >
        {{ one.text }}
      </span>

      <span
        v-for="(one, at) in isHonest ? heights : []"
        :key="at"
        class="curve-slider__number"
        data-control="number"
        :style="one.at"
      >
        {{ one.text }}
      </span>

      <CurveCallout :state="props.state" />
    </div>
  </div>
</template>
