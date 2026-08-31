<script setup lang="ts">
/**
 * The one control of a preset: the goal's curve, drawn as the thing a person
 * drags along.
 *
 * The curve is the control. A pointer anywhere over the picture takes the
 * nearest place of the grid, and the knob rides the line there: the height is
 * read off the curve and never off the pointer. The whole picture is one stop
 * on the way round the screen, and the arrow keys walk the grid a place at a
 * time.
 *
 * Every word on the picture is HTML set over it, so the type is the page's and
 * not the picture's. Where two of them would touch, the one further down this
 * file's order gives way: a name is dropped rather than overprinted.
 */
import { computed, shallowRef, watch, useTemplateRef } from 'vue'
import { Waiting } from '@numen/ui'
import type { Curve } from './core'
import {
  AXIS_HIGH,
  BANDS,
  bandOf,
  clearAt,
  FOOT,
  HIGH,
  LABEL,
  LEFT,
  LIFT,
  lineOf,
  MIDDLE,
  placeUnder,
  RIGHT,
  shortOf,
  spotsOf,
  TOP,
  WIDE,
  yOfBand,
  type Band,
  type Spot,
} from './drawing'
import { WORDS as words } from './words'

const props = defineProps<{
  curve: Curve
  /** Where the knob stands, as a place of the curve's grid. */
  place: number
  /** What the knob is announced as standing at. */
  valueText: string
}>()

const raises = defineEmits<{
  moves: [place: number]
  settles: []
}>()

const picture = useTemplateRef<SVGSVGElement>('picture')

/**
 * The stretch of cost the picture is scaled to, taken from the whole grid of
 * the first answer this goal gave and kept while that goal is on screen. A
 * later answer is drawn against it, so the line moves and the axis does not.
 */
const scale = shallowRef<{ goal: string; band: Band } | null>(null)

watch(
  () => props.curve,
  (curve) => {
    if (scale.value?.goal === curve.goal) return
    scale.value = curve.honest ? { goal: curve.goal, band: bandOf(curve) } : null
  },
  { immediate: true },
)

const band = computed<Band>(() => scale.value?.band ?? bandOf(props.curve))

const spots = computed(() => spotsOf(props.curve, band.value))
const line = computed(() => lineOf(spots.value))
/** The stretch the budget does not get through, which is drawn quieter. */
const short = computed(() => shortOf(props.curve, spots.value))

const places = computed(() => props.curve.grid.length)
const knob = computed(() => spots.value[props.place] ?? null)
const suggested = computed(() => spots.value[props.curve.suggested.at] ?? null)

/** A goal of a date stands the mark of the day it names full height, and dashed. */
const dated = computed(() => props.curve.goal === 'date')

/** Whether this is the application's answer. The bands and the marks stand over that alone. */
const honest = computed(() => props.curve.honest)

/**
 * The two marks the picture carries, each with the name it is known by: where
 * the person stands, and what is suggested.
 */
const marks = computed(() => {
  if (!honest.value) return []
  const out: { key: string; spot: Spot; text: string }[] = []
  if (knob.value) out.push({ key: 'knob', spot: knob.value, text: words.now })
  if (suggested.value) {
    out.push({ key: 'suggested', spot: suggested.value, text: words.markName(props.curve.goal) })
  }
  return out
})

/**
 * Where a name over a mark is set: above it, pulled back inside the picture at
 * either end so the whole word stands over it.
 */
const naming = (spot: Spot) => {
  const back = spot.x < LEFT + LABEL ? '0' : spot.x > RIGHT - LABEL ? '-100%' : '-50%'
  return {
    insetInlineStart: `${(spot.x / WIDE) * 100}%`,
    insetBlockStart: `${(Math.max(spot.y - LIFT, TOP) / HIGH) * 100}%`,
    translate: `${back} -100%`,
  }
}

/**
 * The names that fit. They are taken in the order the marks stand in, and one
 * that would touch a name already placed is left off.
 */
