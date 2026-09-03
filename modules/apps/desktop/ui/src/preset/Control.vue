<script setup lang="ts">
/**
 * The one control of a preset: the goal's curve, drawn as the thing a person
 * drags along.
 *
 * A pointer anywhere over the picture takes the nearest place of the grid, and
 * the height is read off the curve and never off the pointer.
 *
 * Every word on the picture is HTML set over it. Where two of them touch, the
 * one further down this file's order gives way and its name is dropped.
 *
 * `data-control` names each part: `material`, `learned`, `tile`, `figure`,
 * `word`, `over`, `room`, `waiting`, `picture`, `rule`, `line`, `drop`,
 * `suggested`, `knob`, `label`, `number`, `perch`, `bought`, `tail`, `foot`,
 * `under`, `ends` and `name`. The picture is the slider; a name carries
 * `data-axis`, the reading at the knob carries `data-at-knob`, and a tail
 * turned under carries `data-under`.
 */
import { computed, shallowRef, watch, useTemplateRef, type CSSProperties } from 'vue'
import { Waiting } from '@numen/ui'
import type { Curve, Material } from './core'
import { clearing } from './curve'
import BacklogBand from './BacklogBand.vue'
import {
  against,
  againstBox,
  apart,
  AXIS_HIGH,
  BANDS,
  bandOf,
  clearAt,
  LABEL,
  FOOT,
  HIGH,
  LEFT,
  lineOf,
  naming,
  namingBox,
  PERCH_GAP,
  PERCH_HIGH,
  PERCH_WIDE,
  placeUnder,
  RIGHT,
  runAt,
  shortOf,
  spotsOf,
  TOP,
  walked,
  WIDE,
  yOfBand,
  type Band,
  type Box,
  type Spot,
} from './drawing'
import { WORDS as words } from './words'
import './control.css'

