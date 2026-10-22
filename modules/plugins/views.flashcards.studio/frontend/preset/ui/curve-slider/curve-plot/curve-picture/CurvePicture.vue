<script setup lang="ts">
/**
 * The curve itself, and the knob dragged along it. It is the slider a person
 * points at, so it takes the pointer and the keys.
 */
import {
  PLOT_BOTTOM,
  GRIDLINES,
  PLOT_HEIGHT,
  PLOT_LEFT,
  PLOT_RIGHT,
  PLOT_TOP,
  PLOT_WIDTH,
  yOfGridline,
} from '../../../../lib/plot'
import { WORDS as words } from '../../../../words'
import type { CurveSliderState } from '../../useCurveSlider'
import '../../curve-slider.css'

// --- Props & Emits ---
const props = defineProps<{
  state: CurveSliderState
  valueText: string
}>()

// --- State ---
const {
  line,
  short,
  knob,
  suggested,
  isDated,
  least,
  most,
  value,
  setPicture,
  onPointerDown,
  onPointerMove,
  onPointerUp,
  onKeyDown,
  onKeyUp,
} = props.state

// --- Handlers ---

// --- Helpers ---
</script>

<template>
  <svg
    :ref="
      (el: unknown) => {
        setPicture(el as SVGSVGElement | null)
      }
    "
    class="curve-slider__picture"
    data-control="picture"
    role="slider"
    tabindex="0"
    :viewBox="`0 0 ${PLOT_WIDTH} ${PLOT_HEIGHT}`"
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
      :x1="PLOT_LEFT"
      :x2="PLOT_RIGHT"
      :y1="yOfGridline(share)"
      :y2="yOfGridline(share)"
    />

    <line
      class="curve-slider__rule"
      data-control="rule"
      :x1="PLOT_LEFT"
      :x2="PLOT_LEFT"
      :y1="PLOT_TOP"
      :y2="PLOT_BOTTOM"
    />
    <line
      class="curve-slider__rule"
      data-control="rule"
      :x1="PLOT_LEFT"
      :x2="PLOT_RIGHT"
      :y1="PLOT_BOTTOM"
      :y2="PLOT_BOTTOM"
    />

    <path class="curve-slider__line" data-control="line" :d="line" />
    <path v-if="short" class="curve-slider__short" :d="short" />

    <line
      v-if="knob"
      class="curve-slider__drop"
      data-control="drop"
      :x1="knob.x"
      :x2="knob.x"
      :y1="isDated ? PLOT_TOP : knob.y"
      :y2="PLOT_BOTTOM"
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
