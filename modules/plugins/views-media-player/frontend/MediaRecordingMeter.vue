<script setup lang="ts">
import { computed } from 'vue'
import { formatClock } from './useAudioPlayback'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly elapsedMs: number
  readonly isRecording: boolean
}>()

/* --------------------------------- State ---------------------------------- */
const liveBars = computed(() => {
  const count = 32
  const bars: number[] = []
  for (let i = 0; i < count; i++) {
    if (props.isRecording) {
      const phase = (props.elapsedMs / 150 + i * 0.4) % (Math.PI * 2)
      const height = Math.max(12, Math.round(Math.abs(Math.sin(phase)) * 85 + 15))
      bars.push(height)
    } else {
      bars.push(15)
    }
  }
  return bars
})
</script>

<template>
  <div class="media-recorder__meter">
    <div class="media-recorder__timer">
      {{ formatClock(props.elapsedMs) }}
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
  background: var(--numen-field-bg, rgba(255, 255, 255, 0.05));
  border-radius: var(--numen-radius-card, 8px);
}

.media-recorder__timer {
  font-size: 2.5rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.05em;
  color: var(--numen-ink, #f3f4f6);
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
  background: var(--numen-rule, #4b5563);
  border-radius: 2px;
  transition:
    height 0.1s ease,
    background-color 0.1s ease;
}

.media-recorder__waveform-bar--active {
  background: #ef4444;
}
</style>