const named = computed(() => {
  const placed: Spot[] = []
  const out: { key: string; text: string; at: Record<string, string> }[] = []
  for (const mark of marks.value) {
    const clear = placed.every(
      (one) =>
        Math.abs(one.x - mark.spot.x) > LABEL * 2 ||
        Math.abs(one.y - mark.spot.y) > AXIS_HIGH * 2,
    )
    if (!clear) continue
    placed.push(mark.spot)
    out.push({ key: mark.key, text: mark.text, at: naming(mark.spot) })
  }
  return out
})

/** Where a number against one of the picture's own lines is set. */
const against = (y: number, lift: string) => ({
  insetInlineStart: `${(LEFT / WIDE) * 100}%`,
  insetBlockStart: `${(y / HIGH) * 100}%`,
  translate: `0 ${lift}`,
})

/**
 * What the height of the picture comes to, against the lines it is read off. A
 * band of no width is one number and is said once, in the middle, where a curve
 * that never moves is drawn. A number the line or a mark stands on is dropped:
 * the axis gives way, and the drawing keeps what it has to say.
 */
const heights = computed(() => {
  const { least, most } = band.value
  const said = (value: number) => words.heightAt(props.curve.goal, value)
  const standing = marks.value.map((one) => one.spot)
  const fits = (y: number, lift: string, value: number) =>
    clearAt(y, spots.value, standing) ? [{ at: against(y, lift), text: said(value) }] : []
  // A band of no width has one number and nothing else to read, so it is set
  // over the line it names rather than given way to it.
  if (most === least) return [{ at: against(MIDDLE, '-100%'), text: said(most) }]
  return [...fits(TOP, '-100%', most), ...fits(FOOT, '0', least)]
})

/**
 * Where the knob's own value is set. It rides a line of its own under the
 * picture, so it never prints over a number read off the picture's edges.
 */
const reading = computed(() => {
  const spot = knob.value
  if (!spot) return {}
  const back = spot.x < LEFT + LABEL ? '0' : spot.x > RIGHT - LABEL ? '-100%' : '-50%'
  return { insetInlineStart: `${(spot.x / WIDE) * 100}%`, translate: `${back} 0` }
})

/** The value at the knob, and at either end of the range, in the goal's units. */
const atKnob = computed(() => words.widthAt(props.curve.goal, props.curve.grid[props.place] ?? 0))
/** An end the knob is standing on is left to the knob, which says it already. */
const atLeast = computed(() =>
  props.place === 0 ? '' : words.widthAt(props.curve.goal, props.curve.grid[0] ?? 0),
)
const atMost = computed(() =>
  props.place === places.value - 1
    ? ''
    : words.widthAt(props.curve.goal, props.curve.grid[places.value - 1] ?? 0),
)

const least = computed(() => props.curve.grid[0] ?? 0)
const most = computed(() => props.curve.grid[places.value - 1] ?? 0)
const value = computed(() => props.curve.grid[props.place] ?? 0)

/**
 * The place a pointer stands over. Where it stands is read through the
 * picture's own transform, so the place is the one under the pointer whatever
 * room the picture was given.
 */
const under = (event: PointerEvent): number => {
  const at = picture.value?.getScreenCTM?.()
  if (!at || at.a === 0) return props.place
  return placeUnder((event.clientX - at.e) / at.a, places.value)
}

const took = (event: PointerEvent) => {
  if (event.button !== 0) return
  event.preventDefault()
  picture.value?.focus()
  picture.value?.setPointerCapture(event.pointerId)
  raises('moves', under(event))
}

const dragged = (event: PointerEvent) => {
  if (!picture.value?.hasPointerCapture(event.pointerId)) return
  raises('moves', under(event))
}

const letGo = (event: PointerEvent) => {
  if (!picture.value?.hasPointerCapture(event.pointerId)) return
  picture.value.releasePointerCapture(event.pointerId)
  raises('settles')
}

/** Where a keystroke takes the knob, and nothing for a keystroke of somebody else's. */
const walked = (key: string): number | null => {
  const last = places.value - 1
  if (key === 'ArrowLeft' || key === 'ArrowDown') return Math.max(props.place - 1, 0)
  if (key === 'ArrowRight' || key === 'ArrowUp') return Math.min(props.place + 1, last)
  if (key === 'Home') return 0
  if (key === 'End') return last
  return null
}

