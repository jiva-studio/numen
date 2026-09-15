<script setup lang="ts">
/**
 * What the place the knob stands at buys, said beside it, with the tail that
 * points from the words back to the knob.
 */
import type { CurveSliderState } from '../../useCurveSlider'
import '../../curve-slider.css'

// --- Props & Emits ---
const props = defineProps<{ state: CurveSliderState }>()

// --- State ---
const { callout } = props.state

// --- Handlers ---

// --- Helpers ---
</script>

<template>
  <span v-if="callout" class="curve-slider__callout" data-control="callout" :style="callout.at">
    <span
      v-for="one in callout.lines"
      :key="one"
      class="curve-slider__bought"
      data-control="bought"
    >
      {{ one }}
    </span>
  </span>

  <span
    v-if="callout"
    class="curve-slider__tail"
    data-control="tail"
    :data-under="callout.under || undefined"
    :class="{ 'curve-slider__tail--under': callout.under }"
    :style="callout.tail"
  />
</template>
