<script setup lang="ts">
/**
 * What the goals of this vault come to today: one line for each preset, saying
 * what its goal is and how far through it the day stands.
 *
 * A preset keeps a budget in cards and a budget in time. The day stands as far
 * through as the fuller of the two, which is the one a person is nearest the
 * end of.
 *
 * The screen is read at a glance, so a row is one line of words.
 */
import { computed } from 'vue'

import { goalWords, holds } from './scheduling'
import type { Preset } from './scheduling'

const props = defineProps<{
  presets: readonly Preset[]
  /** The day this is being read on, as the year, the month and the day. */
  today: string
}>()

/** One preset's line: its goal, and how far through the day it stands. */
interface Line {
  readonly one: Preset
  readonly goal: string
  /** How much of the day is done, and above one for a day drawn past it. */
  readonly done: number
  /** How far through the day it stands, in the words the row says it in. */
  readonly says: string
  /** Whether no deck points at it, which is a preset that schedules nobody. */
  readonly alone: boolean
  /** Why it has no cards to schedule, and empty where it has some. */
  readonly bare: string
}

/** How many decks point here and hold nothing between them. */
const bareWords = (decks: number): string =>
  decks === 1 ? 'One deck, no cards' : `${decks} decks, no cards`

const lines = computed<Line[]>(() =>
  props.presets.map((one) => {
    const done = through(one)
    const alone = one.named === 0
    return {
      one,
      goal: one.settings ? goalWords(one.settings, props.today) : '',
      done,
      says: done > 1 ? 'over budget' : percent(done),
      alone,
      bare: !alone && one.faces === 0 ? bareWords(one.named) : '',
    }
  }),
)

/**
 * How far through its day a preset stands: what has been answered against the
 * cards the day holds, and what it has taken against the minutes the day runs,
 * whichever of the two is further along.
 *
 * A day answered past what its budget holds stands above one, which is a day
 * that is over its budget and not a day that is done.
 */
const through = (one: Preset): number => {
  const ofCards = holds(one.budget)
  const ofMinutes = one.budget.minutes
  return Math.max(
    0,
    ofCards > 0 ? one.answered / ofCards : 0,
    ofMinutes > 0 ? one.took / ofMinutes : 0,
  )
}

/** How far through the day a line stands, as a person reads it. */
const percent = (done: number): string => `${Math.round(done * 100)}%`
</script>

<template>
  <section v-if="lines.length" class="presets">
    <ul class="presets__list">
      <li
        v-for="line in lines"
        :key="line.one.path"
        class="presets__preset"
        :class="{ 'presets__preset--paused': line.one.paused || line.alone || line.bare }"
      >
        <p class="presets__name">{{ line.one.name }}</p>
        <!-- A preset with no cards under it schedules nobody, so the day it
             would hold is not drawn against it. A deck holding nothing still
             points here, and that is a state of its own. -->
        <p v-if="line.alone" class="presets__alone">Nothing points here</p>
        <p v-else-if="line.bare" class="presets__alone">{{ line.bare }}</p>
        <p v-else class="presets__goal">{{ line.one.paused || line.goal }}</p>

        <!-- A day answered past its budget says so, rather than standing at a
             round number it is already past. -->
        <p
          v-if="!line.alone && !line.bare && !line.one.paused"
          class="presets__done"
          :data-over="line.done > 1 ? '' : undefined"
        >
          {{ line.says }}
        </p>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.presets {
  display: flex;
  flex: none;
  flex-direction: column;
  gap: var(--numen-inset);
}

.presets__list {
  display: flex;
  margin: 0;
  padding: 0;
  flex-direction: column;
  gap: var(--numen-inset);
  list-style: none;
}

/* A row is one line: the name, the goal, and how far through the day it is at
   the end of it. */
.presets__preset {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: baseline;
  column-gap: var(--numen-inset);
}

.presets__name {
  margin: 0;
  font-weight: 600;
}

.presets__goal,
.presets__alone {
  margin: 0;
  color: var(--numen-hushed);
}

.presets__done {
  margin: 0;
  justify-self: end;
  color: var(--numen-hushed);
  font-size: var(--numen-edge-label-size);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

/* A day over its budget is the one thing on the row worth catching the eye. */
.presets__done[data-over] {
  color: var(--numen-caution-fg);
}

/* A preset scheduling nothing stands with its reason and nothing else. */
.presets__preset--paused .presets__name,
.presets__preset--paused .presets__goal,
.presets__preset--paused .presets__alone {
  color: var(--numen-hushed);
}
</style>
