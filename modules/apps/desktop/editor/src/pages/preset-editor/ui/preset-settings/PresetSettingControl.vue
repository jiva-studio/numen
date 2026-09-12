<script setup lang="ts">
/**
 * The specific control for one preset setting row.
 */
import { computed } from 'vue'
import { NumberField, SegmentedControl, Select, Slider, Switch, WeekdayChips, WEEK } from '@numen/ui'
import type { Day } from '@numen/ui'
import type { PresetTabState, SettingValue } from '../../types'
import { BUDGET_UNITS, LOADS, RULES, WHOLE_LOAD, loadOn, setLoadOn } from '../../types'
import type { Bounds, BudgetUnit } from '../../types'
import { round } from '../../lib/curve'
import type { Field } from '../../lib/fields'
import { WORDS as words } from '../../words'

// --- Props & Emits ---
const props = defineProps<{
  field: Field
  state: PresetTabState
}>()

// --- State ---
const { bounds, settings } = props.state

const budgetUnits = BUDGET_UNITS.map((one) => ({ id: one, text: words.budgetUnitName(one) }))
const rules = RULES.map((one) => ({ id: one, text: words.ruleName(one) }))

const week = computed<readonly Day[]>(() =>
  WEEK.map((one) => ({ ...one, level: loadOn(settings.value.load, one.id) / WHOLE_LOAD })),
)

const levels = LOADS.map((one) => one / WHOLE_LOAD)

const fieldBounds = computed(() => boundsOf(props.field, bounds.value))
const fieldCount = computed(() => getFieldCount(props.field))

// --- Handlers ---
function onSelectLoad(day: string, level: number) {
  onChooseSetting('load', setLoadOn(settings.value.load, day, Math.round(level * WHOLE_LOAD)))
}

function onSelectRule(rule: string) {
  if (rule === 'interval' || rule === 'retention') onChooseSetting('learned', rule)
}

function onFieldType(field: Field, value: number | null) {
  if (value === null) return
  props.state.updateSetting(field, field === 'retention' ? round(value / 100, 2) : value)
}

function onFieldSettle() {
  props.state.settle()
}

function onChooseSetting(field: Field, value: SettingValue) {
  props.state.updateSetting(field, value)
  props.state.settle()
}

function onUpdateSlider(field: Field, share: number) {
  props.state.updateSetting(field, share)
}

function onToggleEvenLoad(on: boolean) {
  onChooseSetting('evenLoad', on)
}

function onChooseUnit(field: Field, unit: string) {
  onChooseSetting(field, unit as BudgetUnit)
}

function onDateChange(event: Event) {
  onChooseSetting('byDate', (event.target as HTMLInputElement).value)
}

// --- Helpers ---
type ControlBounds = { min: number; max: number } | Record<string, never>

function minMaxOf(one: Bounds | undefined, per = 1): ControlBounds {
  return one ? { min: one.least * per, max: one.most * per } : {}
}

function boundsOf(field: Field, within: typeof bounds.value): ControlBounds {
  if (field === 'newADay') return minMaxOf(within.newADay)
  if (field === 'reviewsADay') return minMaxOf(within.reviewsADay)
  if (field === 'minutesADay') return minMaxOf(within.minutesADay)
  if (field === 'backlog') return minMaxOf(within.backlog)
  if (field === 'interval') return minMaxOf(within.interval)
  if (field === 'retention') return minMaxOf(within.retention, 100)
  return {}
}

function getFieldCount(field: Field): number | null {
  if (field === 'newADay') return settings.value.newADay
  if (field === 'reviewsADay') return settings.value.reviewsADay
  if (field === 'retention') return Math.round(settings.value.retention * 100)
  if (field === 'minutesADay') return settings.value.minutesADay
  if (field === 'interval') return settings.value.interval
  return null
}
</script>

<template>
  <span class="preset-settings__value">
    <input
      v-if="field === 'byDate'"
      type="date"
      class="preset-settings__day"
      data-preset="day"
      :value="settings.byDate"
      :aria-labelledby="`preset-${field}`"
      @change="onDateChange"
    />
    <Select
      v-else-if="field === 'learned'"
      :model-value="settings.learned"
      :choices="rules"
      :name="words.fieldName('learned')"
      :aria-labelledby="`preset-${field}`"
      class="preset-settings__choice"
      data-preset="choice"
      @update:model-value="onSelectRule"
    />
    <SegmentedControl
      v-else-if="field === 'counts'"
      :model-value="settings.counts"
      :choices="budgetUnits"
      :aria-labelledby="`preset-${field}`"
      @update:model-value="(one: string) => onChooseUnit(field, one)"
    />
    <template v-else-if="field === 'backlog'">
      <Slider
        :model-value="settings.backlog"
        v-bind="fieldBounds"
        :step="1"
        :aria-labelledby="`preset-${field}`"
        class="preset-settings__slider"
        @update:model-value="(share: number) => onUpdateSlider(field, share)"
        @settles="onFieldSettle"
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
      @chooses="onSelectLoad"
    />
    <Switch
      v-else-if="field === 'evenLoad'"
      :model-value="settings.evenLoad"
      :aria-labelledby="`preset-${field}`"
      @update:model-value="onToggleEvenLoad"
    />
    <NumberField
      v-else
      :model-value="fieldCount"
      v-bind="fieldBounds"
      :step="1"
      :aria-labelledby="`preset-${field}`"
      class="preset-settings__number"
      @update:model-value="(value: number | null) => onFieldType(field, value)"
      @settles="onFieldSettle"
    />
  </span>
</template>

<style scoped>
.preset-settings__value {
  display: flex;
  align-items: center;
  justify-content: end;
  gap: var(--numen-node-gap);
}

.preset-settings__number {
  inline-size: var(--preset-settings-value, 6rem);
}

.preset-settings__slider {
  inline-size: var(--preset-settings-track, 9rem);
}

.preset-settings__percent {
  min-inline-size: var(--preset-settings-percent, 2.25rem);
  color: var(--numen-hushed);
  font-variant-numeric: tabular-nums;
  text-align: end;
}

.preset-settings__choice {
  inline-size: var(--preset-settings-choice, 10rem);
}

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
  outline: none;
  outline-offset: var(--numen-stroke);
}
</style>
