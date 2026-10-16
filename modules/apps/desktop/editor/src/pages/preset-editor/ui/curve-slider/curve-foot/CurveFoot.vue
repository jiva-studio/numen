<script setup lang="ts">
/**
 * What is read under the curve: the value at the knob, the two ends of the
 * grid, and the name of the axis they run along.
 */
import type { Goal } from '../../../types'
import { WORDS as words } from '../../../words'
import type { CurveSliderState } from '../useCurveSlider'
import '../curve-slider.css'

// --- Props & Emits ---
const props = defineProps<{
  state: CurveSliderState
  /** What the curve is scheduled by, which names the axis. */
  goal: Goal
}>()

// --- State ---
const { isHonest, reading, atKnob, atLeast, atMost } = props.state

// --- Handlers ---

// --- Helpers ---
</script>

<template>
  <div class="curve-slider__foot" data-control="foot">
    <p class="curve-slider__under" data-control="under">
      <span
        v-if="isHonest"
        class="curve-slider__number curve-slider__number--knob"
        data-control="number"
        data-at-knob
        :style="reading"
      >
        {{ atKnob }}
      </span>
    </p>

    <p class="curve-slider__ends" data-control="ends">
      <span>{{ isHonest ? atLeast : '' }}</span>
      <span>{{ isHonest ? atMost : '' }}</span>
    </p>

    <p class="curve-slider__name curve-slider__name--x" data-control="name" data-axis="x">
      {{ words.axisX(props.goal) }}
    </p>
  </div>
</template>
