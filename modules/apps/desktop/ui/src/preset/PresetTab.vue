<script setup lang="ts">
/**
 * A preset tab: the one control at the top, and under it the settings its goal
 * schedules by. The goal owns one value and writes only that; every other
 * setting stands as the person left it.
 *
 * A row of the receipt carries `data-preset-row`, the field it is about.
 * `data-preset` names the rest: `label`, `unpointed`, `stopped`, `name`,
 * `detail`, `day`, `choice` and `percent`.
 */
import { computed } from 'vue'
import { Days, NumberField, Segmented, Select, Slider, Switch, WEEK } from '@numen/ui'
import type { Day } from '@numen/ui'
import CurveSlider from './CurveSlider.vue'
import type { PresetTabState, SettingValue } from './kind'
import { COUNTS, GOALS, LOADS, RULES, WHOLE_LOAD, loadOn, loaded } from './core'
import type { Bounds, Counts, Goal } from './core'
import { fieldsUnder, idle, round, type Field } from './curve'
import { WORDS as words } from './words'

const props = defineProps<{ held: PresetTabState }>()

// The tab's state outlives this component, so what it holds is bound once here
// and the template unwraps it.
const {
  bounds,
  changed,
  curve,
  material,
  place,
  problems,
  saying,
  settings,
  stopped: stoppedAt,
  waiting,
} = props.held

/** Why the goal has nothing to work on, and empty where it has. */
const nothing = computed(() => idle(curve.value))

/** What is said in the control's place where the goal has nothing to work on. */
const saidInstead = computed(() => {
  if (nothing.value === 'unpointed') return words.unpointed
  if (nothing.value === 'noCards') return words.noCards(curve.value.decks)
  return nothing.value === 'beginsNothing' ? words.beginsNothing : ''
})

/** The rows the chosen goal schedules by, which are the ones drawn. */
const fields = computed(() => fieldsUnder(settings.value.goal, settings.value.learned))

/** The three goals, as the switch above the curve offers them. */
const goals = computed(() => GOALS.map((one) => ({ id: one, text: words.goalName(one) })))

/** The two things a budget is spent on, as the row offers them. */
const counts = COUNTS.map((one) => ({ id: one, text: words.countsName(one) }))

/** The two rules for what is learned, as the row offers them. */
const rules = RULES.map((one) => ({ id: one, text: words.ruleName(one) }))

/** The week, each day drawn at the share of a day's load it carries. */
const week = computed<readonly Day[]>(() =>
  WEEK.map((one) => ({ ...one, level: loadOn(settings.value.load, one.id) / WHOLE_LOAD })),
)

/** The shares a day may be put at, as the row draws them. */
const levels = LOADS.map((one) => one / WHOLE_LOAD)

/** One day of the week put at a share of a day's load. */
const loads = (day: string, level: number) => {
  chose('load', loaded(settings.value.load, day, Math.round(level * WHOLE_LOAD)))
}

const ruled = (said: string) => {
  if (said === 'interval' || said === 'retention') chose('learned', said)
}

/** The day the knob stands at, under the goal that steers one. */
const day = computed(() => curve.value.days[place.value] ?? settings.value.byDate)

/**
 * The value the knob stands at. A preset's own value need not sit on the grid,
 * and the place it opens at is the one nearest it, so while the knob has not
 * been moved off that place the preset's own value is what is read out.
 */
const value = computed(() =>
  curve.value.now.at >= 0 && place.value === curve.value.now.at
    ? curve.value.now.value
    : (curve.value.grid[place.value] ?? 0),
)

/** The value the control stands at, in the units of its goal. */
const reading = computed(() => words.value(curve.value.goal, value.value, day.value))

/** How far a control runs, and nothing at all where nothing was said. */
type Ends = { min: number; max: number } | Record<string, never>

/** One pair of ends as a control takes them, counted in the field's own units. */
const ends = (one: Bounds | undefined, per = 1): Ends =>
  one ? { min: one.least * per, max: one.most * per } : {}

/**
 * How far a field goes, as the control it is drawn in takes it. A field the
 * application has said no bound for is left to the control's own ends.
 */