const pressed = (event: KeyboardEvent) => {
  const step = walked(event.key)
  if (step === null) return
  event.preventDefault()
  raises('moves', step)
}

// The group is written once the key is let go of, so a held arrow key walks
// the grid and writes at the end of the walk.
const released = (event: KeyboardEvent) => {
  if (walked(event.key) === null) return
  raises('settles')
}
</script>

<template>
  <div class="control">
    <div class="control__frame">
      <!-- The y's name runs along the axis it names, outside the plot. -->
      <div class="control__axis">
        <p class="control__name control__name--y">{{ words.axisY(props.curve.goal) }}</p>
      </div>

      <div class="control__over">
        <div
          v-if="!honest"
          class="control__waiting"
          role="status"
          :style="{ aspectRatio: `${WIDE} / ${HIGH}` }"
        >
          <Waiting class="control__ring" />
          <span>{{ words.waiting }}</span>
        </div>

        <svg
          v-else
          ref="picture"
          class="control__picture"
          role="slider"
          tabindex="0"
          :viewBox="`0 0 ${WIDE} ${HIGH}`"
          :style="{ aspectRatio: `${WIDE} / ${HIGH}` }"
          :aria-label="words.knob"
          :aria-valuemin="least"
          :aria-valuemax="most"
          :aria-valuenow="value"
          :aria-valuetext="props.valueText"
          @pointerdown="took"
          @pointermove="dragged"
          @pointerup="letGo"
          @pointercancel="letGo"
          @keydown="pressed"
          @keyup="released"
        >
          <line
            v-for="share in honest ? BANDS : []"
            :key="share"
            class="control__band"
            :x1="LEFT"
            :x2="RIGHT"
            :y1="yOfBand(share)"
            :y2="yOfBand(share)"
          />

          <path class="control__line" :d="line" />
          <path v-if="honest && short" class="control__short" :d="short" />

          <line
            v-if="honest && knob"
            class="control__drop"
            :class="{ 'control__drop--dated': dated }"
            :x1="knob.x"
            :x2="knob.x"
            :y1="dated ? TOP : knob.y"
            :y2="FOOT"
          />

          <circle
            v-if="honest && suggested"
            class="control__suggested"
            :cx="suggested.x"
            :cy="suggested.y"
            r="3.5"
          />

          <circle v-if="knob" class="control__knob" :cx="knob.x" :cy="knob.y" r="7" />
        </svg>

        <span v-for="one in named" :key="one.key" class="control__label" :style="one.at">
          {{ one.text }}
        </span>

        <span
          v-for="one in honest ? heights : []"
          :key="one.text"
          class="control__number"
          :style="one.at"
          >{{ one.text }}</span
        >
      </div>
    </div>

    <!-- Every row keeps its room while the answer is on its way, so the picture
         is the only thing that changes when it lands. -->
    <div class="control__foot">
      <p class="control__under">
        <span v-if="honest" class="control__number control__number--knob" :style="reading">
          {{ atKnob }}
        </span>
      </p>

      <p class="control__ends">
        <span>{{ honest ? atLeast : '' }}</span>
        <span>{{ honest ? atMost : '' }}</span>
      </p>

      <p class="control__name control__name--x">{{ words.axisX(props.curve.goal) }}</p>

      <p v-if="honest" class="control__why">{{ words.markRule(props.curve.goal) }}</p>
    </div>
  </div>
</template>

<style scoped>
.control {
  /* One line of the small print the picture is annotated in, and the track the
     y's name runs along beside the plot. */
  --control-line: calc(var(--numen-text-1) * 1.4);
  --control-axis: calc(var(--numen-text-1) * 1.6);
  display: flex;
  flex-direction: column;
  gap: var(--numen-dot-gap);
}

/* The y's name beside the plot, and the plot. */
.control__frame {
  display: flex;
  align-items: stretch;
  gap: var(--numen-node-gap);
}

