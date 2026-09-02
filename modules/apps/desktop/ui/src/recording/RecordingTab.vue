<script setup lang="ts">
/**
 * A recording tab: a player for one recording, and under it the words heard in
 * it.
 *
 * A cue is a place in the recording, so choosing one plays from there. What the
 * tab draws is what `listening` hands it, and nothing is worked out here.
 */
import { ref, watch } from 'vue'
import { WORDS as words } from './words'
import type { Held } from './kind'

const props = defineProps<{ held: Held }>()

/** The player, which stands only while the tab is drawn. */
const player = ref<HTMLAudioElement | null>(null)

/** The lines of the transcript, in the order they are drawn. */
const said = ref<HTMLElement[]>([])

watch(player, (element) => {
  if (!element) return props.held.plays(null)
  props.held.plays({
    seek: (ms) => {
      element.currentTime = ms / 1000
    },
  })
})

// The words move under the recording as it plays: the line being said is
// brought back into view.
watch(
  () => props.held.lines.value.findIndex((line) => line.now),
  (at) => said.value[at]?.scrollIntoView({ block: 'nearest' }),
)

/** Where the player stands now, in the milliseconds the words are counted in. */
const moved = () => props.held.moved((player.value?.currentTime ?? 0) * 1000)

/** The player could not load the recording, and says which failure it was. */
const failed = () => props.held.failed(player.value?.error?.code)
</script>

<template>
  <div class="recording">
    <audio
      v-if="props.held.playing.value"
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
      v-if="props.held.lines.value.length"
      class="recording__transcript"
      :aria-label="words.transcript"
    >
      <li v-for="(line, at) in props.held.lines.value" :key="at" ref="said">
        <button
          type="button"
          class="recording__cue"
          :class="{ 'recording__cue--now': line.now }"
          :aria-current="line.now ? 'true' : undefined"
          @click="props.held.go(line.from)"
        >
          <span class="recording__at">{{ line.at }}</span>
          <span class="recording__text">{{ line.text }}</span>
        </button>
      </li>
    </ol>

    <p v-if="props.held.note.value" class="recording__note">
      {{ props.held.note.value }}
    </p>
  </div>
</template>

<style scoped>
.recording {
  /* The measure the words are read at. */
  --recording-measure: 46rem;
  /* Between the player and the words, and between one cue and the next. */
  --recording-apart: 1rem;
  --recording-near: 0.25rem;
  /* The tab is what scrolls, so the bar stands at the edge of the pane. */
  block-size: 100%;
  overflow-y: auto;
  padding: var(--numen-gutter);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-node-fg);
}

/* Held at the top: the pause stays where a person can reach it. */
.recording__player {
  position: sticky;
  inset-block-start: 0;
  z-index: 1;
  display: block;
  inline-size: 100%;
  max-inline-size: var(--recording-measure);
  margin-inline: auto;
  margin-block-end: var(--recording-apart);
}

/* The words are read in one column, centred in whatever room the pane has. */
.recording__transcript {
  max-inline-size: var(--recording-measure);
  margin: 0 auto;
  padding: 0;
  list-style: none;
}

/* One cue: the moment it was spoken at, and what was said then. The times take
   the room the longest of them needs, so an hour in and a minute in line up. */
.recording__cue {
  display: grid;
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

/* The line being said is drawn in the accent, and carries no fill of its own:
   the pointer is what fills a line. */
.recording__cue--now,
.recording__cue--now .recording__at {
  color: var(--numen-focus-border);
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
