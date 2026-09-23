<script setup lang="ts">
/**
 * The one slider of a preset: the goal's curve, drawn as the track a person
 * drags the knob along.
 */
import type { Curve, PresetCounts } from '../../types'
import BacklogPlot from './backlog-plot/BacklogPlot.vue'
import CurveTiles from './CurveTiles.vue'
import { CurveFoot } from './curve-foot'
import { CurvePlot } from './curve-plot'
import { useCurveSlider } from './useCurveSlider'
import './curve-slider.css'

// --- Props & Emits ---
const props = defineProps<{
  curve: Curve
  material: PresetCounts | null
  place: number
  valueText: string
  isWaiting: boolean
}>()

const emit = defineEmits<{
  (event: 'move', place: number): void
  (event: 'settle'): void
}>()

// --- State ---
const state = useCurveSlider(props, emit)
const { isHonest, counts, learning } = state

// --- Handlers ---

// --- Helpers ---
</script>

<template>
  <div class="curve-slider">
    <CurveTiles control="material" :tiles="counts" />

    <div class="curve-slider__island">
      <CurvePlot
        :state="state"
        :goal="props.curve.goal"
        :is-waiting="props.isWaiting"
        :value-text="props.valueText"
      />

      <CurveFoot :state="state" :goal="props.curve.goal" />

      <BacklogPlot :curve="props.curve" :place="props.place" :is-honest="isHonest" />
    </div>

    <CurveTiles control="learned" :tiles="isHonest ? learning : []" />
  </div>
</template>

<style scoped>
.curve-slider__island {
  display: flex;
  flex-direction: column;
  gap: var(--numen-dot-gap);
  padding: var(--numen-box-air);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-raised);
}
</style>
