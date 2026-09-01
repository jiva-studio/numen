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

/** The player could not load the recording, and says which failure it was. */
const failed = () => props.held.failed(player.value?.error?.code)
</script>

<template>
  <div class="recording">
    <audio
      v-if="props.held.playable && props.held.address.value"
      ref="player"
      class="recording__player"
      controls
      preload="none"
      :src="props.held.address.value"
      :aria-label="words.player"
      @timeupdate="moved"
      @seeked="moved"
      @error="failed"
    />
    <p v-else class="recording__note">{{ words.unplayable }}</p>
    <p v-if="props.held.broken.value" class="recording__note">
      {{ props.held.broken.value }}
    </p>

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
    <p v-else-if="props.held.working.value" class="recording__note">{{ words.transcribing }}</p>
    <p v-else-if="!props.held.cues.value.length" class="recording__note">{{ words.silence }}</p>
  </div>
</template>

<style scoped>
.recording {
  /* The measure the words are read at. */
  --recording-measure: 46rem;
  /* Between the player and the words, and between one cue and the next. */
  --recording-apart: 1rem;
  --recording-near: 0.25rem;
  /* The tab is what scrolls, so the bar stands at the edge of the pane and not
     beside the words. */
  block-size: 100%;
  overflow-y: auto;
  padding: var(--numen-gutter);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-node-fg);
}

.recording__player {
  /* Held at the top: an hour of words scrolls past, and the pause is where it
     was left. */
  position: sticky;
  inset-block-start: 0;
  z-index: 1;
  display: block;
  inline-size: 100%;
  max-inline-size: var(--recording-measure);
  margin-inline: auto;
  margin-block-end: var(--recording-apart);
  background: var(--numen-node-bg);
}

/* The words are read in one column, centred in whatever room the pane has. */
.recording__said {
  max-inline-size: var(--recording-measure);
  margin: 0 auto;
  padding: 0;
  list-style: none;
}

/* One cue: the moment it was spoken at, and what was said then. The one being
   said is lit and not thickened: a line that changes weight moves the words
   under it. */
.recording__cue {
  display: grid;
  /* The times take the room the longest of them needs and no more, so an hour
     in and a minute in line up without a column set by hand. */
  grid-template-columns: max-content 1fr;
  gap: 0 var(--recording-near);
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


.recording__cue--now {
  background: var(--numen-field-bg);
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