const props = defineProps<{
  curve: Curve
  /**
   * What the preset schedules, and nothing where it has never been counted.
   * These figures are facts about the material and stand under every answer, so
   * they are handed in beside the curve and not read off it.
   */
  material: Material | null
  /** Where the knob stands, as a place of the curve's grid. */
  place: number
  /** What the knob is announced as standing at. */
  valueText: string
  /** An answer to the picture is on its way. */
  waiting: boolean
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

/**
 * The value the knob stands at. A preset's own value need not sit on the grid,
 * and the place it opens at is the one nearest it, so while the knob has not
 * been moved off that place the preset's own value is what is said. A knob
 * walked anywhere else stands on a place, and the place is exact.
 */
const held = computed(() =>
  props.curve.now.at >= 0 && props.place === props.curve.now.at
    ? props.curve.now.value
    : (props.curve.grid[props.place] ?? 0),
)

/** A goal of a date stands the mark of the day it names full height, and dashed. */
const dated = computed(() => props.curve.goal === 'date')

/** Whether this is the application's answer. The bands and the marks stand over that alone. */
const honest = computed(() => props.curve.honest)

/**
 * The two marks the picture carries. The knob is where the person put it and
 * needs no name; the other one does, and carries it.
 */
const marks = computed(() => {
  if (!honest.value) return []
  const out: { key: string; spot: Spot; text: string }[] = []
  if (knob.value) out.push({ key: 'knob', spot: knob.value, text: '' })
  if (suggested.value) {
    out.push({ key: 'suggested', spot: suggested.value, text: words.markName(props.curve.goal) })
  }
  return out
})

/**
 * What this place of the curve buys, said in a bubble over the knob that moves
 * with it. It sits above the knob, and below it where above would take it off
 * the top, so it never covers the curve the knob is riding. The tail is
 * anchored on the knob itself and turns over with the bubble, so what the
 * numbers belong to is never in doubt.
 */
const perched = computed(() => {
  const spot = knob.value
  const point = props.curve.at[props.place]
  if (!honest.value || !spot || !point) return null
  const backlog = runAt(props.curve, props.place)
  const lines = words.buys(props.curve.goal, {
    value: held.value,
    reviews: point.reviews,
    minutes: point.minutes,
    horizon: backlog.length,
    clears: clearing(backlog),
    short: point.short,
    cards: props.curve.cards,
  })
  const under = spot.y - PERCH_GAP - PERCH_HIGH < TOP
  const half = PERCH_WIDE / 2
  const back = spot.x < LEFT + half ? 0 : spot.x > RIGHT - half ? PERCH_WIDE : half
  const edge = under ? spot.y + PERCH_GAP : spot.y - PERCH_GAP
  return {
    lines,
    under,
    box: {
      x: spot.x - back,
      y: under ? edge : edge - PERCH_HIGH,
      wide: PERCH_WIDE,
      high: PERCH_HIGH,
    },
    at: {
      insetInlineStart: `${(spot.x / WIDE) * 100}%`,
      insetBlockStart: `${(edge / HIGH) * 100}%`,
      translate: `${(-back / PERCH_WIDE) * 100}% ${under ? '0' : '-100%'}`,
    },
    tail: {
      insetInlineStart: `${(spot.x / WIDE) * 100}%`,
      insetBlockStart: `${(edge / HIGH) * 100}%`,
    },
  }
})

/**
 * The names that fit. They are taken in the order the marks stand in, and one
 * that would touch the readout over the knob, or a name already placed, is
 * left off.
 */
const named = computed(() => {
  const placed: Box[] = perched.value ? [perched.value.box] : []
  const out: { key: string; text: string; at: CSSProperties; box: Box }[] = []
  for (const mark of marks.value) {
    if (!mark.text) continue
    const box = namingBox(mark.spot)
    if (!placed.every((one) => apart(box, one))) continue
    placed.push(box)
    out.push({ key: mark.key, text: mark.text, at: naming(mark.spot), box })
  }
  return out
})

/**
 * What the height of the picture comes to, against the lines it is read off. A
 * band of no width is one number and is said once, on the foot, where a curve
 * that never moves is drawn. A number the line, a mark or the readout over the
 * knob stands on is dropped: the axis gives way, and the drawing keeps what it
 * has to say.
 */
const heights = computed(() => {
  const { least, most } = band.value
  const said = (value: number) => words.heightAt(props.curve.goal, value)
  const standing = marks.value.map((one) => one.spot)
  const over = perched.value?.box
  const fits = (y: number, lift: string, value: number) => {
    const box = againstBox(y, lift)
    if (!clearAt(y, spots.value, standing)) return []
    if (over && !apart(box, over)) return []
    return [{ at: against(y, lift), box, text: said(value) }]
  }
  // A band of no width has one number and nothing else to read, and it is set
  // over the line it names. That line is the foot, which is where a run with no
  // height is drawn.
  if (most === least) {
    return [{ at: against(FOOT, '0'), box: againstBox(FOOT, '0'), text: said(most) }]
  }
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
const atKnob = computed(() => words.widthAt(props.curve.goal, held.value))

/** What the control is acting on, in the pieces the row is scanned in. */
const figures = computed(() => {
  const one = props.material
  return one ? words.material(one.decks, one.cards, one.overdue, one.unbegun) : []
})
/**
 * When the material this place buys is learned, said as tiles under the
 * picture. It is read off the very point the line is drawn through, so the
 * picture and the tiles cannot disagree.
 */
const learning = computed(() => {
  const one = props.curve.at[props.place]
  if (!one) return []
  return words.learning(one.learns, one.learned, props.curve.cards)
})

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
const value = computed(() => held.value)

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

const pressed = (event: KeyboardEvent) => {
  const step = walked(event.key, props.place, places.value)
  if (step === null) return
  event.preventDefault()
  raises('moves', step)
}

// The group is written once the key is let go of, so a held arrow key walks
// the grid and writes at the end of the walk.
const released = (event: KeyboardEvent) => {
  if (walked(event.key, props.place, places.value) === null) return
  raises('settles')
}
</script>

<template>
  <div class="control">
    <!-- What the control is acting on, said before the picture of it. Each
         figure is its own tile, and the tiles share the width of the column.
         The figures stand while the answer to a new curve is on its way. -->
    <div class="control__material" data-control="material">
      <span v-for="one in figures" :key="one.name" class="control__tile" data-control="tile">
        <span class="control__figure" data-control="figure">{{ one.figure }}</span>
        <span class="control__word" data-control="word">{{ one.name }}</span>
      </span>
    </div>

    <!-- The whole chart as one block on the page: the plot, the band under it,
         and every name and number read off either. -->
    <div class="control__island">
      <div class="control__frame">
        <!-- The y's name runs along the axis it names, outside the plot. -->
        <div class="control__axis">
          <p class="control__name control__name--y" data-control="name" data-axis="y">
            {{ words.axisY(props.curve.goal) }}
          </p>
        </div>

        <div class="control__over" data-control="over">
          <!-- The room the plot is drawn in. It is the picture's own proportion
               whatever stands in it, so nothing below moves when the answer
               lands. -->
          <div
            class="control__room"
            data-control="room"
            :style="{ aspectRatio: `${WIDE} / ${HIGH}` }"
          >
            <!-- The room keeps its proportion where no line is drawn in it,
                 so nothing below moves. -->
            <div
              v-if="!honest && props.waiting"
              class="control__waiting"
              data-control="waiting"
              role="status"
            >
              <Waiting class="control__ring" />
              <span>{{ words.waiting }}</span>
            </div>

            <svg
              v-else-if="honest"
              ref="picture"
              class="control__picture"
              data-control="picture"
              role="slider"
              tabindex="0"
              :viewBox="`0 0 ${WIDE} ${HIGH}`"
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
                v-for="share in BANDS"
                :key="share"
                class="control__band"
                :x1="LEFT"
                :x2="RIGHT"
                :y1="yOfBand(share)"
                :y2="yOfBand(share)"
              />

              <!-- The two axes the figures are read against. -->
              <line
                class="control__rule"
                data-control="rule"
                :x1="LEFT"
                :x2="LEFT"
                :y1="TOP"
                :y2="FOOT"
              />
              <line
                class="control__rule"
                data-control="rule"
                :x1="LEFT"
                :x2="RIGHT"
                :y1="FOOT"
                :y2="FOOT"
              />

              <path class="control__line" data-control="line" :d="line" />
              <path v-if="short" class="control__short" :d="short" />

              <line
                v-if="knob"
                class="control__drop"
                data-control="drop"
                :x1="knob.x"
                :x2="knob.x"
                :y1="dated ? TOP : knob.y"
                :y2="FOOT"
              />

              <circle
                v-if="suggested"
                class="control__suggested"
                data-control="suggested"
                :cx="suggested.x"
                :cy="suggested.y"
                r="3.5"
              />

              <circle
                v-if="knob"
                class="control__knob"
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
            class="control__label"
            data-control="label"
            :style="one.at"
          >
            {{ one.text }}
          </span>

          <span
            v-for="(one, at) in honest ? heights : []"
            :key="at"
            class="control__number"
            data-control="number"
            :style="one.at"
            >{{ one.text }}</span
          >

          <!-- What this place buys, in a bubble over the knob, with its tail
               on the knob it belongs to. -->
          <template v-if="perched">
            <span class="control__perch" data-control="perch" :style="perched.at">
              <span
                v-for="one in perched.lines"
                :key="one"
                class="control__bought"
                data-control="bought"
                >{{ one }}</span
              >
            </span>
            <span
              class="control__tail"
              data-control="tail"
              :data-under="perched.under || undefined"
              :class="{ 'control__tail--under': perched.under }"
              :style="perched.tail"
            />
          </template>
        </div>
      </div>

      <!-- Every row keeps its room while the answer is on its way, so the
           picture is the only thing that changes when it lands. -->
      <div class="control__foot" data-control="foot">
        <p class="control__under" data-control="under">
          <span
            v-if="honest"
            class="control__number control__number--knob"
            data-control="number"
            data-at-knob
            :style="reading"
          >
            {{ atKnob }}
          </span>
        </p>

        <p class="control__ends" data-control="ends">
          <span>{{ honest ? atLeast : '' }}</span>
          <span>{{ honest ? atMost : '' }}</span>
        </p>

        <p class="control__name control__name--x" data-control="name" data-axis="x">
          {{ words.axisX(props.curve.goal) }}
        </p>
      </div>

      <BacklogBand :curve="props.curve" :place="props.place" :honest="honest" />
    </div>

    <!-- When the material is learned at the place the knob stands, read off
         the same run the picture is drawn from. -->
    <div class="control__material control__learned" data-control="learned">
      <span
        v-for="one in honest ? learning : []"
        :key="one.name"
        class="control__tile"
        data-control="tile"
      >
        <span class="control__figure" data-control="figure">{{ one.figure }}</span>
        <span class="control__word" data-control="word">{{ one.name }}</span>
      </span>
    </div>
  </div>
</template>

<style scoped>
.control {
  /* One line of the small print the picture is annotated in, and the track the
     y's name runs along beside the plot. */
  --control-line: calc(var(--numen-text-1) * 1.4);
  --control-axis: calc(var(--numen-text-1) * 1.6);
  /* The square the bubble's tail is turned out of. */
  --control-tail: 0.4375rem;
  /* How much taller a line box is than the letters standing in it. */
  --control-lead: 0.125rem;
  /* One tile of the readout, in the lengths the tile is built from. */
  --control-tile: calc(
    var(--numen-text-2) * 1.1 + var(--numen-dot-gap) + var(--numen-text-1) * 1.2 + 2 *
      var(--numen-inset) + 2 * var(--numen-stroke)
  );
  display: flex;
  flex-direction: column;
  /* The readout, the chart and what is read off it are parts of one block. */
  gap: var(--numen-node-gap);
}

/*
 * What the control is acting on, over the picture. It is a readout and is
 * scanned, so each figure is a tile of its own and the tiles take an equal
 * share of the width the tab is read at.
 */
/* The row of tiles draws a box, so its edge stands on the column's own. */
.control__material {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: 1fr;
  gap: var(--numen-node-gap);
  min-block-size: var(--control-tile);
  margin: 0;
}

/* One tile: the figure, and under it the word for what it counts. */
.control__tile {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: var(--numen-dot-gap);
  min-inline-size: 0;
  /* The lines inside stand on their own leading, which is taller than the
     letters, so the block clearance is trimmed by that difference and the four
     gaps read alike. */
  padding: calc(var(--numen-inset) - var(--control-lead)) var(--numen-box-air);
  border: var(--numen-stroke) solid var(--numen-node-border);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-node-bg);
}

.control__figure {
  color: var(--numen-node-fg);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  font-variant-numeric: tabular-nums;
  line-height: 1.1;
}

.control__word {
  color: var(--numen-hushed);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-1);
  line-height: 1;
  overflow-wrap: break-word;
}

/*
 * The block the whole chart sits on: the plot, the band under it, and every
 * name and number read off either. The surface stands around the plots and
 * adds nothing to the room inside them.
 */
.control__island {
  display: flex;
  flex-direction: column;
  gap: var(--numen-dot-gap);
  padding: var(--numen-box-air);
  border: var(--numen-stroke) solid var(--numen-node-border);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-node-bg);
}


