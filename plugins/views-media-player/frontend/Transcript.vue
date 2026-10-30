<script setup lang="ts">
/**
 * The transcript under the player, with the time of each line, or an empty-state
 * message when there is none. Clicking a time in the gutter plays from there.
 */
import { computed, watch } from 'vue'
import { LoadingIndicator } from '@numen/ui'
import { createTimeGutter } from './lib/timeGutter'
import TranscriptText from './TranscriptText.vue'
import { WORDS as words } from './words'
import type { TranscriptState } from './model/transcript'

/* ----------------------------- Props & Emits ------------------------------ */
const props = withDefaults(
  defineProps<{
    state: TranscriptState
    isPlaying?: boolean | undefined
  }>(),
  { isPlaying: undefined },
)

/* --------------------------------- State ---------------------------------- */
const { failure, isPlaying: playbackIsPlaying } = props.state.playback
const { current, following, note, isLoading, error, isEditable } = props.state.content
const { prose, typing } = props.state.draft
const { timed, times: cueTimes, hasText } = props.state.lines
const { isStartable, starter } = props.state.offers
const isPlaying = computed(() => props.isPlaying ?? playbackIsPlaying.value)

/** The times in the editor's gutter, and the line being played. */
const times = createTimeGutter((line) => props.state.navigation.goToLine(line))

/* --------------------------------- Hooks ---------------------------------- */
watch(
  [cueTimes, current, following, isPlaying, typing],
  ([timesValue, currentLine, isFollowing, isPlayingValue, isTyping]) => {
    times.show({
      times: timesValue,
      current: currentLine,
      isFollowing: isFollowing && isPlayingValue && !isTyping,
    })
  },
  { immediate: true, flush: 'post' },
)

watch(
  following,
  (isFollowing, wasFollowing) => {
    if (isFollowing && !wasFollowing && !isPlaying.value && !typing.value) {
      times.centerCurrent()
    }
  },
  { flush: 'post' },
)

/* -------------------------------- Handlers -------------------------------- */
function onTranscribe() {
  props.state.offers.start().catch(props.state.reportFailure)
}

function onUpdateModelValue(text: string) {
  props.state.draft.setProse(text)
}

function onSave() {
  props.state.draft.keep().catch(props.state.reportFailure)
}
</script>

<template>
  <div class="transcript">
    <p v-if="failure" class="transcript__note">
      {{ failure }}
    </p>
    <p v-if="error" role="alert" class="transcript__note">
      {{ error }}
    </p>

    <LoadingIndicator v-if="isLoading" :label="words.loading" />

    <div v-else-if="!hasText" class="transcript__silence">
      <p v-if="!isStartable" class="transcript__note">
        {{ note }}
      </p>
      <button v-else type="button" class="transcript__ask" @click="onTranscribe">
        {{ starter?.title }}
      </button>
    </div>

    <TranscriptText
      v-else
      class="transcript__text"
      :model-value="prose"
      :is-editable="isEditable"
      :label="words.transcript"
      :extensions="timed ? times.extension : []"
      @update:model-value="onUpdateModelValue"
      @save="onSave"
    />
  </div>
</template>

<style scoped>
.transcript {
  --transcript-measure: 46rem;

  display: flex;
  flex-direction: column;
  flex: 1;
  min-block-size: 0;
}

/* An empty transcript centres its note or its Transcribe button. */
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

.transcript .transcript__text {
  --editor-measure: var(--transcript-measure);
  --editor-lead: var(--numen-inset);
  --editor-margin: var(--numen-gutter);

  flex: 1;
  min-block-size: 0;
  isolation: isolate;
  font-size: inherit;
}

.transcript__note {
  flex: none;
  inline-size: 100%;
  max-inline-size: var(--transcript-measure);
  margin: 0;
  padding: var(--numen-gutter) var(--numen-gutter) 0;
  color: var(--numen-hushed);
}
</style>
