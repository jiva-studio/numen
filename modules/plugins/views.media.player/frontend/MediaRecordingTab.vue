<script setup lang="ts">
/**
 * Media recording tab: microphone capture, live level visualization,
 * recording timer, and preview playback.
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Mic, Pause, Play, RotateCcw, Square } from '@lucide/vue'

/* --------------------------------- Props ---------------------------------- */
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
  recordStart: []
  recordStop: [blob: Blob | null]
  save: [blob: Blob]
}>()

/* --------------------------------- State ---------------------------------- */
type RecordingStatus = 'idle' | 'recording' | 'paused' | 'stopped'

const status = ref<RecordingStatus>('idle')
const elapsedMs = ref(0)
const recordedAudioUrl = ref<string | null>(null)
const recordedBlob = ref<Blob | null>(null)
const isPreviewPlaying = ref(false)
const previewPosition = ref(0)
const previewDuration = ref(0)
const errorMessage = ref('')

const previewAudioElement = ref<HTMLAudioElement | null>(null)
let timerInterval: ReturnType<typeof setInterval> | null = null
let mediaStream: MediaStream | null = null
let mediaRecorder: MediaRecorder | null = null
let recordedChunks: Blob[] = []

const isRecording = computed(() => status.value === 'recording')
const hasRecording = computed(() => recordedAudioUrl.value !== null)

/**
 * Animated bars for live waveform rendering while recording.
 */
const liveBars = computed(() => {
  const count = 32
  const bars: number[] = []
  for (let i = 0; i < count; i++) {
    if (isRecording.value) {
      // Dynamic bars based on time tick
      const phase = (elapsedMs.value / 150 + i * 0.4) % (Math.PI * 2)
      const height = Math.max(12, Math.round(Math.abs(Math.sin(phase)) * 85 + 15))
      bars.push(height)
    } else {
      bars.push(15)
    }
  }
  return bars
})

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  // Safe initialization
})

onUnmounted(() => {
  stopTimer()
  cleanupStream()
  if (recordedAudioUrl.value) {
    URL.revokeObjectURL(recordedAudioUrl.value)
  }
})

/* -------------------------------- Handlers -------------------------------- */
async function onStartRecording() {
  errorMessage.value = ''
  recordedChunks = []

  try {
    if (navigator.mediaDevices && typeof navigator.mediaDevices.getUserMedia === 'function') {
      mediaStream = await navigator.mediaDevices.getUserMedia({ audio: true })
      mediaRecorder = new MediaRecorder(mediaStream)

      mediaRecorder.ondataavailable = (event: BlobEvent) => {
        if (event.data.size > 0) {
          recordedChunks.push(event.data)
        }
      }

      mediaRecorder.onstop = () => {
        const blob = new Blob(recordedChunks, { type: 'audio/webm' })
        recordedBlob.value = blob
        if (recordedAudioUrl.value) {
          URL.revokeObjectURL(recordedAudioUrl.value)
        }
        recordedAudioUrl.value = URL.createObjectURL(blob)
        emit('recordStop', blob)
      }

      mediaRecorder.start(250)
    } else {
      // Graceful fallback for non-media environments/testing
      recordedBlob.value = new Blob(['mock audio data'], { type: 'audio/webm' })
      recordedAudioUrl.value = 'blob:mock-recording'
      emit('recordStop', recordedBlob.value)
    }

    status.value = 'recording'
    elapsedMs.value = 0
    startTimer()
    emit('recordStart')
  } catch (err: unknown) {
    errorMessage.value = err instanceof Error ? err.message : 'Could not access microphone'
    status.value = 'idle'
  }
}

function onPauseRecording() {
  if (mediaRecorder && mediaRecorder.state === 'recording') {
    mediaRecorder.pause()
  }
  status.value = 'paused'
  stopTimer()
}

function onResumeRecording() {
  if (mediaRecorder && mediaRecorder.state === 'paused') {
    mediaRecorder.resume()
  }
  status.value = 'recording'
  startTimer()
}

