<script setup lang="ts">
/**
 * The curve itself, and the knob dragged along it. It is the slider a person
 * points at, so it takes the pointer and the keys.
 */
import { FOOT, GRIDLINES, HIGH, LEFT, RIGHT, TOP, WIDE, yOfGridline } from '../../../../lib/plot'
import { WORDS as words } from '../../../../words'
import type { CurveSliderState } from '../../useCurveSlider'
import '../../curve-slider.css'

// --- Props & Emits ---
const props = defineProps<{
  state: CurveSliderState
  /** The value the knob stands at, said in the units of the goal. */
  valueText: string
}>()

// --- State ---
const {
  picture, line, short, knob, suggested, isDated, least, most, value,
  onPointerDown, onPointerMove, onPointerUp, onKeyDown, onKeyUp,
} = props.state

// --- Handlers ---

// --- Helpers ---
</script>

<template>
  <svg
    ref="picture"
    class="curve-slider__picture"
    data-control="picture"
    role="slider"
    tabindex="0"
    :viewBox="`0 0 ${WIDE} ${HIGH}`"
    :aria-label="words.knob"
    :aria-valuemin="least"
    :aria-valuemax="most"
    :aria-valuenow="value"
    :aria-valuetext="props.valueText"
    @pointerdown="onPointerDown"
    @pointermove="onPointerMove"
    @pointerup="onPointerUp"
    @pointercancel="onPointerUp"
    @keydown="onKeyDown"
    @keyup="onKeyUp"
  >
    <line
      v-for="share in GRIDLINES"
      :key="share"
      class="curve-slider__gridline"
      :x1="LEFT"
      :x2="RIGHT"
      :y1="yOfGridline(share)"
      :y2="yOfGridline(share)"
    />

    <line class="curve-slider__rule" data-control="rule" :x1="LEFT" :x2="LEFT" :y1="TOP" :y2="FOOT" />
    <line class="curve-slider__rule" data-control="rule" :x1="LEFT" :x2="RIGHT" :y1="FOOT" :y2="FOOT" />

    <path class="curve-slider__line" data-control="line" :d="line" />
    <path v-if="short" class="curve-slider__short" :d="short" />

    <line
      v-if="knob"
      class="curve-slider__drop"
      data-control="drop"
      :x1="knob.x"
      :x2="knob.x"
      :y1="isDated ? TOP : knob.y"
      :y2="FOOT"
    />

    <circle
      v-if="suggested"
      class="curve-slider__suggested"
      data-control="suggested"
      :cx="suggested.x"
      :cy="suggested.y"
      r="3.5"
    />

    <circle
      v-if="knob"
      class="curve-slider__knob"
      data-control="knob"
      :cx="knob.x"
      :cy="knob.y"
      r="7"
    />
  </svg>
</template>
