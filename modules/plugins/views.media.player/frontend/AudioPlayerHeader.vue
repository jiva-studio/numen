<script setup lang="ts">
import { ref } from 'vue'
import { formatClock } from './useAudioPlayback'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly title: string
  readonly currentPosition: number
  readonly totalDuration: number
  readonly playbackRate: number
  readonly isTranscribable: boolean
  readonly isDeletable: boolean
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  transcribe: []
  'delete-transcript': []
  'rate-change': [rate: number]
}>()

/* --------------------------------- State ---------------------------------- */
const isMenuOpen = ref(false)
const AVAILABLE_RATES = [0.5, 0.75, 1.0, 1.25, 1.5, 2.0] as const

/* -------------------------------- Handlers -------------------------------- */
function onTranscribe() {
  emit('transcribe')
}

function onDeleteTranscript() {
  isMenuOpen.value = false
  emit('delete-transcript')
}

function onToggleMenu() {
  isMenuOpen.value = !isMenuOpen.value
}

function onRateSelect(rate: number) {
  emit('rate-change', rate)
}
</script>

<template>
  <header class="audio-player__header">
    <div class="audio-player__title-box">
      <h2 class="audio-player__title">{{ props.title }}</h2>
      <span class="audio-player__meta">
        {{ formatClock(props.currentPosition) }} / {{ formatClock(props.totalDuration) }}
      </span>
    </div>

    <div class="media__actions audio-player__actions">
      <button
        v-if="props.isTranscribable"
        type="button"
        class="audio-player__btn audio-player__transcribe-btn"
        @click="onTranscribe"
      >
        Transcribe
      </button>
      <button
        v-if="props.isDeletable"
        type="button"
        class="media__more audio-player__btn"
        aria-label="More actions"
        @click="onToggleMenu"
      >
        ···
      </button>
      <div v-if="isMenuOpen && props.isDeletable" class="audio-player__dropdown">
        <button type="button" class="audio-player__dropdown-item" @click="onDeleteTranscript">
          Delete the transcript
        </button>
      </div>
    </div>

    <div class="audio-player__speed-selector" role="group" aria-label="Playback speed">
      <span class="audio-player__speed-label">Speed:</span>
      <button
        v-for="rate in AVAILABLE_RATES"
        :key="rate"
        type="button"
        class="audio-player__speed-btn"
        :class="{ 'audio-player__speed-btn--active': props.playbackRate === rate }"
        :aria-pressed="props.playbackRate === rate ? 'true' : 'false'"
        @click="onRateSelect(rate)"
      >
        {{ rate }}x
      </button>
    </div>
  </header>
</template>

<style scoped>
.audio-player__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
}

.audio-player__title-box {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.audio-player__title {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 600;
}

.audio-player__meta {
  font-size: 0.875rem;
  color: var(--numen-hushed, #9ca3af);
  font-variant-numeric: tabular-nums;
}

.audio-player__speed-selector {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.audio-player__speed-label {
  font-size: 0.8125rem;
  color: var(--numen-hushed, #9ca3af);
  margin-inline-end: 0.25rem;
}

.audio-player__speed-btn {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  border-radius: var(--numen-radius-small, 4px);
  border: 1px solid var(--numen-rule, #374151);
  background: transparent;
  color: inherit;
  cursor: pointer;
  transition: all 0.15s ease;
}

.audio-player__speed-btn--active {
  background: var(--numen-accent, #3b82f6);
  border-color: var(--numen-accent, #3b82f6);
  color: #fff;
}

.audio-player__btn {
  display: grid;
  place-items: center;
  inline-size: 2.25rem;
  block-size: 2.25rem;
  border-radius: 50%;
  border: none;
  background: var(--numen-field-bg, rgba(255, 255, 255, 0.1));
  color: inherit;
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.audio-player__btn:hover {
  background: var(--numen-rule, rgba(255, 255, 255, 0.2));
}

.audio-player__transcribe-btn {
  inline-size: auto;
  block-size: auto;
  padding: 0.4rem 0.8rem;
  border-radius: 4px;
}

.audio-player__dropdown {
  position: absolute;
  background: var(--numen-surface, #1e1e24);
  border: 1px solid var(--numen-rule, #374151);
  border-radius: 4px;
  padding: 0.25rem;
  z-index: 10;
}

.audio-player__dropdown-item {
  background: none;
  border: none;
  color: inherit;
  padding: 0.5rem 0.75rem;
  cursor: pointer;
  font-size: 0.875rem;
  width: 100%;
  text-align: left;
}

.audio-player__dropdown-item:hover {
  background: var(--numen-hover, rgba(255, 255, 255, 0.05));
}
</style>