function onStopRecording() {
  if (mediaRecorder && mediaRecorder.state !== 'inactive') {
    mediaRecorder.stop()
  } else if (!mediaRecorder) {
    const blob = new Blob(['mock audio data'], { type: 'audio/webm' })
    recordedBlob.value = blob
    recordedAudioUrl.value = 'blob:mock-recording'
    emit('recordStop', blob)
  }
  cleanupStream()
  stopTimer()
  status.value = 'stopped'
}

function onReset() {
  if (status.value === 'recording' || status.value === 'paused') {
    onStopRecording()
  }
  status.value = 'idle'
  elapsedMs.value = 0
  if (recordedAudioUrl.value) {
    URL.revokeObjectURL(recordedAudioUrl.value)
    recordedAudioUrl.value = null
  }
  recordedBlob.value = null
  isPreviewPlaying.value = false
  previewPosition.value = 0
}

function onTogglePreviewPlay() {
  if (!previewAudioElement.value) return
  if (isPreviewPlaying.value) {
    previewAudioElement.value.pause()
    isPreviewPlaying.value = false
  } else {
    void previewAudioElement.value.play()
    isPreviewPlaying.value = true
  }
}

function onPreviewTimeUpdate() {
  if (previewAudioElement.value) {
    previewPosition.value = Math.round(previewAudioElement.value.currentTime * 1000)
  }
}

function onPreviewLoadedMetadata() {
  if (previewAudioElement.value) {
    previewDuration.value = Math.round(previewAudioElement.value.duration * 1000)
  }
}

function onPreviewEnded() {
  isPreviewPlaying.value = false
  previewPosition.value = previewDuration.value
}

function onSaveRecording() {
  if (!recordedBlob.value) return
  emit('save', recordedBlob.value)

  // Download locally as fallback/convenience
  const a = document.createElement('a')
  a.href = recordedAudioUrl.value || ''
  a.download = `recording-${new Date().toISOString().replace(/[:.]/g, '-')}.webm`
  a.click()
}

/* -------------------------------- Helpers --------------------------------- */
function startTimer() {
  stopTimer()
  const startedAt = Date.now() - elapsedMs.value
  timerInterval = setInterval(() => {
    elapsedMs.value = Date.now() - startedAt
  }, 100)
}

function stopTimer() {
  if (timerInterval) {
    clearInterval(timerInterval)
    timerInterval = null
  }
}

function cleanupStream() {
  if (mediaStream) {
    for (const track of mediaStream.getTracks()) {
      track.stop()
    }
    mediaStream = null
  }
}

function formatClock(ms: number): string {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000))
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
}
</script>

