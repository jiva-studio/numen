<script setup lang="ts">
import { Mic, Pause, Play, RotateCcw, Square } from '@lucide/vue'
import type { RecordingStatus, RecordAction } from './useMediaRecording'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly status: RecordingStatus
  readonly isRecording: boolean
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  action: [type: RecordAction]
}>()

/* -------------------------------- Handlers -------------------------------- */
function onAction(type: RecordAction) {
  emit('action', type)
}
</script>

<template>
  <div class="media-recorder__actions">
    <button
      v-if="props.status === 'idle'"
      type="button"
      class="media-recorder__btn media-recorder__btn--record"
      aria-label="Start recording"
      @click="onAction('start')"
    >
      <Mic class="media-recorder__icon" />
      <span>Start Recording</span>
    </button>

    <template v-else-if="props.status === 'recording' || props.status === 'paused'">
      <button
        v-if="props.isRecording"
        type="button"
        class="media-recorder__btn media-recorder__btn--pause"
        aria-label="Pause recording"
        @click="onAction('pause')"
      >
        <Pause class="media-recorder__icon" />
        <span>Pause</span>
      </button>

      <button
        v-else
        type="button"
        class="media-recorder__btn media-recorder__btn--resume"
        aria-label="Resume recording"
        @click="onAction('resume')"
      >
        <Play class="media-recorder__icon" />
        <span>Resume</span>
      </button>

      <button
        type="button"
        class="media-recorder__btn media-recorder__btn--stop"
        aria-label="Stop recording"
        @click="onAction('stop')"
      >
        <Square class="media-recorder__icon" />
        <span>Stop</span>
      </button>
    </template>

    <button
      v-if="props.status === 'stopped'"
      type="button"
      class="media-recorder__btn media-recorder__btn--new"
      aria-label="New recording"
      @click="onAction('reset')"
    >
      <RotateCcw class="media-recorder__icon" />
      <span>New Recording</span>
    </button>
  </div>
</template>

<style scoped>
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

.media-recorder__icon {
  inline-size: 1.1rem;
  block-size: 1.1rem;
}
</style>
