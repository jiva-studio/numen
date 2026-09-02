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
import { Editor, Player, timing } from '@numen/ui'
import { WORDS as words } from './words'
import type { Held } from './kind'

const props = defineProps<{ held: Held }>()

/** The times in the editor's gutter, and the line being said. */
const heard = timing((line) => props.held.goes(line))

// The words move under the recording as it plays: the line being said is drawn
// in the accent, and following is what brings it back into view.
watchPostEffect(() =>
  heard.show({
    times: props.held.lines.value.map((line) => line.at),
    now: props.held.current.value,
    follows: props.held.following.value,
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
        :length="props.held.length.value"
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
        :aria-pressed="follows ? 'true' : 'false'"
        @click="props.held.follows(!follows)"
      >
        {{ words.follow }}
      </button>
    </div>
    <p v-if="props.held.broken.value" class="recording__note">
      {{ props.held.broken.value }}
    </p>

    <Editor
      v-if="props.held.lines.value.length"
      class="recording__transcript"
      :model-value="props.held.prose.value"
      :readonly="!props.held.editable.value"
      :live="false"
      :extensions="heard.extension"
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
  /* Between the player and the words. */
  --recording-apart: 1rem;
  /* The tab is what scrolls, so the bar stands at the edge of the pane. */
  /* The words scroll and the player does not, so the two are laid out one
     above the other and only the words are given the room that is left. */
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-node-fg);
}

/* The pause and the follow stay where a person can reach them. */
.recording__head {
  display: flex;
  flex: none;
  align-items: center;
  gap: var(--recording-apart);
  max-inline-size: var(--recording-measure);
  margin-inline: auto;
  padding: var(--numen-gutter) var(--numen-gutter) var(--recording-apart);
  inline-size: 100%;
}

.recording__player {
  flex: 1;
  min-inline-size: 0;
}

.recording__follow {
  padding: 0;
  border: 0;
  background: none;
  color: var(--numen-hushed);
  font: inherit;
  text-decoration: underline;
  text-underline-offset: 0.15em;
  cursor: pointer;
}

.recording__follow[aria-pressed='true'] {
  color: var(--numen-focus-border);
}

/* The editor scrolls, so the bar stands at the edge of the pane and only the
   lines on screen are drawn. */
.recording .recording__transcript {
  flex: 1;
  min-block-size: 0;
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

.recording .recording__transcript :deep(.cm-scroller) {
  justify-content: center;
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
</style>
