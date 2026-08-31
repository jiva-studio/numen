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
import { computed, shallowRef, watch, useTemplateRef } from 'vue'
import type { Curve } from './core'
import {
  areaOf,
  BANDS,
  bandOf,
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
  type Band,
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

/**
 * Where the name of the suggested mark is set. It is set over the picture and
 * not in it, so it takes the page's type; the picture's own units place it, as
 * shares of the room the picture was given. It is pulled back inside at either
 * end so that the whole word stands over the picture.
 */
const naming = computed(() => {
  const spot = suggested.value
  if (!spot) return {}
  const back = spot.x < LEFT + LABEL ? '0' : spot.x > RIGHT - LABEL ? '-100%' : '-50%'
  return {
    insetInlineStart: `${(spot.x / WIDE) * 100}%`,
    insetBlockStart: `${(Math.max(spot.y - LIFT, TOP) / HIGH) * 100}%`,
    translate: `${back} -100%`,
  }
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
    <p class="control__height">{{ words.height(props.curve.goal) }}</p>

    <div class="control__over">
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
          v-for="share in honest ? BANDS : []"
          :key="share"
          class="control__band"
          :x1="LEFT"
          :x2="RIGHT"
          :y1="yOfBand(share)"
          :y2="yOfBand(share)"
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

        <circle
          v-if="honest && suggested"
          class="control__suggested"
          :cx="suggested.x"
          :cy="suggested.y"
          r="3.5"
        />

        <circle v-if="knob" class="control__knob" :cx="knob.x" :cy="knob.y" r="7" />
      </svg>

      <span v-if="honest && suggested" class="control__label" :style="naming">
        {{ words.suggested }}
      </span>
    </div>

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
  gap: var(--numen-dot-gap);
}

/* What the height of the picture is read in. */
.control__height {
  margin: 0;
  color: var(--numen-hushed);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-1);
}

/* The picture, and what is named over it. */
.control__over {
  position: relative;
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
  stroke: var(--numen-focus-bg);
  stroke-width: 1;
}

.control__drop--dated {
  stroke-dasharray: 4 4;
}

/* Where the preset stands: the accent, hollow. */
.control__now {
  fill: none;
  stroke: var(--numen-focus-bg);
  stroke-width: 1.5;
}

/* What is suggested: the accent again, filled and lighter. */
.control__suggested {
  fill: var(--numen-focus-bg);
}

/* The name of a mark, set over the picture in the colour of the mark it names. */
.control__label {
  position: absolute;
  color: var(--numen-focus-bg);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-1);
  line-height: 1;
  white-space: nowrap;
  pointer-events: none;
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
  padding-inline: var(--numen-inset);
  color: var(--numen-hushed);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-1);
}
</style>