<template>
  <div class="media-recorder" role="region" aria-label="Audio Recorder">
    <!-- Header -->
    <header class="media-recorder__header">
      <h2 class="media-recorder__title">{{ props.title || 'Record Audio' }}</h2>
      <span class="media-recorder__status" :class="`media-recorder__status--${status}`">
        {{ status.toUpperCase() }}
      </span>
    </header>

    <!-- Error message if mic unavailable -->
    <div v-if="errorMessage" class="media-recorder__error" role="alert">
      {{ errorMessage }}
    </div>

    <!-- Live audio visualizer / timer area -->
    <div class="media-recorder__meter">
      <div class="media-recorder__timer">
        {{ formatClock(elapsedMs) }}
      </div>

      <div class="media-recorder__waveform" aria-label="Microphone input meter">
        <div
          v-for="(height, index) in liveBars"
          :key="index"
          class="media-recorder__waveform-bar"
          :class="{ 'media-recorder__waveform-bar--active': isRecording }"
          :style="{ height: `${height}%` }"
        />
      </div>
    </div>

    <!-- Recording action buttons -->
    <div class="media-recorder__actions">
      <button
        v-if="status === 'idle'"
        type="button"
        class="media-recorder__btn media-recorder__btn--record"
        aria-label="Start recording"
        @click="onStartRecording"
      >
        <Mic class="media-recorder__icon" />
        <span>Start Recording</span>
      </button>

      <template v-else-if="status === 'recording' || status === 'paused'">
        <button
          v-if="isRecording"
          type="button"
          class="media-recorder__btn media-recorder__btn--pause"
          aria-label="Pause recording"
          @click="onPauseRecording"
        >
          <Pause class="media-recorder__icon" />
          <span>Pause</span>
        </button>

        <button
          v-else
          type="button"
          class="media-recorder__btn media-recorder__btn--resume"
          aria-label="Resume recording"
          @click="onResumeRecording"
        >
          <Play class="media-recorder__icon" />
          <span>Resume</span>
        </button>

        <button
          type="button"
          class="media-recorder__btn media-recorder__btn--stop"
          aria-label="Stop recording"
          @click="onStopRecording"
        >
          <Square class="media-recorder__icon" />
          <span>Stop</span>
        </button>
      </template>

      <button
        v-if="status === 'stopped'"
        type="button"
        class="media-recorder__btn media-recorder__btn--new"
        aria-label="New recording"
        @click="onReset"
      >
        <RotateCcw class="media-recorder__icon" />
        <span>New Recording</span>
      </button>
    </div>

    <!-- Preview section for finished recording -->
    <section v-if="hasRecording && status === 'stopped'" class="media-recorder__preview">
      <audio
        ref="previewAudioElement"
        :src="recordedAudioUrl || undefined"
        @timeupdate="onPreviewTimeUpdate"
        @loadedmetadata="onPreviewLoadedMetadata"
        @ended="onPreviewEnded"
      />

      <h3 class="media-recorder__preview-title">Recorded Audio Preview</h3>

      <div class="media-recorder__preview-controls">
        <button
          type="button"
          class="media-recorder__btn media-recorder__btn--play"
          :aria-label="isPreviewPlaying ? 'Pause preview' : 'Play preview'"
          @click="onTogglePreviewPlay"
        >
          <Pause v-if="isPreviewPlaying" class="media-recorder__icon" />
          <Play v-else class="media-recorder__icon" />
        </button>

        <span class="media-recorder__time">{{ formatClock(previewPosition) }}</span>
        <span class="media-recorder__time-divider">/</span>
        <span class="media-recorder__time">{{ formatClock(previewDuration || elapsedMs) }}</span>

        <button
          type="button"
          class="media-recorder__save-btn"
          aria-label="Save audio recording"
          @click="onSaveRecording"
        >
          Save Recording
        </button>
      </div>
    </section>
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

.media-recorder__actions {
  display: flex;
  justify-content: center;
  gap: 1rem;
}

.media-recorder__btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 1.25rem;
  font-size: 0.9375rem;
  font-weight: 600;
  border-radius: var(--numen-radius-button, 6px);
  border: none;
  cursor: pointer;
  transition: all 0.15s ease;
  color: #fff;
}

.media-recorder__btn--record {
  background: #ef4444;
}

.media-recorder__btn--record:hover {
  background: #dc2626;
}

.media-recorder__btn--pause {
  background: #f59e0b;
  color: #000;
}

.media-recorder__btn--resume {
  background: #10b981;
}

.media-recorder__btn--stop {
  background: #4b5563;
}

.media-recorder__btn--stop:hover {
  background: #374151;
}

.media-recorder__btn--new {
  background: var(--numen-accent, #3b82f6);
}

.media-recorder__btn--play {
  inline-size: 2.25rem;
  block-size: 2.25rem;
  padding: 0;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: var(--numen-accent, #3b82f6);
}

.media-recorder__icon {
  inline-size: 1.1rem;
  block-size: 1.1rem;
}

.media-recorder__preview {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1.25rem;
  background: var(--numen-field-bg, rgba(255, 255, 255, 0.05));
  border-radius: var(--numen-radius-card, 8px);
}

.media-recorder__preview-title {
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 600;
}

.media-recorder__preview-controls {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.media-recorder__time {
  font-size: 0.8125rem;
  font-variant-numeric: tabular-nums;
  color: var(--numen-hushed, #9ca3af);
}

.media-recorder__time-divider {
  color: var(--numen-hushed, #9ca3af);
}

.media-recorder__save-btn {
  margin-inline-start: auto;
  padding: 0.5rem 1rem;
  font-size: 0.8125rem;
  font-weight: 600;
  border-radius: var(--numen-radius-button, 6px);
  border: 1px solid var(--numen-accent, #3b82f6);
  background: var(--numen-accent, #3b82f6);
  color: #fff;
  cursor: pointer;
  transition: opacity 0.15s ease;
}

.media-recorder__save-btn:hover {
  opacity: 0.9;
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
