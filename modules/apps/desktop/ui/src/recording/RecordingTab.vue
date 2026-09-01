<script setup lang="ts">
/**
 * A recording tab: a player for one recording, and under it the words heard in
 * it.
 *
 * A cue is a place in the recording, so choosing one plays from there. What the
 * recording could not be read as is said where the words would stand.
 */
import { ref, watch } from 'vue'
import { timed } from './listening'
import { WORDS as words } from './words'
import type { Held } from './kind'

const props = defineProps<{ held: Held }>()

/** The player, which stands only while the tab is drawn. */
const player = ref<HTMLAudioElement | null>(null)

watch(player, (element) => {
  if (!element) return props.held.plays(null)
  props.held.plays({
    seek: (ms) => {
      element.currentTime = ms / 1000
    },
  })
})

/** Where the player stands now, in the milliseconds the words are counted in. */
const moved = () => props.held.moved((player.value?.currentTime ?? 0) * 1000)
</script>

<template>
  <div class="recording">
    <audio
      ref="player"
      class="recording__player"
      controls
      preload="metadata"
      :src="props.held.address"
      :aria-label="words.player"
      @timeupdate="moved"
      @seeked="moved"
    />

    <ol
      v-if="props.held.cues.value.length"
      class="recording__said"
      :aria-label="words.transcript"
    >
      <li v-for="(cue, at) in props.held.cues.value" :key="at">
        <button
          type="button"
          class="recording__cue"
          :class="{ 'recording__cue--now': at === props.held.current.value }"
          :aria-current="at === props.held.current.value ? 'true' : undefined"
          @click="props.held.go(cue.from)"
        >
          <span class="recording__at">{{ timed(cue.from) }}</span>
          <span class="recording__text">{{ cue.text }}</span>
        </button>
      </li>
    </ol>

    <p v-if="props.held.trouble.value" class="recording__note">
      {{ props.held.trouble.value }}
    </p>
    <p v-else-if="props.held.working.value" class="recording__note">{{ words.listening }}</p>
    <p v-else-if="!props.held.cues.value.length" class="recording__note">{{ words.silence }}</p>
  </div>
</template>

<style scoped>
.recording {
  /* The measure the words are read at, and the column the times stand in. */
  --recording-measure: 46rem;
  --recording-at: 4.5rem;
  /* Between the player and the words, and between one cue and the next. */
  --recording-apart: 1rem;
  --recording-near: 0.25rem;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  padding: var(--numen-gutter);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-node-fg);
}

.recording__player {
  inline-size: 100%;
  max-inline-size: var(--recording-measure);
  margin-inline: auto;
  margin-block-end: var(--recording-apart);
}

/* The words are read in one column, centred in whatever room the pane has. */
.recording__said {
  flex: 1;
  min-block-size: 0;
  max-inline-size: var(--recording-measure);
  margin: 0 auto;
  padding: 0;
  overflow-y: auto;
  list-style: none;
}

/* One cue: the moment it was spoken at, and what was said then. */
.recording__cue {
  display: grid;
  grid-template-columns: var(--recording-at) 1fr;
  gap: 0 var(--numen-node-gap);
  inline-size: 100%;
  padding: var(--recording-near) var(--numen-node-gap);
  border: none;
  border-radius: var(--numen-radius-field);
  background: none;
  color: inherit;
  font: inherit;
  text-align: start;
}

.recording__cue:hover {
  background: var(--numen-field-bg);
}

.recording__cue:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: var(--numen-stroke);
}

.recording__cue--now {
  background: var(--numen-field-bg);
  font-weight: 600;
}

.recording__at {
  color: var(--numen-hushed);
  font-variant-numeric: tabular-nums;
}

.recording__note {
  max-inline-size: var(--recording-measure);
  margin: 0 auto;
  color: var(--numen-hushed);
}
</style>
