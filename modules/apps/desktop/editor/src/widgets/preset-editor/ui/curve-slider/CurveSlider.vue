<script setup lang="ts">
/**
 * The one slider of a preset: the goal's curve, drawn as the track a person
 * drags the knob along.
 */
import { Spinner } from '@numen/ui'
import type { Curve, PresetCounts } from '../../types'
import BacklogPlot from './backlog-plot/BacklogPlot.vue'
import CurveTiles from './CurveTiles.vue'
import { FOOT, GRIDLINES, HIGH, LEFT, RIGHT, TOP, WIDE, yOfGridline } from '../../plot'
import { WORDS as words } from '../../words'
import { useCurveSlider } from './useCurveSlider'
import './curve-slider.css'

// --- Props & Emits ---
const props = defineProps<{
  curve: Curve
  material: PresetCounts | null
  place: number
  valueText: string
  waiting: boolean
}>()

const emit = defineEmits<{
  (event: 'moves', place: number): void
  (event: 'settles'): void
}>()

// --- State ---
const {
  picture, line, short, knob, suggested, isDated, isHonest,
  named, heights, reading, atKnob, counts, learning,
  atLeast, atMost, least, most, value, callout,
  onPointerDown, onPointerMove, onPointerUp, onKeyDown, onKeyUp,
} = useCurveSlider(props, emit)

// --- Handlers ---

// --- Helpers ---
</script>

<template>
  <div class="curve-slider">
    <CurveTiles control="material" :tiles="counts" />

    <div class="curve-slider__island">
      <div class="curve-slider__frame">
        <div class="curve-slider__axis">
          <p class="curve-slider__name curve-slider__name--y" data-control="name" data-axis="y">
            {{ words.axisY(props.curve.goal) }}
          </p>
        </div>

        <div class="curve-slider__over" data-control="over">
          <div
            class="curve-slider__room"
            data-control="room"
            :style="{ aspectRatio: `${WIDE} / ${HIGH}` }"
          >
            <div
              v-if="!isHonest && props.waiting"
              class="curve-slider__waiting"
              data-control="waiting"
              role="status"
            >
              <Spinner class="curve-slider__ring" />
              <span>{{ words.waiting }}</span>
            </div>

            <svg
              v-else-if="isHonest"
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
          </div>

          <span
            v-for="one in named"
            :key="one.key"
            class="curve-slider__label"
            data-control="label"
            :style="one.at"
          >
            {{ one.text }}
          </span>

          <span
            v-for="(one, at) in isHonest ? heights : []"
            :key="at"
            class="curve-slider__number"
            data-control="number"
            :style="one.at"
          >
            {{ one.text }}
          </span>

          <template v-if="callout">
            <span class="curve-slider__callout" data-control="callout" :style="callout.at">
              <span
                v-for="one in callout.lines"
                :key="one"
                class="curve-slider__bought"
                data-control="bought"
              >{{ one }}</span>
            </span>
            <span
              class="curve-slider__tail"
              data-control="tail"
              :data-under="callout.under || undefined"
              :class="{ 'curve-slider__tail--under': callout.under }"
              :style="callout.tail"
            />
          </template>
        </div>
      </div>

      <div class="curve-slider__foot" data-control="foot">
        <p class="curve-slider__under" data-control="under">
          <span
            v-if="isHonest"
            class="curve-slider__number curve-slider__number--knob"
            data-control="number"
            data-at-knob
            :style="reading"
          >
            {{ atKnob }}
          </span>
        </p>

        <p class="curve-slider__ends" data-control="ends">
          <span>{{ isHonest ? atLeast : '' }}</span>
          <span>{{ isHonest ? atMost : '' }}</span>
        </p>

        <p class="curve-slider__name curve-slider__name--x" data-control="name" data-axis="x">
          {{ words.axisX(props.curve.goal) }}
        </p>
      </div>

      <BacklogPlot :curve="props.curve" :place="props.place" :honest="isHonest" />
    </div>

    <CurveTiles control="learned" :tiles="isHonest ? learning : []" />
  </div>
</template>

<style scoped>
.curve-slider__island {
  display: flex;
  flex-direction: column;
  gap: var(--numen-dot-gap);
  padding: var(--numen-box-air);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-raised);
}
</style>