/* The ring turning in the middle of that room, with the one line beside it. */
.control__waiting {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--numen-node-gap);
  inline-size: 100%;
  block-size: 100%;
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

/* Focus is shown on the knob, which is the thing the keyboard moves. */
.control__picture:focus-visible {
  outline: none;
}

.control__picture:focus-visible .control__knob {
  stroke: var(--numen-ring);
  stroke-width: 3;
}

.control__band {
  stroke: var(--numen-node-border);
  stroke-width: 1;
  stroke-dasharray: 2 5;
}

/*
 * The two axes a picture's figures are read against. A rule separates and does
 * not state, so it is drawn quieter and thinner than anything it measures.
 */
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

/* The line dropping from the knob, drawn one way under every goal. */
.control__drop {
  stroke: var(--numen-focus-bg);
  stroke-width: 1;
  stroke-dasharray: var(--numen-thread-dash);
}

/* What stands overdue over the days ahead, in the colour a debt is said in. */
/* What is suggested: the accent again, filled and lighter. */
.control__suggested {
  fill: var(--numen-focus-bg);
}

/* The knob's own value, which reads out where the knob is dragged to. */
.control__number--knob {
  color: var(--numen-focus-bg);
}

/*
 * What this place of the curve buys, in a bubble over the knob. It stands on a
 * ground of its own so the line behind it never reads through, and holds one
 * short line to a row.
 */