const boundsOf = (field: Field): Ends => {
  const within = bounds.value
  if (field === 'newADay') return ends(within.newADay)
  if (field === 'reviewsADay') return ends(within.reviewsADay)
  if (field === 'minutesADay') return ends(within.minutesADay)
  if (field === 'backlog') return ends(within.backlog)
  if (field === 'interval') return ends(within.interval)
  // A chance of recall is read and typed in per cent, which is how every figure
  // beside it is said.
  if (field === 'retention') return ends(within.retention, 100)
  return {}
}

/** Whether a row draws a number, and the number it draws. */
const counted = (field: Field): number | null => {
  if (field === 'newADay') return settings.value.newADay
  if (field === 'reviewsADay') return settings.value.reviewsADay
  if (field === 'retention') return Math.round(settings.value.retention * 100)
  if (field === 'minutesADay') return settings.value.minutesADay
  if (field === 'interval') return settings.value.interval
  return null
}

/** A number typed into a row. An empty field leaves the setting as it stands. */
const typed = (field: Field, said: number | null) => {
  if (said === null) return
  props.held.types(field, field === 'retention' ? round(said / 100, 2) : said)
}

/**
 * A row a person is done with, which is what writes the group. A control moved
 * a step at a time says so when it is let go of; one that turns in a single
 * gesture is done the moment it turns.
 */
const chose = (field: Field, value: SettingValue) => {
  props.held.types(field, value)
  props.held.settles()
}

/** A day typed into the row that holds one. */
const dated = (said: Event) => {
  chose('byDate', (said.target as HTMLInputElement).value)
}

/** Why the preset schedules nothing, and empty while it schedules something. */
const stopped = computed(() => words.stopped(stoppedAt.value))
</script>

<template>
  <div class="preset">
    <!-- Reading the file again is the way out of anything the tab has to say,
         and it waits on nothing in the vault. -->
    <p v-if="saying" role="alert" class="preset__warning preset__answering">
      {{ saying }}
      <button type="button" class="answer" @click="props.held.again()">
        {{ words.reads }}
      </button>
    </p>

    <p v-if="changed" role="status" class="preset__warning preset__answering">
      {{ words.changed }}
      <button type="button" class="answer" @click="props.held.again()">
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

          <Segmented
            :model-value="settings.goal"
            :choices="goals"
            @update:model-value="(one: string) => props.held.chooses(one as Goal)"
          />

          <p v-if="nothing" class="preset__unpointed" data-preset="unpointed">{{ saidInstead }}</p>

          <template v-else>
            <CurveSlider
              :curve="curve"
              :material="material"
              :place="place"
              :value-text="reading"
              :waiting="waiting"
              @moves="(at: number) => props.held.moves(at)"
              @settles="props.held.settles()"
            />
          </template>

          <p v-if="stopped" class="preset__stopped" data-preset="stopped">{{ stopped }}</p>
        </section>

        <section class="preset__settings" :aria-label="words.settings">
          <div v-for="field in fields" :key="field" class="preset__row" :data-preset-row="field">
            <span class="preset__said">
              <span class="preset__name" :id="`preset-${field}`" data-preset="name">{{
                words.fieldName(field)
              }}</span>
              <span class="preset__detail" data-preset="detail">{{
                words.fieldDetail(field)
              }}</span>
            </span>

            <span class="preset__value">
              <input
                v-if="field === 'byDate'"
                type="date"
                class="preset__day"
                data-preset="day"
                :value="settings.byDate"
                :aria-labelledby="`preset-${field}`"
                @change="dated"
              />
              <Select
                v-else-if="field === 'learned'"
                :model-value="settings.learned"
                :choices="rules"
                :name="words.fieldName('learned')"
                :aria-labelledby="`preset-${field}`"
                class="preset__choice"
                data-preset="choice"
                @update:model-value="ruled"
              />
              <Segmented
                v-else-if="field === 'counts'"
                :model-value="settings.counts"
                :choices="counts"
                :aria-labelledby="`preset-${field}`"
                @update:model-value="(one: string) => chose(field, one as Counts)"
              />
              <!-- A share is moved along its whole range and read out beside
                   the track, which draws no figure of its own. -->
              <template v-else-if="field === 'backlog'">
                <Slider
                  :model-value="settings.backlog"
                  v-bind="boundsOf(field)"
                  :step="1"
                  :aria-labelledby="`preset-${field}`"
                  class="preset__slider"
                  @update:model-value="(share: number) => props.held.types(field, share)"
                  @settles="props.held.settles()"
                />
                <span class="preset__percent" data-preset="percent">{{
                  words.percent(settings.backlog)
                }}</span>
              </template>
              <Days
                v-else-if="field === 'load'"
                :days="week"
                :levels="levels"
                :aria-labelledby="`preset-${field}`"
                @chooses="loads"
              />
              <Switch
                v-else-if="field === 'evenLoad'"
                :model-value="settings.evenLoad"
                :aria-labelledby="`preset-${field}`"
                @update:model-value="(on: boolean) => chose(field, on)"
              />
              <NumberField
                v-else
                :model-value="counted(field)"
                v-bind="boundsOf(field)"
                :step="1"
                :aria-labelledby="`preset-${field}`"
                class="preset__number"
                @update:model-value="(said: number | null) => typed(field, said)"
                @settles="props.held.settles()"
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
  /* The step every gap down the column is set by. */
  --preset-step: 1.25rem;
  /* One row of the receipt: the box a number is typed into, the list a rule is
     taken from, the air around the row, and the space between what it is
     called and what it means. */
  --preset-value: 6rem;
  --preset-choice: 10rem;
  --preset-row-air: 0.5rem;
  --preset-said-gap: 0.125rem;
  /* The track a share is moved along, and the room the figure beside it takes. */
  --preset-track: 9rem;
  --preset-percent: 2.25rem;
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

