<script setup lang="ts">
/**
 * A preset tab: the one control at the top, and under it the settings its goal
 * schedules by.
 *
 * The control is the goal's curve. The goal owns one value and writes only
 * that; every other setting is the person's own and stands as they left it,
 * whatever the picture says it comes to.
 */
import { computed, ref } from 'vue'
import { Days, Menu, NumberField, Segmented, Slider, Switch } from '@numen/ui'
import type { Point } from '@numen/ui'
import { ChevronDown } from '@lucide/vue'
import Control from './Control.vue'
import type { Held, Said } from './kind'
import { BOUNDS, COUNTS, GOALS, RULES } from './core'
import type { Counts, Goal, Load, Rule } from './core'
import { fieldsUnder, idle, paused, spent, type Field } from './curve'
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
const fields = computed(() => fieldsUnder(settings.value.goal, settings.value.learned))

/** The three goals, as the switch above the curve offers them. */
const goals = computed(() => GOALS.map((one) => ({ id: one, text: words.goalName(one) })))

/** The two things a budget is spent on, as the row offers them. */
const counts = COUNTS.map((one) => ({ id: one, text: words.countsName(one) }))

/** The two rules for what is learned, as the row offers them. */
const rules = RULES.map((one) => ({ id: one, text: words.ruleName(one) }))

/** Where the rules were asked for, and nothing while they are not. */
const asking = ref<Point | null>(null)

