<script setup lang="ts">
/**
 * The text under a player: the words with the times they were said at, or the
 * prose a page is written around, or what stands where there is neither.
 *
 * A cue is a place in what is played, so its time in the gutter is clicked to
 * play from there. Prose carries no times, so it is read with no gutter and
 * nothing to seek.
 */
import { computed, watchPostEffect } from 'vue'
import { Editor, timing } from '@numen/ui'
import { WORDS as words } from '../words'
import type { MediaTabState } from '../kind'

// --- Props & Emits ---
const props = defineProps<{ state: MediaTabState }>()

// --- State ---
const isEditable = computed(() => props.state.isEditable.value)

const {
  broken,
  current,
  following,
  note,
  prose,
  timed,
  times: cues,
  transcribable,
  error,
  typing,
  written,
} = props.state

/** The times in the editor's gutter, and the line being said. */
const times = timing((line) => props.state.goToLine(line))

// The words move under what is played: the line being said is drawn in the
// accent, and following is what brings it back into view.
watchPostEffect(() =>
  times.show({
    times: cues.value,
    current: current.value,
    following: following.value && !typing.value,
  }),
)

// --- Handlers ---
function onTranscribe() {
  props.state.transcribe()
}

function onUpdateModelValue(text: string) {
  props.state.setProse(text)
}

function onSave() {
  void props.state.keep()
}
</script>

<template>
  <div class="transcript">
    <!-- What went wrong stands above the text, where it is read whether or not
         there is any. -->
    <p v-if="broken" class="transcript__note">
      {{ broken }}
    </p>
    <p v-if="error" role="alert" class="transcript__note">
      {{ error }}
    </p>

    <!-- The button says there is nothing here, so the note says it only where
         there is no button. -->
    <div v-if="!written" class="transcript__silence">
      <p v-if="!transcribable" class="transcript__note">
        {{ note }}
      </p>
      <button v-else type="button" class="transcript__ask" @click="onTranscribe">
        {{ words.transcribe }}
      </button>
    </div>

    <Editor
      v-else
      class="transcript__text"
      :model-value="prose"
      :readonly="!isEditable"
      :extensions="timed ? times.extension : []"
      :aria-label="words.transcript"
      @update:model-value="onUpdateModelValue"
      @save="onSave"
    />
  </div>
</template>

<style scoped>
.transcript {
  /* The measure the text is read at. */
  --transcript-measure: 46rem;

  display: flex;
  flex-direction: column;
  flex: 1;
  min-block-size: 0;
}

/* With nothing here, what a person can ask for stands in the middle of the room
   the text would have. */
.transcript__silence {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  margin: auto;
  padding: var(--numen-gutter);
  text-align: center;
}

.transcript__ask {
  flex: none;
  padding: 0.3rem 0.9rem;
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius-field);
  background: var(--numen-field-bg);
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.transcript__ask:hover {
  background: var(--numen-field-bg-hover);
}

/* The text is read in one column, clear of the rule the player stands on, and
   the editor scrolls so the bar stands at the edge of the pane. */
.transcript .transcript__text {
  --editor-measure: var(--transcript-measure);
  --editor-lead: var(--numen-inset);
  --editor-margin: var(--numen-gutter);

  flex: 1;
  min-block-size: 0;
  isolation: isolate;
  font-size: inherit;
}

/* What went wrong is read at the measure the text is, above it. */
.transcript__note {
  flex: none;
  inline-size: 100%;
  max-inline-size: var(--transcript-measure);
  margin: 0;
  padding: var(--numen-gutter) var(--numen-gutter) 0;
  color: var(--numen-hushed);
}
</style>
