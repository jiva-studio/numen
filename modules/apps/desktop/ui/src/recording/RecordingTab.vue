<script setup lang="ts">
/**
 * A recording tab: a player for one recording, and under it the transcript of
 * it, written as one text.
 *
 * A cue is a place in the recording, so its time in the gutter is clicked to
 * play from there. The player stands at the top of the pane on a rule, and
 * everything else is drawn below that rule: what a recording with no transcript
 * offers, what a run says while it goes, and the words themselves.
 *
 * What can be asked over the words already written down is offered in the menu
 * at the end of the strip, and each item stands there only where it applies.
 */
import { computed, ref, watchPostEffect } from 'vue'
import { Ellipsis, LocateFixed } from '@lucide/vue'
import { Editor, Menu, Player, timing } from '@numen/ui'
import type { Point } from '@numen/ui'
import { iconFor } from '../icons'
import { WORDS as words } from './words'
import { DROP, type Held } from './kind'

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

/** Whether this recording has no transcript, which decides what stands below the player. */
const empty = computed(() => props.held.times.value.length === 0)

/** What the menu offers over this recording: each item only where it applies. */
const offered = computed(() =>
  props.held.droppable.value ? [{ id: DROP, text: words.drop }] : [],
)

/** Where the menu was asked for, and nothing while it is not open. */
const asking = ref<{ at: Point; from: HTMLElement } | null>(null)

const asks = (event: Event) => {
  const button = event.currentTarget
  if (!(button instanceof HTMLElement)) return
  const box = button.getBoundingClientRect()
  asking.value = { at: { x: box.left, y: box.bottom }, from: button }
}

const chose = (id: string) => {
  asking.value = null
  if (id === DROP) props.held.drops()
}
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
      <button
        v-if="offered.length"
        type="button"
        class="recording__more"
        :aria-label="words.more"
        :title="words.more"
        aria-haspopup="menu"
        @click="asks"
      >
        <Ellipsis class="recording__icon" />
      </button>
    </div>

    <div class="recording__below">
      <!-- What went wrong stands above the words, where it is read whether or
           not there are any. -->
      <p v-if="props.held.broken.value" class="recording__note">
        {{ props.held.broken.value }}
      </p>
      <p v-if="props.held.trouble.value" role="alert" class="recording__note">
        {{ props.held.trouble.value }}
      </p>

      <!-- The button says there is no transcript, so the note says it only
           where there is no button. -->
      <div v-if="empty" class="recording__silence">
        <p v-if="!props.held.transcribable.value" class="recording__note">
          {{ props.held.note.value }}
        </p>
        <button
          v-else
          type="button"
          class="recording__ask"
          @click="props.held.transcribes()"
        >
          {{ words.transcribe }}
        </button>
      </div>

      <Editor
        v-else
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

    <Menu
      v-if="asking"
      :items="offered"
      :at="asking.at"
      :from="asking.from"
      open
      :name="words.more"
      @choose="chose"
      @dismiss="asking = null"
    >
      <template #icon="{ id }">
        <component
          :is="iconFor(id)"
          v-if="iconFor(id)"
          class="recording__mark"
          aria-hidden="true"
        />
      </template>
    </Menu>
  </div>
</template>

<style scoped>
.recording {
  /* The measure the words are read at. */
  --recording-measure: 46rem;
  /* Between the player and whatever stands beside it. */
  --recording-apart: 1rem;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-node-fg);
}

/* The player heads the pane at its full width, on the rule that separates it
   from what stands below. The strip is one row of controls tall whatever it
   holds, and each control carries its own air around the mark on it. */
.recording__head {
  display: flex;
  align-items: center;
  flex: none;
  gap: var(--recording-apart);
  inline-size: 100%;
  min-block-size: var(--numen-action-size);
  padding-inline: var(--numen-gutter);
  border-block-end: var(--numen-stroke) solid var(--numen-node-border);
}

/* Everything but the player: the words, or what is said where there are none. */
.recording__below {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-block-size: 0;
}

/* With no transcript, what a person can ask for stands in the middle of the
   room the words would have. */
.recording__silence {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--recording-apart);
  margin: auto;
  padding: var(--numen-gutter);
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

/* The two controls at the end of the strip. */
.recording__follow,
.recording__more {
  display: grid;
  place-items: center;
  flex: none;
  inline-size: var(--numen-action-size);
  block-size: var(--numen-action-size);
  padding: 0;
  border: 0;
  border-radius: var(--numen-radius-pill);
  background: none;
  color: var(--numen-hushed);
  cursor: pointer;
}

.recording__follow:hover,
.recording__more:hover {
  background: var(--numen-field-bg);
}

.recording__follow[aria-pressed='true'] {
  color: var(--numen-focus-border);
}

.recording__icon {
  inline-size: 1rem;
  block-size: 1rem;
}

/* The room the menu keeps beside an item for the mark of what it asks for. */
.recording__mark {
  inline-size: 0.875rem;
  block-size: 0.875rem;
}

/* The editor scrolls, so the bar stands at the edge of the pane and only the
   lines on screen are drawn. */
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

/* The words keep clear of the rule the player stands on. */
.recording .recording__transcript :deep(.cm-scroller) {
  justify-content: center;
  padding-block-start: var(--numen-gutter);
  padding-inline: var(--numen-gutter);
}

.recording__note {
  flex: none;
  max-inline-size: var(--recording-measure);
  margin: 0;
  color: var(--numen-hushed);
}

/* What went wrong is read at the measure the words are, above them. */
.recording__below > .recording__note {
  inline-size: 100%;
  margin-inline: auto;
  padding: var(--numen-gutter) var(--numen-gutter) 0;
}
</style>
