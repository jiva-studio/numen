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
  --meter-gap: 1rem;
  --meter-padding: 2rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--meter-gap);
  padding: var(--meter-padding);
  background: var(--numen-field-bg);
  border-radius: var(--numen-radius);
}

.media-recorder__timer {
  --timer-font-size: 2.5rem;
  --timer-letter-spacing: 0.05em;
  font-size: var(--timer-font-size);
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  letter-spacing: var(--timer-letter-spacing);
  color: var(--numen-ink);
}

.media-recorder__waveform {
  --waveform-gap: 0.25rem;
  --waveform-block-size: 5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--waveform-gap);
  inline-size: 100%;
  block-size: var(--waveform-block-size);
}

.media-recorder__waveform-bar {
  --waveform-bar-inline-size: 0.25rem;
  --waveform-bar-border-radius: 0.125rem;
  inline-size: var(--waveform-bar-inline-size);
  background: var(--numen-rule);
  border-radius: var(--waveform-bar-border-radius);
  transition:
    height 0.1s ease,
    background-color 0.1s ease;
}

.media-recorder__waveform-bar--active {
  background: var(--numen-alarm);
}
</style>
