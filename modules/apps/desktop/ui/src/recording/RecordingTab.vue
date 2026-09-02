<script setup lang="ts">
/**
 * A recording tab: a player for one recording, and under it the transcript of
 * it, written as one text.
 *
 * A cue is a place in the recording, so its time in the gutter is clicked to
 * play from there. A recording with no transcript is a pane with the player in
 * the middle of it and the run that writes one down under that.
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

/** Whether this recording has no transcript, which the whole pane is given to. */
const empty = computed(() => props.held.times.value.length === 0)
</script>

<template>
  <div class="recording" :class="{ 'recording--empty': empty }">
    <div
      class="recording__head"
      :class="empty ? 'recording__head--middle' : 'recording__head--card'"
    >
      <div class="recording__controls">
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
          v-if="!empty"
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

      <!-- The button says there is no transcript, so the note says it only
           where there is no button. -->
      <p v-if="empty && !props.held.transcribable.value" class="recording__note">
        {{ props.held.note.value }}
      </p>
      <!-- One button, and which it is the words decide: writing them down where
           there are none, taking them away where there are. -->
      <button
        v-if="props.held.transcribable.value"
        type="button"
        class="recording__ask"
        @click="props.held.transcribes()"
      >
        {{ words.transcribe }}
      </button>
      <button
        v-else-if="props.held.droppable.value"
        type="button"
        class="recording__ask"
        @click="props.held.drops()"
      >
        {{ words.drop }}
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
      v-if="!empty"
      class="recording__transcript"
      :model-value="props.held.prose.value"
      :readonly="!props.held.editable.value"
      :live="false"
      :extensions="times.extension"
      :aria-label="words.transcript"
      @update:model-value="(said: string) => props.held.typed(said)"
      @save="props.held.keep()"
    />
  </div>
</template>

<style scoped>
.recording {
  /* The measure the words are read at. */
  --recording-measure: 46rem;
  /* Between the player and whatever stands beside it or under it. */
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

.recording__head {
  display: flex;
  max-inline-size: var(--recording-measure);
  margin-inline: auto;
}

/* The pause and the follow float over the words and stay where a person can
   reach them, whatever the words do underneath. */
.recording__head--card {
  position: absolute;
  inset-block-start: var(--numen-panel-gap);
  inset-inline: var(--numen-gutter);
  z-index: 1;
  align-items: center;
  gap: var(--recording-apart);
  padding: var(--numen-panel-padding);
  border: var(--numen-stroke) solid var(--numen-panel-border);
  border-radius: var(--numen-radius-panel);
  /* The card is opaque: the words go on under it, and nothing of them shows
     through. */
  background: var(--numen-node-bg);
  box-shadow: var(--numen-shadow-card);
}

/* With no transcript the player is the whole of the pane, and what a person can
   ask for stands under it. */
.recording__head--middle {
  flex-direction: column;
  align-items: center;
  gap: var(--recording-apart);
  inline-size: 100%;
  margin: auto;
  padding: var(--numen-gutter);
}

.recording__controls {
  display: flex;
  align-items: center;
  gap: var(--recording-apart);
  inline-size: 100%;
}

.recording__head--middle .recording__note {
  padding: 0;
  text-align: center;
}

.recording__ask {
  flex: none;
  padding: 0.3rem 0.9rem;
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius-field);
  background: var(--numen-field-bg);
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.recording__ask:hover {
  border-color: var(--numen-focus-border);
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
.recording:not(.recording--empty) > .recording__note:first-of-type {
  padding-block-start: var(--recording-card);
}
</style>
