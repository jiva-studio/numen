<script setup lang="ts">
import { formatClock } from './useAudioPlayback'

/* --------------------------------- Props ---------------------------------- */
export interface CueItem {
  readonly text: string
  readonly from: number
  readonly to: number
}

const props = defineProps<{
  readonly cues: readonly CueItem[]
  readonly activeIndex: number
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  seek: [ms: number]
}>()

/* -------------------------------- Handlers -------------------------------- */
function onSeek(from: number) {
  emit('seek', from)
}
</script>

<template>
  <section class="audio-player__transcript" aria-label="Transcript">
    <h3 class="audio-player__transcript-header">Transcript</h3>
    <div class="audio-player__cues">
      <article
        v-for="(cue, idx) in props.cues"
        :key="idx"
        class="audio-player__cue"
        :class="{ 'audio-player__cue--active': idx === props.activeIndex }"
      >
        <button
          type="button"
          class="audio-player__cue-timestamp"
          :aria-label="`Seek to ${formatClock(cue.from)}`"
          @click="onSeek(cue.from)"
        >
          {{ formatClock(cue.from) }}
        </button>
        <p class="audio-player__cue-text transcript__text">{{ cue.text }}</p>
      </article>
    </div>
  </section>
</template>

<style scoped>
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
</style>
