<script setup lang="ts">
/**
 * The distribution of similarities between unrelated texts, with a marker at the threshold.
 */
import { computed } from 'vue'
import { Tooltip, useTooltip } from '@numen/ui'
import { describeHistogram, getHistogramBars, getMarkerPercent } from '../model/present'
import type { FloorCalibration } from '../model/types'
import { WORDS } from '../lib/words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  calibration: FloorCalibration
  floor: number
}>()

/* --------------------------------- State ---------------------------------- */
const tip = useTooltip()
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
      data-testid="floor-marker"
      @mouseenter="tip.open"
      @mouseleave="tip.close"
    />
    <Teleport to="body">
      <Tooltip v-if="tip.at.value" :id="tip.id" :at="tip.at.value">
        {{ WORDS.histogramMarker }}
      </Tooltip>
    </Teleport>
  </div>
</template>
