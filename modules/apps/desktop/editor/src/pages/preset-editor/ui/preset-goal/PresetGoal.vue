<script setup lang="ts">
/**
 * The goal a preset is scheduled by, the switch that chooses it, and the curve
 * that goal draws. Where the goal has nothing to work on, a line stands in the
 * curve's place and says why.
 */
import { computed } from 'vue'
import { SegmentedControl } from '@numen/ui'
import CurveSlider from '../curve-slider/CurveSlider.vue'
import type { PresetTabState } from '../../types'
import { GOALS } from '../../types'
import type { Goal } from '../../types'
import { idle, valueAt } from '../../lib/curve'
import { WORDS as words } from '../../words'

// --- Props & Emits ---
const props = defineProps<{ state: PresetTabState }>()

// --- State ---
// The tab's state outlives this component, so what it holds is bound once here
// and the template unwraps it.
const { curve, material, place, settings, stopped: stoppedAt, waiting } = props.state

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
  <section class="preset__goal" :aria-label="words.goal">
    <p class="preset__label" data-preset="label">{{ words.goal }}</p>

    <SegmentedControl
      :model-value="settings.goal"
      :choices="goals"
      @update:model-value="onSelectGoal"
    />

    <p v-if="nothing" class="preset__unpointed" data-preset="unpointed">
      {{ emptyGoalExplanation }}
    </p>

    <CurveSlider
      v-else
      :curve="curve"
      :material="material"
      :place="place"
      :value-text="reading"
      :waiting="waiting"
      @move="onMoveSlider"
      @settle="onSettleSlider"
    />

    <p v-if="stopped" class="preset__stopped" data-preset="stopped">{{ stopped }}</p>
  </section>
</template>

<style scoped>
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
</style>
