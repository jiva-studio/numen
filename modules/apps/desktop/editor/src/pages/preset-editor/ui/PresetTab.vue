<script setup lang="ts">
/**
 * Displays flashcard review preset settings and target curve slider.
 */
import { computed } from 'vue'
import { SegmentedControl } from '@numen/ui'
import CurveSlider from './curve-slider/CurveSlider.vue'
import PresetSettings from './preset-settings/PresetSettings.vue'
import type { PresetTabState } from '../types'
import { GOALS } from '../types'
import type { Goal } from '../types'
import { idle, valueAt } from '../lib/curve'
import { WORDS as words } from '../words'

// --- Props & Emits ---
const props = defineProps<{ state: PresetTabState }>()

// --- State ---
// The tab's state outlives this component, so what it holds is bound once here
// and the template unwraps it.
const {
  hasChanged,
  curve,
  material,
  place,
  problems,
  errorMessage,
  settings,
  stopped: stoppedAt,
  waiting,
} = props.state

/** Why the goal has nothing to work on, and empty where it has. */
const nothing = computed(() => idle(curve.value))

/** What is said in the control's place where the goal has nothing to work on. */
const emptyGoalExplanation = computed(() => {
  if (nothing.value === 'unpointed') return words.unpointed
  if (nothing.value === 'noCards') return words.noCards(curve.value.decks)
  return nothing.value === 'beginsNothing' ? words.beginsNothing : ''
})

/** The three goals, as the switch above the curve offers them. */
const goals = computed(() => GOALS.map((one) => ({ id: one, text: words.goalName(one) })))

/** The day the knob stands at, under the goal that steers one. */
const day = computed(() => curve.value.days[place.value] ?? settings.value.byDate)

/** The value the control stands at, in the units of its goal. */
const reading = computed(() =>
  words.value(curve.value.goal, valueAt(curve.value, place.value), day.value),
)

/** Why the preset schedules nothing, and empty while it schedules something. */
const stopped = computed(() => words.stopped(stoppedAt.value))

// --- Handlers ---
function onRefresh() {
  props.state.reload()
}

function onSelectGoal(one: string) {
  props.state.chooseGoal(one as Goal)
}

function onMoveSlider(at: number) {
  props.state.moveSlider(at)
}

function onSettleSlider() {
  props.state.settle()
}

// --- Helpers ---
</script>

<template>
  <div class="preset">
    <!-- Reading the file again is the way out of anything the tab has to say,
         and it waits on nothing in the vault. -->
    <p v-if="errorMessage" role="alert" class="preset__warning preset__answering">
      {{ errorMessage }}
      <button type="button" class="answer" @click="onRefresh">
        {{ words.reads }}
      </button>
    </p>

    <p v-if="hasChanged" role="status" class="preset__warning preset__answering">
      {{ words.changed }}
      <button type="button" class="answer" @click="onRefresh">
        {{ words.reads }}
      </button>
    </p>

    <ul
      v-if="problems.length"
      class="preset__warning preset__problems"
      :aria-label="words.problems"
    >
      <li v-for="(text, at) in problems" :key="at">{{ text }}</li>
    </ul>

    <div class="preset__page">
      <div class="preset__column">
        <section class="preset__goal" :aria-label="words.goal">
          <p class="preset__label" data-preset="label">{{ words.goal }}</p>

          <SegmentedControl
            :model-value="settings.goal"
            :choices="goals"
            @update:model-value="onSelectGoal"
          />

          <p v-if="nothing" class="preset__unpointed" data-preset="unpointed">{{ emptyGoalExplanation }}</p>

          <template v-else>
            <CurveSlider
              :curve="curve"
              :material="material"
              :place="place"
              :value-text="reading"
              :waiting="waiting"
              @move="onMoveSlider"
              @settle="onSettleSlider"
            />
          </template>

          <p v-if="stopped" class="preset__stopped" data-preset="stopped">{{ stopped }}</p>
        </section>

        <PresetSettings :state="props.state" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.preset {
  /* The measure a preset is read at, which the goal and the settings share. */
  --preset-measure: 46rem;
  /* The step every gap down the column is set by. */
  --preset-step: 1.25rem;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
}

.preset__page {
  flex: 1;
  min-block-size: 0;
  overflow-y: auto;
  padding: var(--numen-gutter);
}

/*
 * The column the tab is read in, centred in whatever room the pane has. Its
 * two edges are the lines this tab is read against: a block that draws a box
 * puts the box's edge on them, and a block that draws none puts its text
 * there. Nothing in the column is inset from them, and nothing is pulled out.
 */
.preset__column {
  display: flex;
  flex-direction: column;
  gap: var(--preset-step);
  inline-size: 100%;
  max-inline-size: var(--preset-measure);
  margin-inline: auto;
}

/* The goal, its picture and what it reads are parts of one block, so they
   stand close together and the block itself a whole step from the next. */
.preset__goal {
  display: flex;
  flex-direction: column;
  gap: var(--numen-node-gap);
}

/* What the three segments are, said over them. */
.preset__label {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-weight: 600;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

/* No deck points here, said where the curve would stand. */
.preset__unpointed {
  margin: 0;
  color: var(--numen-hushed);
  line-height: var(--numen-line-height);
}

.preset__stopped {
  margin: 0;
  padding: var(--numen-inset) var(--numen-inset-wide);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
}

.preset__warning {
  margin: 0;
  padding: var(--numen-inset) var(--numen-gutter);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  overflow-wrap: break-word;
}

.preset__answering {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0 var(--numen-panel-gap);
}

.preset__problems {
  padding-inline-start: calc(var(--numen-gutter) * 2);
  list-style: disc;
}
</style>
