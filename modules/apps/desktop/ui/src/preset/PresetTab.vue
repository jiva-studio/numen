<script setup lang="ts">
/**
 * A preset tab: the one control at the top, and under it the settings its goal
 * schedules by.
 *
 * The control is the goal's curve. The goal owns one value and writes only
 * that; every other setting is the person's own and stands as they left it,
 * whatever the picture says it comes to.
 */
import { computed } from 'vue'
import { Days, NumberField, Segmented, Switch } from '@numen/ui'
import Control from './Control.vue'
import type { Held } from './kind'
import { BOUNDS, COUNTS, GOALS } from './core'
import type { Counts, Goal } from './core'
import { closed, fieldsUnder, idle, paused, spent, type Field } from './curve'
import { WORDS as words } from './words'

const props = defineProps<{ held: Held }>()

const settings = computed(() => props.held.settings())
const curve = computed(() => props.held.curve())
const place = computed(() => props.held.place())

/** Why the goal has nothing to work on, and empty where it has. */
const nothing = computed(() => idle(curve.value))

/** What is said in the control's place where the goal has nothing to work on. */
const saidInstead = computed(() => {
  if (nothing.value === 'unpointed') return words.unpointed
  return nothing.value === 'noCards' ? words.noCards(curve.value.decks) : ''
})

/** The rows the chosen goal schedules by, which are the ones drawn. */
const fields = computed(() => fieldsUnder(settings.value.goal))

/** The three goals, as the switch above the curve offers them. */
const goals = computed(() => GOALS.map((one) => ({ id: one, text: words.goalName(one) })))

/** The two things a budget is spent on, as the row offers them. */
const counts = COUNTS.map((one) => ({ id: one, text: words.countsName(one) }))

/** What the preset comes to where the knob stands. */
const point = computed(() => curve.value.at[place.value] ?? null)
const value = computed(() => curve.value.grid[place.value] ?? 0)
const day = computed(() => curve.value.days[place.value] ?? settings.value.byDate)

/** The value the control stands at, in the units of its goal. */
const reading = computed(() => words.value(curve.value.goal, value.value, day.value))

/** What standing there costs, in one sentence. */
const costing = computed(() =>
  point.value
    ? words.costs(
        curve.value.goal,
        value.value,
        point.value.reviews,
        point.value.minutes,
        point.value.retained,
      )
    : '',
)

/** How far a field goes. A field that holds no number is bounded by nothing. */
const boundsOf = (field: Field): { least: number; most: number } => {
  if (field === 'newADay') return BOUNDS.newADay
  if (field === 'reviewsADay') return BOUNDS.reviewsADay
  if (field === 'retention') return BOUNDS.retention
  if (field === 'minutesADay') return BOUNDS.minutesADay
  return { least: 0, most: 0 }
}

/** Whether a row draws a number, and the number it draws. */
const counted = (field: Field): number | null => {
  if (field === 'newADay') return settings.value.newADay
  if (field === 'reviewsADay') return settings.value.reviewsADay
  if (field === 'retention') return settings.value.retention
  if (field === 'minutesADay') return settings.value.minutesADay
  return null
}

const stepOf = (field: Field): number => (field === 'retention' ? 0.01 : 1)

/** A number typed into a row. An empty field leaves the setting as it stands. */
const typed = (field: Field, said: number | null) => {
  if (said === null) return
  props.held.types(field, said)
}

/** A day typed into the row that holds one. */
const dated = (said: Event) => {
  props.held.types('byDate', (said.target as HTMLInputElement).value)
}

/** Whether the preset schedules nothing, and why. */
const stopped = computed(() => {
  const today = new Date()
  if (spent(settings.value, today)) return words.spent
  return paused(settings.value, today) ? words.paused : ''
})

/** The arithmetic under a goal of a date, with the sum already done. */
const sums = computed<readonly string[]>(() => {
  const one = point.value
  if (curve.value.goal !== 'date' || !one) return []
  const lines = [
    // Nothing owed on that day is nothing to say about it.
    ...(one.owed > 0 ? [words.owing(one.owed)] : []),
    words.needing(one.minutes, settings.value.minutesADay),
    words.through(one.through, settings.value.minutesADay),
  ]
  return one.met ? lines : [...lines, words.unmet]
})