/** The line the row stands on opens the rules under itself. */
const asks = (event: Event) => {
  const line = event.currentTarget
  if (!(line instanceof HTMLElement)) return
  const box = line.getBoundingClientRect()
  asking.value = { x: box.left, y: box.bottom }
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

/** How far a field goes. A field that holds no number is bounded by nothing. */
const boundsOf = (field: Field): { least: number; most: number } => {
  if (field === 'newADay') return BOUNDS.newADay
  if (field === 'reviewsADay') return BOUNDS.reviewsADay
  if (field === 'retention') return BOUNDS.retention
  if (field === 'minutesADay') return BOUNDS.minutesADay
  if (field === 'backlog') return BOUNDS.backlog
  if (field === 'interval') return BOUNDS.interval
  return { least: 0, most: 0 }
}

/** Whether a row draws a number, and the number it draws. */
const counted = (field: Field): number | null => {
  if (field === 'newADay') return settings.value.newADay
  if (field === 'reviewsADay') return settings.value.reviewsADay
  if (field === 'retention') return settings.value.retention
  if (field === 'minutesADay') return settings.value.minutesADay
  if (field === 'interval') return settings.value.interval
  return null
}

const stepOf = (field: Field): number => (field === 'retention' ? 0.01 : 1)

/** A number typed into a row. An empty field leaves the setting as it stands. */
const typed = (field: Field, said: number | null) => {
  if (said === null) return
  props.held.types(field, said)
}

/**
 * A row a person is done with, which is what writes the group. A control moved
 * a step at a time says so when it is let go of; one that turns in a single
 * gesture is done the moment it turns.
 */
const chose = (field: Field, value: Said) => {
  props.held.types(field, value)
  props.held.settles()
}

/** A day typed into the row that holds one. */
const dated = (said: Event) => {
  chose('byDate', (said.target as HTMLInputElement).value)
}

/** Whether the preset schedules nothing, and why. */
const stopped = computed(() => {
  const today = new Date()
  if (spent(settings.value, today)) return words.spent
  return paused(settings.value, today) ? words.paused : ''
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
            class="preset__goals"
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
              :waiting="props.held.waiting()"
              @moves="(at: number) => props.held.moves(at)"
              @settles="props.held.settles()"
            />
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
              <!-- One line saying what the rule is, and the rules under it
                   when it is asked. -->
              <button
                v-else-if="field === 'learned'"
                type="button"
                class="preset__choice"
                aria-haspopup="menu"
                :aria-labelledby="`preset-${field}`"
                @click="asks"
              >
                {{ words.ruleName(settings.learned) }}
                <ChevronDown class="preset__icon" aria-hidden="true" />
              </button>
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
                  :min="boundsOf(field).least"
                  :max="boundsOf(field).most"
                  :step="stepOf(field)"
                  :aria-labelledby="`preset-${field}`"
                  class="preset__slider"
                  @update:model-value="(share: number) => props.held.types(field, share)"
                  @settles="props.held.settles()"
                />
                <span class="preset__percent">{{ words.percent(settings.backlog) }}</span>
              </template>
              <Days
                v-else-if="field === 'load'"
                :model-value="settings.load"
                :aria-labelledby="`preset-${field}`"
                @update:model-value="(load: Load) => chose(field, load)"
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
                :min="boundsOf(field).least"
                :max="boundsOf(field).most"
                :step="stepOf(field)"
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

    <Menu
      v-if="asking"
      :items="rules"
      :at="asking"
      open
      opening="keyboard"
      :name="words.fieldName('learned')"
      @choose="ruled"
      @dismiss="asking = null"
    />
  </div>
</template>

<style scoped>
.preset {
  /* The measure a preset is read at, which the goal and the settings share. */
  --preset-measure: 46rem;
  /* Where every line of this tab begins, measured from the column: the
     clearance a box keeps inside its own border. A block that draws a box is
     pulled out by that border so what it holds lands on the line, and a block
     that draws none is inset to it. */
  --preset-ink: var(--numen-box-air);
  /* One row of the receipt: the box a number is typed into, the air around the
     row, and the space between what it is called and what it means. */
  --preset-value: 6rem;
  --preset-row-air: 0.5rem;
  --preset-said-gap: 0.125rem;
  /* The clearance typing keeps from the edges of its box. A field keeps a line
     box taller than the digits in it, so the block clearance is set under the
     inline one by that difference and the four gaps read alike. */
  --preset-field-inset: var(--numen-field-padding) var(--numen-inset);
  /* The mark a row carries beside its small print. */
  --preset-icon: 0.875rem;
  /* The track a share is moved along, and the room the figure beside it takes. */
  --preset-track: 9rem;
  --preset-percent: 2.25rem;
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
  gap: var(--numen-step);
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

/* The switch spans the column and hangs out at both ends by its own border and
   the hairline inside it, so what stands on its first segment begins on the
   line every line of this tab begins on and its far end reads level with the
   blocks under it. */
.preset__goals {
  margin-inline: calc(-2 * var(--numen-stroke));
}

/* What the three segments are, said over them. */
.preset__label {
  margin: 0;
  padding-inline: var(--preset-ink);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-weight: 600;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

/* No deck points here, said where the curve would stand. */
.preset__unpointed {
  margin: 0;
  padding-inline: var(--preset-ink);
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
  padding-inline-start: calc(var(--preset-ink) - var(--numen-caret));
  padding-inline-end: var(--preset-ink);
  border-inline-start: var(--numen-caret) solid transparent;
  border-block-end: var(--numen-stroke) solid var(--numen-node-border);
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
  color: var(--numen-node-fg);
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

/* The rule stands on a line of its own, drawn as the boxes a value is typed
   into are, with the mark that says it opens. */
/* A line of text stands shorter than a field a number is typed into, so this
   one is held to the height every control on a row shares. */
.preset__choice {
  display: inline-flex;
  align-items: center;
  gap: var(--numen-node-gap);
  min-block-size: var(--numen-action-size);
  padding: var(--preset-field-inset);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-field-bg);
  color: inherit;
  font: inherit;
  line-height: 1;
  cursor: pointer;
}

.preset__choice:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: var(--numen-stroke);
}

/* The same box the numbers of the receipt are typed into. */
.preset__day {
  padding: var(--preset-field-inset);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-field-bg);
  color: inherit;
  font: inherit;
  line-height: 1;
}

.preset__day:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: var(--numen-stroke);
}

.preset__detail {
  color: var(--numen-hushed);
}

/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.preset__icon {
  inline-size: var(--preset-icon);
  block-size: var(--preset-icon);
  stroke-width: 1.875;
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
