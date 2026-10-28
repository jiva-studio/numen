<script setup lang="ts">
import { computed } from 'vue'
import { clock } from '@numen/ui'
import { getWaveformBars } from './useMediaRecording'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly elapsedMs: number
  readonly isRecording: boolean
}>()

/* --------------------------------- State ---------------------------------- */
const liveBars = computed(() => getWaveformBars(props.elapsedMs, props.isRecording))
</script>

<template>
  <div class="media-recorder__meter">
    <div class="media-recorder__timer">
      {{ clock(props.elapsedMs) }}
    </div>

    <div class="media-recorder__waveform" aria-label="Microphone input meter">
      <div
        v-for="(height, index) in liveBars"
        :key="index"
        class="media-recorder__waveform-bar"
        :class="{ 'media-recorder__waveform-bar--active': props.isRecording }"
        :style="{ height: `${height}%` }"
      />
    </div>
  </div>
</template>

<style scoped>
.media-recorder__meter {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  padding: 2rem;
  background: var(--numen-field-bg);
  border-radius: var(--numen-radius-card);
}

.media-recorder__timer {
  font-size: 2.5rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.05em;
  color: var(--numen-ink);
}

.media-recorder__waveform {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  inline-size: 100%;
  block-size: 80px;
}

.media-recorder__waveform-bar {
  inline-size: 4px;
  background: var(--numen-rule);
  border-radius: 2px;
  transition:
    height 0.1s ease,
    background-color 0.1s ease;
}

.media-recorder__waveform-bar--active {
  background: var(--numen-danger);
}
</style>
