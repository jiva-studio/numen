<script setup lang="ts">
/**
 * The settings a preset's goal schedules by, a row to each. A row is drawn in
 * the control its field takes, held to the ends the application answered with,
 * and written when the person is done with it.
 *
 * A row carries `data-preset-row`, the field it is about. `data-preset` names
 * the parts of one: `name`, `detail`, `day`, `choice` and `percent`.
 */
import { computed } from 'vue'
import { NumberField, SegmentedControl, Select, Slider, Switch, WeekdayChips, WEEK } from '@numen/ui'
import type { Day } from '@numen/ui'
import type { PresetTabState, SettingValue } from './kind'
import { BUDGET_UNITS, LOADS, RULES, WHOLE_LOAD, loadOn, loaded } from './core'
import type { Bounds, BudgetUnit } from './core'
import { fieldsUnder, round, type Field } from './curve'
import { WORDS as words } from './words'

const props = defineProps<{ state: PresetTabState }>()

const { bounds, settings } = props.state

/** The rows the chosen goal schedules by, which are the ones drawn. */
const fields = computed(() => fieldsUnder(settings.value.goal, settings.value.learned))

/** The two units a budget is spent in, as the row offers them. */
const budgetUnits = BUDGET_UNITS.map((one) => ({ id: one, text: words.budgetUnitName(one) }))

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

/** How far a control runs, and nothing at all where nothing was said. */
type ControlBounds = { min: number; max: number } | Record<string, never>

/** One pair of ends as a control takes them, counted in the field's own units. */
const ends = (one: Bounds | undefined, per = 1): ControlBounds =>
  one ? { min: one.least * per, max: one.most * per } : {}

/**
 * How far a field goes, as the control it is drawn in takes it. A field the
 * application has said no bound for is left to the control's own ends.
 */
const boundsOf = (field: Field): ControlBounds => {
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
  props.state.types(field, field === 'retention' ? round(said / 100, 2) : said)
}

/**
 * A row a person is done with, which is what writes the group. A control moved
 * a step at a time says so when it is let go of; one that turns in a single
 * gesture is done the moment it turns.
 */
const chose = (field: Field, value: SettingValue) => {
  props.state.types(field, value)
  props.state.settles()
}

/** A day typed into the row that holds one. */
const dated = (said: Event) => {
  chose('byDate', (said.target as HTMLInputElement).value)
}
</script>

<template>
  <section class="preset-settings" :aria-label="words.settings">
    <div
      v-for="field in fields"
      :key="field"
      class="preset-settings__row"
      :data-preset-row="field"
    >
      <span class="preset-settings__said">
        <span class="preset-settings__name" :id="`preset-${field}`" data-preset="name">{{
          words.fieldName(field)
        }}</span>
        <span class="preset-settings__detail" data-preset="detail">{{
          words.fieldDetail(field)
        }}</span>
      </span>

      <span class="preset-settings__value">
        <input
          v-if="field === 'byDate'"
          type="date"
          class="preset-settings__day"
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
          class="preset-settings__choice"
          data-preset="choice"
          @update:model-value="ruled"
        />
        <SegmentedControl
          v-else-if="field === 'counts'"
          :model-value="settings.counts"
          :choices="budgetUnits"
          :aria-labelledby="`preset-${field}`"
          @update:model-value="(one: string) => chose(field, one as BudgetUnit)"
        />
        <!-- A share is moved along its whole range and read out beside
             the track, which draws no figure of its own. -->
        <template v-else-if="field === 'backlog'">
          <Slider
            :model-value="settings.backlog"
            v-bind="boundsOf(field)"
            :step="1"
            :aria-labelledby="`preset-${field}`"
            class="preset-settings__slider"
            @update:model-value="(share: number) => props.state.types(field, share)"
            @settles="props.state.settles()"
          />
          <span class="preset-settings__percent" data-preset="percent">{{
            words.percent(settings.backlog)
          }}</span>
        </template>
        <WeekdayChips
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
          class="preset-settings__number"
          @update:model-value="(said: number | null) => typed(field, said)"
          @settles="props.state.settles()"
        />
      </span>
    </div>
  </section>
</template>

<style scoped>
/* Every row is one grid, so a control begins on the line the control above it
   begins on and every line of prose wraps at the one measure. The control
   column is as wide as the widest control the rows hold. */
.preset-settings {
  /* One row: the box a number is typed into, the list a rule is taken from,
     the air around the row, and the space between what it is called and what
     it means. */
  --preset-settings-value: 6rem;
  --preset-settings-choice: 10rem;
  --preset-settings-air: 0.5rem;
  --preset-settings-said-gap: 0.125rem;
  /* The track a share is moved along, and the room the figure beside it takes. */
  --preset-settings-track: 9rem;
  --preset-settings-percent: 2.25rem;
  display: grid;
  grid-template-columns: 1fr max-content;
}

/* The name and what it means on the left, the control on the right. The row
   takes the settings' own two columns, so nothing is measured row by row. */
.preset-settings__row {
  display: grid;
  grid-column: 1 / -1;
  grid-template-columns: subgrid;
  align-items: center;
  gap: 0 var(--numen-panel-gap);
  padding-block: var(--preset-settings-air);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

/* What the row is called, and under it what it means. */
.preset-settings__said {
  display: flex;
  flex-direction: column;
  gap: var(--preset-settings-said-gap);
  min-inline-size: 0;
}

/* The name of a row and what it means are one size, and the name carries the
   weight and the colour that tell them apart. */
.preset-settings__name {
  color: var(--numen-ink);
  font-weight: 500;
}

.preset-settings__detail {
  color: var(--numen-hushed);
}

/* Controls of every width end at the one edge. */
.preset-settings__value {
  display: flex;
  align-items: center;
  justify-content: end;
  gap: var(--numen-node-gap);
}

.preset-settings__number {
  inline-size: var(--preset-settings-value);
}

.preset-settings__slider {
  inline-size: var(--preset-settings-track);
}

/* What the track stands at, which the track itself does not draw. */
.preset-settings__percent {
  min-inline-size: var(--preset-settings-percent);
  color: var(--numen-hushed);
  font-variant-numeric: tabular-nums;
  text-align: end;
}

/* The rule is taken from a list, and the list is as wide as the rules are. */
.preset-settings__choice {
  inline-size: var(--preset-settings-choice);
}

/* The same box the numbers of the receipt are typed into, one row tall with
   the day centred in it. */
.preset-settings__day {
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
.preset-settings__day::-webkit-inner-spin-button,
.preset-settings__day::-webkit-clear-button {
  display: none;
  -webkit-appearance: none;
  appearance: none;
}

.preset-settings__day::-webkit-datetime-edit {
  padding: 0;
  line-height: 1;
}

.preset-settings__day:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: var(--numen-stroke);
}
</style>
