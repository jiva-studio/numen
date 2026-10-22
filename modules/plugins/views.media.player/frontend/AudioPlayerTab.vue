<script setup lang="ts">
/**
 * Audio player tab: audio playback controls, interactive waveform
 * visualization, playback speed adjustment, and transcript timestamp navigation.
 */
import { useTemplateRef } from 'vue'
import AudioControls from './AudioControls.vue'
import AudioPlayerHeader from './AudioPlayerHeader.vue'
import AudioTrackBar from './AudioTrackBar.vue'
import AudioTranscript, { type CueItem } from './AudioTranscript.vue'
import { useAudioPlayback, type ExternalPlayerState } from './useAudioPlayback'
import { useAudioTranscript } from './useAudioTranscript'

/* --------------------------------- Props ---------------------------------- */
export type { CueItem, ExternalPlayerState }

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
  'rate-change': [rate: number]
}>()

/* --------------------------------- State ---------------------------------- */
const audioElement = useTemplateRef<HTMLAudioElement>('audioElement')

const {
  isPlaying,
  currentPosition,
  totalDuration,
  playbackRate,
  isMuted,
  onTogglePlay,
  onSeekMs,
  onToggleMute,
} = useAudioPlayback(props, emit, audioElement)

const {
  transcriptCues,
  isTranscribable,
  isDeletable,
  activeCueIndex,
  onTranscribe,
  onDeleteTranscript,
} = useAudioTranscript(props, currentPosition, totalDuration)

/* -------------------------------- Handlers -------------------------------- */
function onRateSelect(rate: number) {
  playbackRate.value = rate
  if (audioElement.value) {
    audioElement.value.playbackRate = rate
  }
  if (props.state?.setPlaybackRate) {
    props.state.setPlaybackRate(rate)
  }
  emit('rate-change', rate)
}

function onResetPosition() {
  onSeekMs(0)
}
</script>

<template>
  <div class="audio-player recording__player" role="region" aria-label="Audio Player">
    <audio ref="audioElement" preload="metadata" />

    <AudioPlayerHeader
      :title="props.title || props.path || 'Audio Track'"
      :current-position="currentPosition"
      :total-duration="totalDuration"
      :playback-rate="playbackRate"
      :is-transcribable="isTranscribable"
      :is-deletable="isDeletable"
      @transcribe="onTranscribe"
      @delete-transcript="onDeleteTranscript"
      @rate-change="onRateSelect"
    />

    <AudioTrackBar
      :current-position="currentPosition"
      :total-duration="totalDuration"
      @seek="onSeekMs"
    />

    <AudioControls
      :is-playing="isPlaying"
      :is-muted="isMuted"
      :current-position="currentPosition"
      :total-duration="totalDuration"
      @toggle-play="onTogglePlay"
      @toggle-mute="onToggleMute"
      @seek="onSeekMs"
      @reset="onResetPosition"
    />

    <AudioTranscript
      v-if="transcriptCues.length > 0"
      :cues="transcriptCues"
      :active-index="activeCueIndex"
      @seek="onSeekMs"
    />

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

.audio-player__empty-transcript {
  padding: 1.5rem;
  text-align: center;
  color: var(--numen-hushed, #9ca3af);
  font-size: 0.875rem;
}
</style>
