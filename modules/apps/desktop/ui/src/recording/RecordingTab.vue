<script setup lang="ts">
/**
 * A recording tab: a player for one recording, and under it the words heard in
 * it, written as one text.
 *
 * A cue is a place in the recording, so its time in the gutter is clicked to
 * play from there. What the tab draws is what `listening` hands it, and nothing
 * is worked out here.
 */
import { computed, watchPostEffect } from 'vue'
import { LocateFixed } from '@lucide/vue'
import { Editor, Player, timing } from '@numen/ui'
import { WORDS as words } from './words'
import type { Held } from './kind'

const props = defineProps<{ held: Held }>()

/** The times in the editor's gutter, and the line being said. */
const times = timing((line) => props.held.goes(line))

// The words move under the recording as it plays: the line being said is drawn
// in the accent, and following is what brings it back into view.
watchPostEffect(() =>
  times.show({
    times: props.held.times.value,
    current: props.held.current.value,
    following: props.held.following.value && !props.held.typing.value,
  }),
)

/** Whether the view keeps the line being said in sight. */
const follows = computed(() => props.held.following.value)

</script>

<template>
  <div class="recording">
    <div class="recording__head">
      <Player
        v-if="props.held.playable.value"
        class="recording__player"
        :at="props.held.now.value"
        :length="props.held.runs.value"
        :playing="props.held.playing.value"
        :label="words.player"
        @play="props.held.play()"
        @pause="props.held.pause()"
        @seek="props.held.go($event)"
      />
      <p v-else class="recording__note">{{ words.unplayable }}</p>
      <button
        type="button"
        class="recording__follow"
        :aria-label="words.follow"
        :title="words.follow"
        :aria-pressed="follows ? 'true' : 'false'"
        @click="props.held.follows(!follows)"
      >
        <LocateFixed class="recording__icon" />
      </button>
    </div>
    <!-- What went wrong stands above the words, where it is read whether or
         not there are any. -->
    <p v-if="props.held.broken.value" class="recording__note">
      {{ props.held.broken.value }}
    </p>
    <p v-if="props.held.trouble.value" role="alert" class="recording__note">
      {{ props.held.trouble.value }}
    </p>

    <Editor
      v-if="props.held.times.value.length"
      class="recording__transcript"
      :model-value="props.held.prose.value"
      :readonly="!props.held.editable.value"
      :live="false"
      :extensions="times.extension"
      :aria-label="words.transcript"
      @update:model-value="(said: string) => props.held.typed(said)"
      @save="props.held.keep()"
    />

    <p v-if="props.held.note.value" class="recording__note">
      {{ props.held.note.value }}
    </p>
  </div>
</template>

<style scoped>
.recording {
  /* The measure the words are read at. */
  --recording-measure: 46rem;
  /* Between the player and the follow. */
  --recording-apart: 1rem;
  /* What the card standing over the words takes at the top of the pane. */
  --recording-card: calc(2rem + 2 * var(--numen-panel-padding) + 2 * var(--numen-panel-gap));
  /* The tab is what the card is placed in, and the editor fills it. */
  position: relative;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-node-fg);
}

/* The pause and the follow float over the words and stay where a person can
   reach them, whatever the words do underneath. */
.recording__head {
  position: absolute;
  inset-block-start: var(--numen-panel-gap);
  inset-inline: var(--numen-gutter);
  z-index: 1;
  display: flex;
  align-items: center;
  gap: var(--recording-apart);
  max-inline-size: var(--recording-measure);
  margin-inline: auto;
  padding: var(--numen-panel-padding);
  border: var(--numen-stroke) solid var(--numen-panel-border);
  border-radius: var(--numen-radius-panel);
  /* The card is opaque: the words go on under it, and nothing of them shows
     through. */
  background: var(--numen-node-bg);
  box-shadow: var(--numen-shadow-card);
}

.recording__player {
  flex: 1;
  min-inline-size: 0;
}

.recording__follow {
  display: grid;
  place-items: center;
  flex: none;
  inline-size: 2rem;
  block-size: 2rem;
  padding: 0;
  border: 0;
  border-radius: var(--numen-radius-field);
  background: none;
  color: var(--numen-hushed);
  cursor: pointer;
}

.recording__follow:hover {
  background: var(--numen-field-bg);
}

.recording__follow[aria-pressed='true'] {
  color: var(--numen-focus-border);
}

.recording__icon {
  inline-size: 1rem;
  block-size: 1rem;
}

/* The editor scrolls, so the bar stands at the edge of the pane and only the
   lines on screen are drawn. Its own layers stay under the card. */
.recording .recording__transcript {
  flex: 1;
  min-block-size: 0;
  isolation: isolate;
  font-size: inherit;
}

.recording .recording__transcript :deep(.cm-editor) {
  block-size: 100%;
  font-size: inherit;
}

/* The words are read in one column, centred in whatever room the pane has. */
.recording .recording__transcript :deep(.cm-content),
.recording .recording__transcript :deep(.cm-gutters) {
  max-inline-size: var(--recording-measure);
}

/* The first line clears the card, so no word is ever hidden under it. */
.recording .recording__transcript :deep(.cm-scroller) {
  justify-content: center;
  padding-block-start: var(--recording-card);
  padding-inline: var(--numen-gutter);
}

.recording__note {
  flex: none;
  max-inline-size: var(--recording-measure);
  margin: 0 auto;
  padding: 0 var(--numen-gutter) var(--numen-gutter);
  inline-size: 100%;
  color: var(--numen-hushed);
}

/* Whatever stands where the words would begins below the card. */
.recording > .recording__note:first-of-type {
  padding-block-start: var(--recording-card);
}
</style>
