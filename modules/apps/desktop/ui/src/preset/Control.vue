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
 */
import { computed, useTemplateRef } from 'vue'
import type { Curve } from './core'
import {
  areaOf,
  BANDS,
  FOOT,
  HIGH,
  LABEL,
  LEFT,
  LIFT,
  lineOf,
  placeUnder,
  RIGHT,
  shortOf,
  spotsOf,
  TOP,
  WIDE,
  yOfBand,
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

const spots = computed(() => spotsOf(props.curve))
const line = computed(() => lineOf(spots.value))
const area = computed(() => areaOf(spots.value))
/** The stretch the budget does not get through, which is drawn quieter. */
const short = computed(() => shortOf(props.curve, spots.value))

const places = computed(() => props.curve.grid.length)
const knob = computed(() => spots.value[props.place] ?? null)
const now = computed(() => spots.value[props.curve.now.at] ?? null)
const suggested = computed(() => spots.value[props.curve.suggested.at] ?? null)

/** A goal of a date stands the mark of the day it names full height, and dashed. */
const dated = computed(() => props.curve.goal === 'date')

/** Whether this is the application's answer. The bands and the marks stand over that alone. */
const honest = computed(() => props.curve.honest)

/** How the name of the suggested mark is set, so that it stays inside the picture. */
const naming = computed(() => {
  const spot = suggested.value
  if (!spot) return { x: 0, y: 0, anchor: 'middle' }
  const anchor = spot.x < LEFT + LABEL ? 'start' : spot.x > RIGHT - LABEL ? 'end' : 'middle'
  return { x: spot.x, y: Math.max(spot.y - LIFT, TOP), anchor }
})

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
    <svg
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
      <defs>
        <linearGradient id="preset-fade" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stop-color="var(--numen-focus-bg)" stop-opacity="0.22" />
          <stop offset="1" stop-color="var(--numen-focus-bg)" stop-opacity="0" />
        </linearGradient>
      </defs>

      <line
        v-for="band in honest ? BANDS : []"
        :key="band"
        class="control__band"
        :x1="LEFT"
        :x2="RIGHT"
        :y1="yOfBand(band)"
        :y2="yOfBand(band)"
      />

      <path class="control__area" :d="area" />
      <path class="control__line" :d="line" />
      <path v-if="honest && short" class="control__short" :d="short" />

      <g v-if="honest && now">
        <line
          class="control__drop"
          :class="{ 'control__drop--dated': dated }"
          :x1="now.x"
          :x2="now.x"
          :y1="dated ? TOP : now.y"
          :y2="FOOT"
        />
        <circle class="control__now" :cx="now.x" :cy="now.y" r="5" />
      </g>

      <g v-if="honest && suggested">
        <circle class="control__suggested" :cx="suggested.x" :cy="suggested.y" r="3.5" />
        <text
          class="control__label"
          :x="naming.x"
          :y="naming.y"
          :text-anchor="naming.anchor"
        >
          {{ words.suggested }}
        </text>
      </g>

      <circle v-if="knob" class="control__knob" :cx="knob.x" :cy="knob.y" r="7" />
    </svg>

    <p v-if="honest" class="control__ends">
      <span>{{ words.ends(props.curve.goal)[0] }}</span>
      <span>{{ words.ends(props.curve.goal)[1] }}</span>
    </p>
  </div>
</template>

<style scoped>
.control {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.control__picture {
  inline-size: 100%;
  block-size: auto;
  touch-action: none;
  cursor: ew-resize;
  border-radius: var(--numen-radius);
}

.control__picture:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: 2px;
}

.control__band {
  stroke: var(--numen-node-border);
  stroke-width: 1;
  stroke-dasharray: 2 5;
}

/* The accent under the line, thinning to nothing at the foot. */
.control__area {
  fill: url('#preset-fade');
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
  stroke: var(--numen-hushed);
  stroke-width: 1;
}

.control__drop--dated {
  stroke-dasharray: 4 4;
}

.control__now {
  fill: none;
  stroke: var(--numen-hushed);
  stroke-width: 1.5;
}

.control__suggested {
  fill: var(--numen-hushed);
}

.control__label {
  fill: var(--numen-hushed);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-1);
}

.control__knob {
  fill: var(--numen-focus-bg);
  stroke: var(--numen-surface);
  stroke-width: 2;
}

.control__ends {
  display: flex;
  justify-content: space-between;
  margin: 0;
  padding-inline: 0.5rem;
  color: var(--numen-hushed);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-1);
}
</style>