/** Whether a longer day buys nothing, which the card limits close it against. */
const shut = computed(() => closed(curve.value))

/**
 * How far behind the preset stands and what the place the knob is at does
 * about it. A preset with nothing overdue has nothing to clear, and says so by
 * saying nothing.
 */
const behind = computed<readonly string[]>(() => {
  const one = point.value
  if (!curve.value.honest || curve.value.overdue <= 0 || !one) return []
  return [words.behind(curve.value.overdue, curve.value.cards), words.clearing(one.clears)]
})
</script>

<template>
  <div class="preset">
    <p v-if="props.held.saying()" role="alert" class="preset__warning">
      {{ props.held.saying() }}
    </p>

    <p v-if="props.held.changed()" role="status" class="preset__warning preset__answering">
      {{ words.changed }}
      <button type="button" class="preset__answer" @click="props.held.again()">
        {{ words.reads }}
      </button>
    </p>

    <ul
      v-if="props.held.problems().length"
      class="preset__warning preset__problems"
      :aria-label="words.problems"
    >
      <li v-for="(text, at) in props.held.problems()" :key="at">{{ text }}</li>
    </ul>

    <div class="preset__page">
      <div class="preset__column">
        <section class="preset__goal" :aria-label="words.goal">
          <p class="preset__label">{{ words.goal }}</p>

          <Segmented
            :model-value="settings.goal"
            :choices="goals"
            @update:model-value="(one: string) => props.held.chooses(one as Goal)"
          />

          <p v-if="nothing" class="preset__unpointed">{{ saidInstead }}</p>

          <template v-else>
            <Control
              :curve="curve"
              :place="place"
              :value-text="reading"
              @moves="(at: number) => props.held.moves(at)"
              @settles="props.held.settles()"
            />

            <p class="preset__reading">
              <span class="preset__figure">{{ reading }}</span>
              <span class="preset__costing">
                <span v-if="!curve.honest" class="preset__about" :title="words.aboutMeaning">
                  {{ words.about }}
                </span>
                {{ costing }}
              </span>
            </p>

            <!-- How far behind, what this pace does about it, and what a
                 longer day would not buy. -->
            <p v-for="(text, at) in behind" :key="at" class="preset__shut">{{ text }}</p>

            <p v-if="shut" class="preset__shut">
              {{ words.closed(settings.newADay, settings.reviewsADay) }}
            </p>

            <ul v-if="sums.length" class="preset__sums">
              <li v-for="(text, at) in sums" :key="at">{{ text }}</li>
            </ul>
          </template>

          <p v-if="stopped" class="preset__stopped">{{ stopped }}</p>
        </section>

        <section class="preset__settings" :aria-label="words.settings">
          <div v-for="field in fields" :key="field" class="preset__row">
            <span class="preset__said">
              <span class="preset__name" :id="`preset-${field}`">{{ words.fieldName(field) }}</span>
              <span class="preset__detail">{{ words.fieldDetail(field) }}</span>
            </span>

            <span class="preset__value">
              <input
                v-if="field === 'byDate'"
                type="date"
                class="preset__day"
                :value="settings.byDate"
                :aria-labelledby="`preset-${field}`"
                @change="dated"
              />
              <Segmented
                v-else-if="field === 'counts'"
                :model-value="settings.counts"
                :choices="counts"
                :aria-labelledby="`preset-${field}`"
                @update:model-value="(one: string) => props.held.types(field, one as Counts)"
              />
              <Days
                v-else-if="field === 'lightDays'"
                :model-value="settings.lightDays"
                :aria-labelledby="`preset-${field}`"
                @update:model-value="(days: readonly string[]) => props.held.types(field, days)"
              />
              <Switch
                v-else-if="field === 'evenLoad'"
                :model-value="settings.evenLoad"
                :aria-labelledby="`preset-${field}`"
                @update:model-value="(on: boolean) => props.held.types(field, on)"
              />
              <NumberField
                v-else
                :model-value="counted(field)"
                :min="boundsOf(field).least"
                :max="boundsOf(field).most"
                :step="stepOf(field)"
                :aria-labelledby="`preset-${field}`"
                class="preset__number"
                @update:model-value="(said: number | null) => typed(field, said)"
              />
            </span>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<style scoped>
