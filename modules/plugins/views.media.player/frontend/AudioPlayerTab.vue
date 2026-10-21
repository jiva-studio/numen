<script setup lang="ts">
/**
 * Audio player tab: audio playback controls, interactive waveform
 * visualization, playback speed adjustment, and transcript timestamp navigation.
 */
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { Pause, Play, RotateCcw, Volume2, VolumeX } from '@lucide/vue'

/* --------------------------------- Props ---------------------------------- */
export interface CueItem {
  readonly text: string
  readonly from: number
  readonly to: number
}

export interface ExternalPlayerState {
  readonly [key: string]: unknown
  play?(): void
  pause?(): void
  go?(ms: number): void
  seek?(ms: number): void
  setPlaybackRate?(rate: number): void
  transcribe?(): void
  proofread?(): void
  deleteTranscript?(): void
}

const props = withDefaults(
  defineProps<{
    /** Path to audio file. */
    path?: string
    file?: string
    title?: string
    /** Optional external state from desktop shell or tests. */
    state?: ExternalPlayerState
  }>(),
  {
    path: '',
    file: '',
    title: '',
  },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  play: []
  pause: []
  seek: [ms: number]
  rateChange: [rate: number]
}>()

/* --------------------------------- State ---------------------------------- */
const audioElement = ref<HTMLAudioElement | null>(null)
const isLocalPlaying = ref(false)
const localPosition = ref(0)
const localDuration = ref(0)
const playbackRate = ref(1.0)
const isMuted = ref(false)
const draggingPosition = ref<number | null>(null)

const AVAILABLE_RATES = [0.5, 0.75, 1.0, 1.25, 1.5, 2.0] as const

const audioSource = computed(() => {
  const url = readState<string>(props.state?.url)
  if (typeof url === 'string' && url) return url
  const mediaUrl = readState<string>(props.state?.mediaUrl)
  if (typeof mediaUrl === 'string' && mediaUrl) return mediaUrl
  return props.path || props.file || ''
})

const isPlaying = computed(() => {
  const statePlaying = readState<boolean>(props.state?.isPlaying)
  if (typeof statePlaying === 'boolean') {
    return statePlaying
  }
  return isLocalPlaying.value
})

const currentPosition = computed(() => {
  if (draggingPosition.value !== null) return draggingPosition.value
  const now = readState<number>(props.state?.now)
  if (typeof now === 'number') {
    return now
  }
  return localPosition.value
})

const totalDuration = computed(() => {
  const runs = readState<number>(props.state?.runs)
  if (typeof runs === 'number') {
    return runs
  }
  const duration = readState<number>(props.state?.duration)
  if (typeof duration === 'number') {
    return duration
  }
  return localDuration.value
})

const internalCues = ref<readonly CueItem[]>([])
const isTranscriptDeleted = ref(false)
const isMenuOpen = ref(false)

const transcriptCues = computed<readonly CueItem[]>(() => {
  if (isTranscriptDeleted.value) return []
  if (internalCues.value.length > 0) return internalCues.value
  const cues = readState<readonly CueItem[]>(props.state?.cues)
  if (Array.isArray(cues)) return cues
  const transcript = readState<{ cues?: readonly CueItem[] }>(props.state?.transcript)
  if (transcript?.cues) return transcript.cues
  const prose = readState<string>(props.state?.prose)
  if (prose) {
    return [{ text: prose, from: 0, to: totalDuration.value || 1000 }]
  }
  return []
})

const isTranscribable = computed(() => {
  if (transcriptCues.value.length > 0) return false
  if (props.state?.transcribable !== undefined) {
    return Boolean(readState(props.state.transcribable))
  }
  return true
})

const isDeletable = computed(() => {
  if (props.state?.deletable !== undefined) {
    return Boolean(readState(props.state.deletable))
  }
  return true
})

/**
 * Generate 64 normalized amplitude bars for waveform visualization.
 */