/*
 * The room the y's name runs in. It is a box of its own width and no height of
 * its own, so however long the name is the plot keeps its height.
 */
.control__axis {
  position: relative;
  flex: none;
  inline-size: var(--control-axis);
}

.control__name {
  margin: 0;
  color: var(--numen-hushed);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-1);
  line-height: 1;
}

/* Read up the picture, the way an axis is named on a chart. */
.control__name--y {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  writing-mode: vertical-rl;
  rotate: 180deg;
  overflow: hidden;
}

.control__name--x {
  text-align: center;
}

/* Everything read off the picture keeps the picture's own width. */
.control__foot {
  display: flex;
  flex-direction: column;
  gap: var(--numen-dot-gap);
  padding-inline-start: calc(var(--control-axis) + var(--numen-node-gap));
}

/* The picture, and what is named over it. */
.control__over {
  position: relative;
  flex: 1;
  min-inline-size: 0;
}

/*
 * The picture's room while the application works the curve out. It is the room
 * the picture takes, so nothing under it moves when the answer lands.
 */
.control__waiting {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--numen-node-gap);
  inline-size: 100%;
  color: var(--numen-hushed);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-1);
}

.control__ring {
  --waiting-size: 1em;
}

/* The line the knob's own value rides, clear of every number on the picture. */
.control__under {
  position: relative;
  margin: 0;
  block-size: var(--control-line);
}

.control__picture {
  display: block;
  inline-size: 100%;
  block-size: auto;
  touch-action: none;
  cursor: ew-resize;
  border-radius: var(--numen-radius);
}

.control__picture:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: var(--numen-caret);
}

.control__band {
  stroke: var(--numen-node-border);
  stroke-width: 1;
  stroke-dasharray: 2 5;
}

.control__line {
  fill: none;
  stroke: var(--numen-focus-bg);
  stroke-width: 2.25;
  stroke-linecap: round;
  stroke-linejoin: round;
}

/* The stretch a budget does not get through, which is still drawn. */
.control__short {
  fill: none;
  stroke: var(--numen-caution-fg);
  stroke-width: 1.25;
  stroke-linecap: round;
}

.control__drop {
  stroke: var(--numen-focus-bg);
  stroke-width: 1;
}

.control__drop--dated {
  stroke-dasharray: 4 4;
}

/* What is suggested: the accent again, filled and lighter. */
.control__suggested {
  fill: var(--numen-focus-bg);
}

/*
 * A number read off the picture: the ends of the band against the lines they
 * are the height of, and the value the knob stands at under it. The halo keeps
 * it legible where the line runs behind it.
 */
.control__number {
  position: absolute;
  padding-inline: var(--numen-edge-label-halo);
  color: var(--numen-hushed);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-1);
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
  white-space: nowrap;
  text-shadow:
    0 0 var(--numen-edge-label-halo) var(--numen-surface),
    0 0 var(--numen-edge-label-halo) var(--numen-surface);
  pointer-events: none;
}

/* The knob's own value, which reads out where the knob is dragged to. */
.control__number--knob {
  color: var(--numen-focus-bg);
}

/* The name of a mark, set over the picture in the colour of the mark it names. */
.control__label {
  position: absolute;
  color: var(--numen-focus-bg);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-1);
  line-height: 1;
  white-space: nowrap;
  text-shadow:
    0 0 var(--numen-edge-label-halo) var(--numen-surface),
    0 0 var(--numen-edge-label-halo) var(--numen-surface);
  pointer-events: none;
}

.control__knob {
  fill: var(--numen-focus-bg);
  stroke: var(--numen-surface);
  stroke-width: 2;
}

/* The two ends of the range, at the ends of the bottom edge. */
.control__ends {
  display: flex;
  justify-content: space-between;
  min-block-size: var(--control-line);
  margin: 0;
  padding-inline: var(--numen-inset);
  color: var(--numen-hushed);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-1);
  font-variant-numeric: tabular-nums;
}

/* What the suggested mark is chosen by, said under the picture and not on it. */
.control__why {
  margin: 0;
  color: var(--numen-hushed);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-1);
  line-height: var(--numen-line-height);
}
</style>
