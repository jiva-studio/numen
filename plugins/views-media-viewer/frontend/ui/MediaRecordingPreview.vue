<script setup lang="ts">
import { ref, useTemplateRef } from 'vue'
import { Pause, Play } from '@lucide/vue'
import { clock, doc, getCurrentDate } from '@numen/ui'

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
    previewAudioElement.value.play().catch(() => {
      isPreviewPlaying.value = false
    })
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

  const anchor = doc.createElement('a')
  anchor.href = props.audioUrl
  anchor.download = `recording-${getCurrentDate().toISOString().replace(/[:.]/g, '-')}.webm`
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

      <span class="media-recorder__time">{{ clock(previewPosition) }}</span>
      <span class="media-recorder__time-divider">/</span>
      <span class="media-recorder__time">
        {{ clock(previewDuration || props.elapsedMs) }}
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
  --preview-gap: 0.75rem;
  --preview-padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: var(--preview-gap);
  padding: var(--preview-padding);
  background: var(--numen-field-bg);
  border-radius: var(--numen-radius);
}

.media-recorder__preview-title {
  --preview-title-font-size: 0.9375rem;
  margin: 0;
  font-size: var(--preview-title-font-size);
  font-weight: 600;
}

.media-recorder__preview-controls {
  --preview-controls-gap: 0.75rem;
  display: flex;
  align-items: center;
  gap: var(--preview-controls-gap);
}

.media-recorder__btn {
  --btn-gap: 0.5rem;
  --btn-padding-block: 0.625rem;
  --btn-padding-inline: 1.25rem;
  --btn-font-size: 0.9375rem;
  display: inline-flex;
  align-items: center;
  gap: var(--btn-gap);
  padding: var(--btn-padding-block) var(--btn-padding-inline);
  font-size: var(--btn-font-size);
  font-weight: 600;
  border-radius: var(--numen-radius);
  border: none;
  cursor: pointer;
  transition: all 0.15s ease;
  color: var(--numen-surface);
}

.media-recorder__btn--play {
  --btn-play-inline-size: 2.25rem;
  --btn-play-block-size: 2.25rem;
  inline-size: var(--btn-play-inline-size);
  block-size: var(--btn-play-block-size);
  padding: 0;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: var(--numen-accent);
  color: var(--numen-accent-ink);
}

.media-recorder__icon {
  --icon-inline-size: 1.1rem;
  --icon-block-size: 1.1rem;
  inline-size: var(--icon-inline-size);
  block-size: var(--icon-block-size);
}

.media-recorder__time {
  --time-font-size: 0.8125rem;
  font-size: var(--time-font-size);
  font-variant-numeric: tabular-nums;
  color: var(--numen-hushed);
}

.media-recorder__time-divider {
  color: var(--numen-hushed);
}

.media-recorder__save-btn {
  --save-btn-padding-block: 0.5rem;
  --save-btn-padding-inline: 1rem;
  --save-btn-font-size: 0.8125rem;
  margin-inline-start: auto;
  padding: var(--save-btn-padding-block) var(--save-btn-padding-inline);
  font-size: var(--save-btn-font-size);
  font-weight: 600;
  border-radius: var(--numen-radius);
  border: 1px solid var(--numen-accent);
  background: var(--numen-accent);
  color: var(--numen-accent-ink);
  cursor: pointer;
  transition: opacity 0.15s ease;
}

.media-recorder__save-btn:hover {
  opacity: 0.9;
}
</style>
