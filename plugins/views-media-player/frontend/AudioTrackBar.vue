<script setup lang="ts">
import { computed } from 'vue'
import { formatClock } from './useAudioPlayback'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly currentPosition: number
  readonly totalDuration: number
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  seek: [ms: number]
}>()

/* --------------------------------- State ---------------------------------- */
const waveformBars = computed(() => {
  const barsCount = 64
  const bars: { height: number; timeMs: number }[] = []
  const duration = Math.max(props.totalDuration, 1000)

  for (let i = 0; i < barsCount; i++) {
    const angle = (i / barsCount) * Math.PI * 4
    const wave = Math.abs(Math.sin(angle) * 0.6 + Math.cos(angle * 2.3) * 0.4)
    const height = Math.max(15, Math.round(wave * 100))
    bars.push({
      height,
      timeMs: Math.round((i / barsCount) * duration),
    })
  }
  return bars
})

/* -------------------------------- Handlers -------------------------------- */
function onWaveformClick(event: MouseEvent) {
  const target = event.currentTarget as HTMLElement
  if (!target) return
  const rect = target.getBoundingClientRect()
  const fraction =
    rect && rect.width > 0 ? Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width)) : 0
  const targetMs = Math.round(fraction * props.totalDuration)
  emit('seek', targetMs)
}
</script>

<template>
  <div
    class="audio-player__waveform"
    role="slider"
    aria-label="Audio waveform timeline"
    :aria-valuenow="props.currentPosition"
    :aria-valuemin="0"
    :aria-valuemax="props.totalDuration"
    :aria-valuetext="formatClock(props.currentPosition)"
    tabindex="0"
    @click="onWaveformClick"
  >
    <div
      v-for="(bar, index) in waveformBars"
      :key="index"
      class="audio-player__waveform-bar"
      :class="{ 'audio-player__waveform-bar--played': props.currentPosition >= bar.timeMs }"
      :style="{ height: `${bar.height}%` }"
      :title="formatClock(bar.timeMs)"
    />
  </div>
</template>

<style scoped>
.audio-player__waveform {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 3px;
  inline-size: 100%;
  block-size: 72px;
  padding: 0.5rem 0.75rem;
  background: var(--numen-field-bg, rgba(255, 255, 255, 0.05));
  border-radius: var(--numen-radius-card, 8px);
  cursor: pointer;
  box-sizing: border-box;
}

.audio-player__waveform-bar {
  flex: 1;
  min-inline-size: 2px;
  background: var(--numen-rule, #4b5563);
  border-radius: 2px;
  transition: background-color 0.1s ease;
}

.audio-player__waveform-bar--played {
  background: var(--numen-accent, #3b82f6);
}
</style>