/* Every row is one grid, so a control begins on the line the control above it
   begins on and every line of prose wraps at the one measure. The control
   column is as wide as the widest control the rows hold. */
.preset__settings {
  display: grid;
  grid-template-columns: 1fr max-content;
}

/* The name and what it means on the left, the control on the right. The row
   takes the settings' own two columns, so nothing is measured row by row. */
.preset__row {
  display: grid;
  grid-column: 1 / -1;
  grid-template-columns: subgrid;
  align-items: center;
  gap: 0 var(--numen-panel-gap);
  padding-block: var(--preset-row-air);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

/* What the row is called, and under it what it means. */
.preset__said {
  display: flex;
  flex-direction: column;
  gap: var(--preset-said-gap);
  min-inline-size: 0;
}

/* The name of a row and what it means are one size, and the name carries the
   weight and the colour that tell them apart. */
.preset__name {
  color: var(--numen-ink);
  font-weight: 500;
}

/* Controls of every width end at the one edge. */
.preset__value {
  display: flex;
  align-items: center;
  justify-content: end;
  gap: var(--numen-node-gap);
}

.preset__number {
  inline-size: var(--preset-value);
}

.preset__slider {
  inline-size: var(--preset-track);
}

/* What the track stands at, which the track itself does not draw. */
.preset__percent {
  min-inline-size: var(--preset-percent);
  color: var(--numen-hushed);
  font-variant-numeric: tabular-nums;
  text-align: end;
}

/* The rule is taken from a list, and the list is as wide as the rules are. */
.preset__choice {
  inline-size: var(--preset-choice);
}

/* The same box the numbers of the receipt are typed into, one row tall with
   the day centred in it. */
.preset__day {
  block-size: var(--numen-action-size);
  padding-inline: var(--numen-inset);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-field-bg);
  color: inherit;
  font: inherit;
  line-height: 1;
}

/* The calendar the machine draws inside the field comes with a spinner and a
   cross of its own. */
.preset__day::-webkit-inner-spin-button,
.preset__day::-webkit-clear-button {
  display: none;
  -webkit-appearance: none;
  appearance: none;
}

.preset__day::-webkit-datetime-edit {
  padding: 0;
  line-height: 1;
}

.preset__day:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: var(--numen-stroke);
}

.preset__detail {
  color: var(--numen-hushed);
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
