<script setup lang="ts">
/**
 * A recording tab: a player for one recording, and under it the words heard in
 * it, written as one text.
 *
 * A cue is a place in the recording, so its time in the gutter is clicked to
 * play from there. What the tab draws is what `listening` hands it, and nothing
 * is worked out here.
 */
import { computed, ref, watch, watchPostEffect } from 'vue'
import { Editor, timing } from '@numen/ui'
import { WORDS as words } from './words'
import type { Held } from './kind'

const props = defineProps<{ held: Held }>()

/** The player, which stands only while the tab is drawn. */
const player = ref<HTMLAudioElement | null>(null)

/** The times in the editor's gutter, and the line being said. */
const heard = timing((line) => props.held.goes(line))

watch(player, (element) => {
  if (!element) return props.held.plays(null)
  props.held.plays({
    seek: (ms) => {
      element.currentTime = ms / 1000
    },
  })
})

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

/** Where the player stands now, in the milliseconds the words are counted in. */
const moved = () => props.held.moved((player.value?.currentTime ?? 0) * 1000)

/** The player could not load the recording, and says which failure it was. */
const failed = () => props.held.failed(player.value?.error?.code)
</script>

<template>
  <div class="recording">
    <div class="recording__head">
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
  block-size: 100%;
  overflow-y: auto;
  padding: var(--numen-gutter);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-node-fg);
}

/* Held at the top: the pause and the follow stay where a person can reach them. */
.recording__head {
  position: sticky;
  inset-block-start: 0;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: var(--recording-apart);
  max-inline-size: var(--recording-measure);
  margin-inline: auto;
  margin-block-end: var(--recording-apart);
  background: var(--numen-panel-bg);
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

/* The words are read in one column, centred in whatever room the pane has. The
   tab owns the scrollbar, so the text stands at its full height. */
.recording .recording__transcript {
  max-inline-size: var(--recording-measure);
  margin: 0 auto;
  block-size: auto;
  min-block-size: 0;
  overflow: visible;
  font-size: inherit;
}

.recording .recording__transcript :deep(.cm-editor) {
  height: auto;
  font-size: inherit;
}

.recording .recording__transcript :deep(.cm-scroller) {
  overflow: visible;
}

.recording__note {
  max-inline-size: var(--recording-measure);
  margin: 0 auto;
  color: var(--numen-hushed);
}
</style>
