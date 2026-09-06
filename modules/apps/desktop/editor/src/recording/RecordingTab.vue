<script setup lang="ts">
/**
 * A recording tab: a player at the top of the pane on a rule, and under it the
 * transcript written as one text.
 *
 * A cue is a place in the recording, so its time in the gutter is clicked to
 * play from there. What can be asked over the words stands in the menu at the
 * end of the strip, each item only where it applies.
 */
import { computed, ref, watchPostEffect } from 'vue'
import { Ellipsis, LocateFixed } from '@lucide/vue'
import { Editor, Menu, Player, timing } from '@numen/ui'
import type { Position } from '@numen/ui'
import { iconFor } from '../icons'
import { DROP, PROOFREAD, WORDS as words } from './words'
import type { RecordingTabState } from './kind'

const props = defineProps<{ state: RecordingTabState }>()

// The tab's state outlives this component, so what it holds is bound once here
// and the template unwraps it.
const {
  broken,
  current,
  droppable,
  editable,
  following,
  note,
  now,
  playable,
  playing,
  proofreadable,
  prose,
  runs,
  times: cues,
  transcribable,
  trouble,
  typing,
} = props.state

/** The times in the editor's gutter, and the line being said. */
const times = timing((line) => props.state.goes(line))

// The words move under the recording as it plays: the line being said is drawn
// in the accent, and following is what brings it back into view.
watchPostEffect(() =>
  times.show({
    times: cues.value,
    current: current.value,
    following: following.value && !typing.value,
  }),
)

/** Whether the view keeps the line being said in sight. */
const follows = computed(() => following.value)

/** Whether this recording has no transcript, which decides what stands below the player. */
const empty = computed(() => cues.value.length === 0)

/**
 * What the menu offers over this recording: each item only where it applies,
 * and the one that takes the words away last.
 */
const offered = computed(() => [
  ...(proofreadable.value ? [{ id: PROOFREAD, text: words.proofread }] : []),
  ...(droppable.value ? [{ id: DROP, text: words.drop }] : []),
])

/** Where the menu was asked for, and nothing while it is not open. */
const asking = ref<{ at: Position; from: HTMLElement } | null>(null)

const asks = (event: Event) => {
  const button = event.currentTarget
  if (!(button instanceof HTMLElement)) return
  const box = button.getBoundingClientRect()
  asking.value = { at: { x: box.left, y: box.bottom }, from: button }
}

const chose = (id: string) => {
  asking.value = null
  if (id === PROOFREAD) props.state.proofreads()
  if (id === DROP) props.state.drops()
}
</script>

<template>
  <div class="recording">
    <div class="recording__head">
      <Player
        v-if="playable"
        class="recording__player"
        :at="now"
        :length="runs"
        :playing="playing"
        :label="words.player"
        @play="props.state.play()"
        @pause="props.state.pause()"
        @seek="props.state.go($event)"
      />
      <p v-else class="recording__note">{{ words.unplayable }}</p>

      <!-- The controls over the words, standing together at the end of the
           strip. -->
      <div v-if="!empty" class="recording__deeds">
        <button
          type="button"
          class="recording__follow"
          :aria-label="words.follow"
          :title="words.follow"
          :aria-pressed="follows ? 'true' : 'false'"
          @click="props.state.follows(!follows)"
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
    </div>

    <div class="recording__below">
      <!-- What went wrong stands above the words, where it is read whether or
           not there are any. -->
      <p v-if="broken" class="recording__note">
        {{ broken }}
      </p>
      <p v-if="trouble" role="alert" class="recording__note">
        {{ trouble }}
      </p>

      <!-- The button says there is no transcript, so the note says it only
           where there is no button. -->
      <div v-if="empty" class="recording__silence">
        <p v-if="!transcribable" class="recording__note">
          {{ note }}
        </p>
        <button
          v-else
          type="button"
          class="recording__ask"
          @click="props.state.transcribes()"
        >
          {{ words.transcribe }}
        </button>
      </div>

      <Editor
        v-else
        class="recording__transcript"
        :model-value="prose"
        :readonly="!editable"
        :live="false"
        :extensions="times.extension"
        :aria-label="words.transcript"
        @update:model-value="(said: string) => props.state.typed(said)"
        @save="props.state.keep()"
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
  /* Between the player and the controls at the end of the strip, and between
     those controls, which are one group. */
  --recording-apart: 1rem;
  --recording-close: 0.25rem;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
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
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
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
  border-color: var(--numen-accent);
}

.recording__player {
  flex: 1;
  min-inline-size: 0;
}

/* The controls at the end of the strip are one group, and stand at the group's
   own spacing. */
.recording__deeds {
  display: flex;
  align-items: center;
  flex: none;
  gap: var(--recording-close);
}

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
  color: var(--numen-accent);
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

/* The words are read in one column, clear of the rule the player stands on, and
   the editor scrolls so the bar stands at the edge of the pane. */
.recording .recording__transcript {
  --editor-measure: var(--recording-measure);
  --editor-lead: var(--numen-gutter);
  --editor-margin: var(--numen-gutter);

  flex: 1;
  min-block-size: 0;
  isolation: isolate;
  font-size: inherit;
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
