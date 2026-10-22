<script setup lang="ts">
import { computed, ref } from 'vue'
import { Pause, Play, RotateCcw, Volume2, VolumeX } from '@lucide/vue'
import { formatClock } from './useAudioPlayback'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly isPlaying: boolean
  readonly isMuted: boolean
  readonly currentPosition: number
  readonly totalDuration: number
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  'toggle-play': []
  'toggle-mute': []
  seek: [ms: number]
  reset: []
}>()

/* --------------------------------- State ---------------------------------- */
const draggingPosition = ref<number | null>(null)

const displayPosition = computed(() => {
  if (draggingPosition.value !== null) return draggingPosition.value
  return props.currentPosition
})

/* -------------------------------- Handlers -------------------------------- */
function onTogglePlay() {
  emit('toggle-play')
}

function onToggleMute() {
  emit('toggle-mute')
}

function onResetPosition() {
  emit('reset')
}

function onScrubberInput(event: Event) {
  const target = event.target as HTMLInputElement
  draggingPosition.value = Number(target.value)
}

function onScrubberChange(event: Event) {
  const target = event.target as HTMLInputElement
  const ms = Number(target.value)
  draggingPosition.value = null
  emit('seek', ms)
}
</script>

<template>
  <div class="audio-player__controls">
    <button
      type="button"
      class="audio-player__btn audio-player__btn--reset"
      aria-label="Reset audio to start"
      @click="onResetPosition"
    >
      <RotateCcw class="audio-player__icon" />
    </button>

    <button
      type="button"
      class="audio-player__btn audio-player__btn--play"
      :aria-label="props.isPlaying ? 'Pause audio' : 'Play audio'"
      @click="onTogglePlay"
    >
      <Pause v-if="props.isPlaying" class="audio-player__icon" />
      <Play v-else class="audio-player__icon" />
    </button>

    <span class="audio-player__time">{{ formatClock(displayPosition) }}</span>

    <input
      type="range"
      class="audio-player__scrubber"
      min="0"
      :max="Math.max(props.totalDuration, 1)"
      step="250"
      :value="displayPosition"
      aria-label="Audio position scrub bar"
      @input="onScrubberInput"
      @change="onScrubberChange"
    />

    <span class="audio-player__time">{{ formatClock(props.totalDuration) }}</span>

    <button
      type="button"
      class="audio-player__btn audio-player__btn--mute"
      :aria-label="props.isMuted ? 'Unmute audio' : 'Mute audio'"
      @click="onToggleMute"
    >
      <VolumeX v-if="props.isMuted" class="audio-player__icon" />
      <Volume2 v-else class="audio-player__icon" />
    </button>
  </div>
</template>

<style scoped>
.audio-player__controls {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  inline-size: 100%;
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

.audio-player__btn--play {
  inline-size: 2.75rem;
  block-size: 2.75rem;
  background: var(--numen-accent, #3b82f6);
  color: #fff;
}

.audio-player__btn--play:hover {
  filter: brightness(1.1);
}

.audio-player__icon {
  inline-size: 1.1rem;
  block-size: 1.1rem;
}

.audio-player__time {
  font-size: 0.8125rem;
  color: var(--numen-hushed, #9ca3af);
  font-variant-numeric: tabular-nums;
  min-inline-size: 4ch;
  text-align: center;
}

.audio-player__scrubber {
  flex: 1;
  min-inline-size: 0;
  accent-color: var(--numen-accent, #3b82f6);
  cursor: pointer;
}
</style>