.control__perch {
  position: absolute;
  display: flex;
  flex-direction: column;
  gap: var(--numen-dot-gap);
  padding: var(--numen-inset);
  border: var(--numen-stroke) solid var(--numen-node-border);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-node-bg);
  box-shadow: var(--numen-shadow-card);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-1);
  line-height: 1.2;
  white-space: nowrap;
  pointer-events: none;
}

/*
 * The tail, aimed at the knob from the bubble's underside. It is a square
 * turned on its corner, showing the two faces that fall toward the knob, and
 * it turns over with the bubble.
 */
.control__tail {
  position: absolute;
  inline-size: var(--control-tail);
  block-size: var(--control-tail);
  border: var(--numen-stroke) solid var(--numen-node-border);
  border-block-start: 0;
  border-inline-start: 0;
  background: var(--numen-node-bg);
  rotate: 45deg;
  translate: -50% -50%;
  pointer-events: none;
}

/* Under the knob the bubble hangs below it, so the tail points up instead. */
.control__tail--under {
  border-block-start: var(--numen-stroke) solid var(--numen-node-border);
  border-inline-start: var(--numen-stroke) solid var(--numen-node-border);
  border-block-end: 0;
  border-inline-end: 0;
}

.control__bought {
  color: var(--numen-node-fg);
  font-variant-numeric: tabular-nums;
}

/* The value being held is the knob's own, and is said in the knob's colour. */
.control__bought:first-child {
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
    0 0 var(--numen-edge-label-halo) var(--numen-node-bg),
    0 0 var(--numen-edge-label-halo) var(--numen-node-bg);
  pointer-events: none;
}

.control__knob {
  fill: var(--numen-focus-bg);
  stroke: var(--numen-node-bg);
  stroke-width: 2;
}

/* The two ends of the range, at the ends of the bottom edge. */
</style>
