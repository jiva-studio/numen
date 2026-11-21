<script setup lang="ts">
/**
 * Media recording tab: microphone capture, live level visualization,
 * recording timer, and preview playback.
 */
import type { RecorderTab } from '../api/saveRecording'
import MediaRecordingActions from './MediaRecordingActions.vue'
import MediaRecordingMeter from './MediaRecordingMeter.vue'
import MediaRecordingPreview from './MediaRecordingPreview.vue'
import { useMediaRecording } from '../model/useMediaRecording'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly title?: string
  readonly state?: RecorderTab
}>()

/* --------------------------------- State ---------------------------------- */
const {
  status,
  elapsedMs,
  bars,
  recordedAudioUrl,
  errorMessage,
  isRecording,
  onRecordAction,
  save,
} = useMediaRecording(() => props.state)
</script>

<template>
  <div class="media-recorder" role="region" aria-label="Audio Recorder">
    <header class="media-recorder__header">
      <h2 class="media-recorder__title">{{ props.title || 'Record Audio' }}</h2>
      <span class="media-recorder__status" :class="`media-recorder__status--${status}`">
        {{ status.toUpperCase() }}
      </span>
    </header>

    <div v-if="errorMessage" class="media-recorder__error" role="alert">
      {{ errorMessage }}
    </div>

    <MediaRecordingMeter :bars="bars" :elapsed-ms="elapsedMs" :is-recording="isRecording" />

    <MediaRecordingActions :status="status" :is-recording="isRecording" @action="onRecordAction" />

    <MediaRecordingPreview
      v-if="recordedAudioUrl && status === 'stopped'"
      :audio-url="recordedAudioUrl"
      :can-retry="Boolean(errorMessage)"
      :elapsed-ms="elapsedMs"
      @save="save"
    />
  </div>
</template>

<style scoped>
.media-recorder {
  --media-recorder-padding: 1.5rem;
  --media-recorder-gap: 1.5rem;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  inline-size: 100%;
  background: var(--numen-surface);
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  padding: var(--media-recorder-padding);
  box-sizing: border-box;
  overflow-y: auto;
  gap: var(--media-recorder-gap);
}

.media-recorder__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.media-recorder__title {
  --title-font-size: var(--numen-title-size);
  margin: 0;
  font-size: var(--title-font-size);
  font-weight: 600;
}

.media-recorder__status {
  --status-padding-block: 0.25rem;
  --status-padding-inline: 0.6rem;
  --status-font-size: var(--numen-font-size);
  padding: var(--status-padding-block) var(--status-padding-inline);
  font-size: var(--status-font-size);
  font-weight: 700;
  border-radius: var(--numen-radius-pill);
  background: var(--numen-field-bg);
}

.media-recorder__status--recording {
  background: var(--numen-alarm);
  color: var(--numen-surface);
  animation: pulse 1.5s infinite;
}

.media-recorder__status--paused {
  background: var(--numen-caution-bg);
  color: var(--numen-ink);
}

.media-recorder__status--stopped {
  background: var(--numen-accent);
  color: var(--numen-accent-ink);
}

.media-recorder__error {
  --error-padding-block: 0.75rem;
  --error-padding-inline: 1rem;
  --error-border-radius: 0.375rem;
  --error-font-size: var(--numen-font-size);
  padding: var(--error-padding-block) var(--error-padding-inline);
  background: var(--numen-alarm-bg);
  border: 1px solid var(--numen-alarm);
  border-radius: var(--error-border-radius);
  color: var(--numen-alarm);
  font-size: var(--error-font-size);
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.6;
  }
}
</style>