.preset {
  /* The measure a preset is read at, which the goal and the settings share. */
  --preset-measure: 46rem;
  /* Between the goal and the settings under it, and inside each. */
  --preset-apart: 1.75rem;
  --preset-near: 0.625rem;
  /* One row of the receipt: the box a number is typed into, the air around the
     row, and the space between what it is called and what it means. */
  --preset-value: 6rem;
  --preset-row-air: 0.5rem;
  --preset-said-gap: 0.125rem;
  /* The clearance typing keeps from the ends of its box. */
  --preset-field-inset: 0.75rem;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-node-fg);
}

.preset__page {
  flex: 1;
  min-block-size: 0;
  overflow-y: auto;
  padding: var(--numen-gutter);
}

/* The column the tab is read in, centred in whatever room the pane has. */
.preset__column {
  display: flex;
  flex-direction: column;
  gap: var(--preset-apart);
  inline-size: 100%;
  max-inline-size: var(--preset-measure);
  margin-inline: auto;
}

.preset__goal {
  display: flex;
  flex-direction: column;
  gap: var(--preset-near);
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

.preset__reading {
  display: flex;
  flex-direction: column;
  gap: var(--numen-dot-gap);
  margin: 0;
}

/* The one line the tab is built around, in figures of one width. */
.preset__figure {
  font-size: var(--numen-text-4);
  font-variant-numeric: tabular-nums;
  line-height: 1.15;
}

.preset__costing {
  color: var(--numen-hushed);
}

/* No deck points here, said where the curve would stand. */
.preset__unpointed {
  margin: 0;
  color: var(--numen-hushed);
  line-height: var(--numen-line-height);
}

/* The window's own arithmetic, standing until the application answers. */
.preset__about {
  margin-inline-end: var(--numen-node-gap);
  padding-inline: var(--numen-node-gap);
  border: var(--numen-stroke) solid var(--numen-node-border);
  border-radius: var(--numen-radius-pill);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  text-transform: lowercase;
}

/* The arithmetic of a goal of a date, aligned with the column it stands in. */
.preset__sums {
  margin: var(--numen-dot-gap) 0 0;
  padding: 0;
  color: var(--numen-hushed);
  line-height: var(--numen-line-height);
  list-style: none;
}

/* A day the card limits close before its minutes run out. */
.preset__shut {
  margin: 0;
  color: var(--numen-hushed);
  line-height: var(--numen-line-height);
}

.preset__stopped {
  margin: 0;
  padding: var(--numen-inset) var(--numen-inset-wide);
  border-radius: var(--numen-radius);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
}

.preset__settings {
  display: flex;
  flex-direction: column;
}

/* The name and what it means on the left, the control at the end of the row. */
.preset__row {
  display: grid;
  grid-template-columns: 1fr max-content;
  align-items: center;
  gap: 0 var(--numen-panel-gap);
  padding-block: var(--preset-row-air);
  border-block-end: var(--numen-stroke) solid var(--numen-node-border);
}

/* What the row is called, and under it what it means. */
.preset__said {
  display: flex;
  flex-direction: column;
  gap: var(--preset-said-gap);
  min-inline-size: 0;
}

.preset__name {
  color: var(--numen-node-fg);
}

/* Controls of every width end at the one edge. */
.preset__value {
  display: flex;
  align-items: center;
  justify-content: end;
}

.preset__number {
  inline-size: var(--preset-value);
}

/* The same box the numbers of the receipt are typed into. */
.preset__day {
  block-size: var(--numen-action-size);
  padding-inline: var(--preset-field-inset);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  background: var(--numen-field-bg);
  color: inherit;
  font: inherit;
}

.preset__day:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: var(--numen-stroke);
}

.preset__detail {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
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

.preset__answer {
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-decoration: underline;
  text-underline-offset: 0.15em;
  cursor: pointer;
}

.preset__answer:focus-visible {
  outline: var(--numen-stroke) solid currentColor;
  outline-offset: var(--numen-caret);
}

.preset__problems {
  padding-inline-start: calc(var(--numen-gutter) * 2);
  list-style: disc;
}
</style>
