<script setup lang="ts">
/**
 * Media recording tab: microphone capture, live level visualization,
 * recording timer, and preview playback.
 */
import MediaRecordingActions from './MediaRecordingActions.vue'
import MediaRecordingMeter from './MediaRecordingMeter.vue'
import MediaRecordingPreview from './MediaRecordingPreview.vue'
import { useMediaRecording, type RecordingStatus } from './useMediaRecording'

/* --------------------------------- Props ---------------------------------- */
export type { RecordingStatus }

const props = withDefaults(
  defineProps<{
    at?: string
    path?: string
    title?: string
    state?: Record<string, unknown>
  }>(),
  {
    at: '',
    path: '',
    title: 'Record Audio',
    state: () => ({}),
  },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  'record-start': []
  'record-stop': [blob: Blob | null]
  save: [blob: Blob]
}>()

/* --------------------------------- State ---------------------------------- */
const {
  status,
  elapsedMs,
  recordedAudioUrl,
  recordedBlob,
  errorMessage,
  isRecording,
  onRecordAction,
} = useMediaRecording(emit)

/* -------------------------------- Handlers -------------------------------- */
function onSave(blob: Blob) {
  emit('save', blob)
}
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

    <MediaRecordingMeter :elapsed-ms="elapsedMs" :is-recording="isRecording" />

    <MediaRecordingActions :status="status" :is-recording="isRecording" @action="onRecordAction" />

    <MediaRecordingPreview
      v-if="recordedAudioUrl !== null && status === 'stopped'"
      :audio-url="recordedAudioUrl"
      :blob="recordedBlob"
      :elapsed-ms="elapsedMs"
      @save="onSave"
    />
  </div>
</template>

<style scoped>
.media-recorder {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  inline-size: 100%;
  background: var(--numen-surface, #1e1e24);
  color: var(--numen-ink, #f3f4f6);
  font-family: var(--numen-font-sans, system-ui, sans-serif);
  padding: 1.5rem;
  box-sizing: border-box;
  overflow-y: auto;
  gap: 1.5rem;
}

.media-recorder__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.media-recorder__title {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 600;
}

.media-recorder__status {
  padding: 0.25rem 0.6rem;
  font-size: 0.75rem;
  font-weight: 700;
  border-radius: 9999px;
  background: var(--numen-field-bg, rgba(255, 255, 255, 0.1));
}

.media-recorder__status--recording {
  background: #ef4444;
  color: #fff;
  animation: pulse 1.5s infinite;
}

.media-recorder__status--paused {
  background: #f59e0b;
  color: #000;
}

.media-recorder__status--stopped {
  background: #10b981;
  color: #fff;
}

.media-recorder__error {
  padding: 0.75rem 1rem;
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid #ef4444;
  border-radius: 6px;
  color: #fca5a5;
  font-size: 0.875rem;
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