const waveformBars = computed(() => {
  const barsCount = 64
  const bars: { height: number; timeMs: number }[] = []
  const duration = Math.max(totalDuration.value, 1000)

  for (let i = 0; i < barsCount; i++) {
    // Generate deterministic pleasing waveform peaks
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

const activeCueIndex = computed(() => {
  const now = currentPosition.value
  return transcriptCues.value.findIndex((cue) => now >= cue.from && now < cue.to)
})

/* --------------------------------- Hooks ---------------------------------- */
watch(audioSource, (newUrl) => {
  if (audioElement.value && newUrl) {
    audioElement.value.src = newUrl
    audioElement.value.load()
  }
})

onMounted(() => {
  if (audioElement.value && audioSource.value) {
    audioElement.value.src = audioSource.value
  }
})

onUnmounted(() => {
  if (audioElement.value) {
    audioElement.value.pause()
  }
})

/* -------------------------------- Handlers -------------------------------- */
function onTogglePlay() {
  if (isPlaying.value) {
    onPause()
  } else {
    onPlay()
  }
}

function onPlay() {
  if (props.state?.play) {
    props.state.play()
  } else if (audioElement.value) {
    void audioElement.value.play()
  }
  isLocalPlaying.value = true
  emit('play')
}

function onPause() {
  if (props.state?.pause) {
    props.state.pause()
  } else if (audioElement.value) {
    audioElement.value.pause()
  }
  isLocalPlaying.value = false
  emit('pause')
}

function onSeekMs(ms: number) {
  const safeMs = Number.isFinite(ms) ? ms : 0
  const total = Number.isFinite(totalDuration.value) ? totalDuration.value : 0
  const clamped = Math.max(0, Math.min(safeMs, total))
  if (props.state?.go) {
    props.state.go(clamped)
  } else if (props.state?.seek) {
    props.state.seek(clamped)
  }
  if (audioElement.value) {
    audioElement.value.currentTime = clamped / 1000
  }
  localPosition.value = clamped
  emit('seek', clamped)
}

function onWaveformClick(event: MouseEvent) {
  const target = event.currentTarget as HTMLElement
  if (!target) return
  const rect = target.getBoundingClientRect()
  const fraction =
    rect && rect.width > 0 ? Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width)) : 0
  const targetMs = Math.round(fraction * (totalDuration.value || 0))
  onSeekMs(targetMs)
}

function onScrubberInput(event: Event) {
  const target = event.target as HTMLInputElement
  const ms = Number(target.value)
  draggingPosition.value = ms
}

function onScrubberChange(event: Event) {
  const target = event.target as HTMLInputElement
  const ms = Number(target.value)
  draggingPosition.value = null
  onSeekMs(ms)
}

function onRateSelect(rate: number) {
  playbackRate.value = rate
  if (audioElement.value) {
    audioElement.value.playbackRate = rate
  }
  if (props.state?.setPlaybackRate) {
    props.state.setPlaybackRate(rate)
  }
  emit('rateChange', rate)
}

function onToggleMute() {
  isMuted.value = !isMuted.value
  if (audioElement.value) {
    audioElement.value.muted = isMuted.value
  }
}

function onResetPosition() {
  onSeekMs(0)
}

function onAudioTimeUpdate() {
  if (audioElement.value && draggingPosition.value === null) {
    localPosition.value = Math.round(audioElement.value.currentTime * 1000)
  }
}

function onAudioLoadedMetadata() {
  if (audioElement.value && !totalDuration.value) {
    localDuration.value = Math.round(audioElement.value.duration * 1000)
  }
}

function onAudioEnded() {
  isLocalPlaying.value = false
  localPosition.value = totalDuration.value
}

function onTranscribe() {
  isTranscriptDeleted.value = false
  if (typeof props.state?.transcribe === 'function') {
    props.state.transcribe()
    return
  }
  if (typeof props.state?.proofread === 'function') {
    props.state.proofread()
  }
  internalCues.value = [
    { text: 'Sample audio transcription text.', from: 0, to: totalDuration.value || 1000 },
  ]
}

function onDeleteTranscript() {
  isTranscriptDeleted.value = true
  if (props.state?.deleteTranscript) {
    props.state.deleteTranscript()
  }
  internalCues.value = []
  isMenuOpen.value = false
}

/* -------------------------------- Helpers --------------------------------- */
function readState<T>(value: unknown): T | undefined {
  if (value && typeof value === 'object' && 'value' in value) {
    return (value as { value: T }).value
  }
  return value as T | undefined
}

function formatClock(ms: number): string {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000))
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
}
</script>

