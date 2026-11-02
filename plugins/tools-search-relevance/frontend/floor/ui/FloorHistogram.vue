<script setup lang="ts">
/**
 * The distribution of similarities between unrelated texts, with a marker at the threshold.
 */
import { computed } from 'vue'
import { describeHistogram, getHistogramBars, getMarkerPercent } from '../model/present'
import type { FloorCalibration } from '../model/types'
import { WORDS } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  calibration: FloorCalibration
  /** The threshold in force. */
  floor: number
}>()

/* --------------------------------- State ---------------------------------- */
const bars = computed(() => getHistogramBars(props.calibration.histogram, props.floor))
const markerPercent = computed(() => getMarkerPercent(props.floor))
const description = computed(() => describeHistogram(props.calibration, props.floor))
</script>

<template>
  <div
    role="img"
    :aria-label="description"
    class="relative h-12 w-full"
    data-testid="floor-histogram"
  >
    <div class="flex h-full items-end gap-px">
      <div
        v-for="(bar, index) in bars"
        :key="index"
        class="flex-1 rounded-t-sm"
        :class="bar.isAbove ? 'bg-accent' : 'bg-hushed'"
        :style="{ height: `${bar.heightPercent}%` }"
        data-testid="floor-bar"
      />
    </div>
    <div
      class="bg-ink absolute inset-y-0 w-px"
      :style="{ left: `${markerPercent}%` }"
      :title="WORDS.histogramMarker"
      data-testid="floor-marker"
    />
  </div>
</template>
