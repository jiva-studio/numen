<script setup lang="ts">
import { ref, useTemplateRef } from 'vue'
import { Pause, Play } from '@lucide/vue'
import { formatClock } from './useAudioPlayback'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly audioUrl: string
  readonly blob: Blob | null
  readonly elapsedMs: number
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  save: [blob: Blob]
}>()

/* --------------------------------- State ---------------------------------- */
const isPreviewPlaying = ref(false)
const previewPosition = ref(0)
const previewDuration = ref(0)
const previewAudioElement = useTemplateRef<HTMLAudioElement>('previewAudioElement')

/* -------------------------------- Handlers -------------------------------- */
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
  if (!props.blob) return
  emit('save', props.blob)

  const anchor = document.createElement('a')
  anchor.href = props.audioUrl
  anchor.download = `recording-${new Date().toISOString().replace(/[:.]/g, '-')}.webm`
  anchor.click()
}
</script>

<template>
  <section class="media-recorder__preview">
    <audio
      ref="previewAudioElement"
      :src="props.audioUrl"
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
      <span class="media-recorder__time">
        {{ formatClock(previewDuration || props.elapsedMs) }}
      </span>

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
</template>

<style scoped>
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
</style>