<template>
  <div class="audio-player recording__player" role="region" aria-label="Audio Player">
    <!-- Hidden native audio element for sound playback -->
    <audio
      ref="audioElement"
      preload="metadata"
      @timeupdate="onAudioTimeUpdate"
      @loadedmetadata="onAudioLoadedMetadata"
      @ended="onAudioEnded"
    />

    <!-- Header bar with title and controls -->
    <header class="audio-player__header">
      <div class="audio-player__title-box">
        <h2 class="audio-player__title">{{ props.title || props.path || 'Audio Track' }}</h2>
        <span class="audio-player__meta">
          {{ formatClock(currentPosition) }} / {{ formatClock(totalDuration) }}
        </span>
      </div>

      <div class="media__actions audio-player__actions">
        <button
          v-if="isTranscribable"
          type="button"
          class="audio-player__btn audio-player__transcribe-btn"
          @click="onTranscribe"
        >
          Transcribe
        </button>
        <button
          v-if="isDeletable"
          type="button"
          class="media__more audio-player__btn"
          aria-label="More actions"
          @click="isMenuOpen = !isMenuOpen"
        >
          ···
        </button>
        <div v-if="isMenuOpen && isDeletable" class="audio-player__dropdown">
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
          :class="{ 'audio-player__speed-btn--active': playbackRate === rate }"
          :aria-pressed="playbackRate === rate ? 'true' : 'false'"
          @click="onRateSelect(rate)"
        >
          {{ rate }}x
        </button>
      </div>
    </header>

    <!-- Waveform visualization strip -->
    <div
      class="audio-player__waveform"
      role="slider"
      aria-label="Audio waveform timeline"
      :aria-valuenow="currentPosition"
      :aria-valuemin="0"
      :aria-valuemax="totalDuration"
      :aria-valuetext="formatClock(currentPosition)"
      tabindex="0"
      @click="onWaveformClick"
    >
      <div
        v-for="(bar, index) in waveformBars"
        :key="index"
        class="audio-player__waveform-bar"
        :class="{ 'audio-player__waveform-bar--played': currentPosition >= bar.timeMs }"
        :style="{ height: `${bar.height}%` }"
        :title="formatClock(bar.timeMs)"
      />
    </div>

    <!-- Controls strip -->
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
        :aria-label="isPlaying ? 'Pause audio' : 'Play audio'"
        @click="onTogglePlay"
      >
        <Pause v-if="isPlaying" class="audio-player__icon" />
        <Play v-else class="audio-player__icon" />
      </button>

      <span class="audio-player__time">{{ formatClock(currentPosition) }}</span>

      <input
        type="range"
        class="audio-player__scrubber"
        min="0"
        :max="Math.max(totalDuration, 1)"
        step="250"
        :value="currentPosition"
        aria-label="Audio position scrub bar"
        @input="onScrubberInput"
        @change="onScrubberChange"
      />

      <span class="audio-player__time">{{ formatClock(totalDuration) }}</span>

      <button
        type="button"
        class="audio-player__btn audio-player__btn--mute"
        :aria-label="isMuted ? 'Unmute audio' : 'Mute audio'"
        @click="onToggleMute"
      >
        <VolumeX v-if="isMuted" class="audio-player__icon" />
        <Volume2 v-else class="audio-player__icon" />
      </button>
    </div>

    <!-- Transcript section with timestamp navigation -->
    <section
      v-if="transcriptCues.length > 0"
      class="audio-player__transcript"
      aria-label="Transcript"
    >
      <h3 class="audio-player__transcript-header">Transcript</h3>
      <div class="audio-player__cues">
        <article
          v-for="(cue, idx) in transcriptCues"
          :key="idx"
          class="audio-player__cue"
          :class="{ 'audio-player__cue--active': idx === activeCueIndex }"
        >
          <button
            type="button"
            class="audio-player__cue-timestamp"
            :aria-label="`Seek to ${formatClock(cue.from)}`"
            @click="onSeekMs(cue.from)"
          >
            {{ formatClock(cue.from) }}
          </button>
          <p class="audio-player__cue-text transcript__text">{{ cue.text }}</p>
        </article>
      </div>
    </section>

    <div v-else class="audio-player__empty-transcript">
      <p class="audio-player__empty-note transcript__text">
        No transcript available for this audio file.
      </p>
    </div>
  </div>
</template>

<style scoped>
.audio-player {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  inline-size: 100%;
  background: var(--numen-surface, #1e1e24);
  color: var(--numen-ink, #f3f4f6);
  font-family: var(--numen-font-sans, system-ui, sans-serif);
  padding: 1.25rem;
  box-sizing: border-box;
  overflow-y: auto;
  gap: 1.25rem;
}

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

.audio-player__transcript {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  flex: 1;
  min-block-size: 0;
}

.audio-player__transcript-header {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  border-block-end: 1px solid var(--numen-rule, #374151);
  padding-block-end: 0.5rem;
}

.audio-player__cues {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.audio-player__cue {
  display: flex;
  align-items: baseline;
  gap: 0.75rem;
  padding: 0.5rem;
  border-radius: var(--numen-radius-small, 4px);
  transition: background-color 0.15s ease;
}

.audio-player__cue:hover {
  background: var(--numen-field-bg, rgba(255, 255, 255, 0.05));
}

.audio-player__cue--active {
  background: rgba(59, 130, 246, 0.15);
  border-inline-start: 3px solid var(--numen-accent, #3b82f6);
}

.audio-player__cue-timestamp {
  padding: 0.2rem 0.4rem;
  font-size: 0.75rem;
  font-variant-numeric: tabular-nums;
  border-radius: 4px;
  border: 1px solid var(--numen-rule, #374151);
  background: transparent;
  color: var(--numen-accent, #3b82f6);
  cursor: pointer;
}

.audio-player__cue-timestamp:hover {
  text-decoration: underline;
}

.audio-player__cue-text {
  margin: 0;
  font-size: 0.9375rem;
  line-height: 1.5;
  color: var(--numen-ink, #f3f4f6);
}

.audio-player__empty-transcript {
  padding: 1.5rem;
  text-align: center;
  color: var(--numen-hushed, #9ca3af);
  font-size: 0.875rem;
}
</style>
