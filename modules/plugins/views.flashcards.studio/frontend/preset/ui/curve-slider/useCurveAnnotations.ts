import { computed, type ComputedRef } from 'vue'
import type { Position } from '@numen/ui'
import type { Curve, Goal } from '../../types'
import type { CurveSliderProps } from './useCurveSlider'
import type { useCurvePlot } from './useCurvePlot'
import { clearBacklog } from '../../lib/curve'
import {
  createCallout,
  createAxisLabels,
  createLabels,
  getKnobValueStyle,
  type Mark,
} from '../../lib/label'
import { getBacklogRun } from '../../lib/plot'
import { WORDS as words } from '../../words'

interface CalloutOptions {
  curve: Curve
  place: number
  position: Position | null
  value: number
}

function createCalloutData(options: CalloutOptions) {
  const { curve, place, position, value } = options
  const point = curve.at[place]
  if (!curve.isHonest || !position || !point) return null
  const backlog = getBacklogRun(curve, place)
  const lines = words.buys(curve.goal, {
    value,
    reviews: point.reviews,
    minutes: point.minutes,
    horizon: backlog.length,
    clears: clearBacklog(backlog),
    short: point.short,
    cards: curve.cards,
  })
  return { lines, ...createCallout(position) }
}

function createMarks(goal: Goal, knob: Position | null, suggested: Position | null): Mark[] {
  const out: Mark[] = []
  if (knob) out.push({ key: 'knob', at: knob, text: '' })
  if (suggested) {
    out.push({
      key: 'suggested',
      at: suggested,
      text: words.markName(goal),
    })
  }
  return out
}

function useCurveBounds(
  props: CurveSliderProps,
  grid: ComputedRef<readonly number[]>,
  lastIndex: ComputedRef<number>,
) {
  const atLeast = computed(() =>
    props.place === 0 ? '' : words.formatGoalValue(props.curve.goal, grid.value[0] ?? 0),
  )
  const atMost = computed(() =>
    props.place === lastIndex.value
      ? ''
      : words.formatGoalValue(props.curve.goal, grid.value[lastIndex.value] ?? 0),
  )
  const least = computed(() => grid.value[0] ?? 0)
  const most = computed(() => grid.value[lastIndex.value] ?? 0)
  return { atLeast, atMost, least, most }
}

export function useCurveAnnotations(
  props: CurveSliderProps,
  plot: ReturnType<typeof useCurvePlot>,
) {
  const isDated = computed(() => props.curve.goal === 'date')
  const isHonest = computed(() => props.curve.isHonest)

  const marks = computed(() =>
    isHonest.value ? createMarks(props.curve.goal, plot.knob.value, plot.suggested.value) : [],
  )

  const callout = computed(() =>
    createCalloutData({
      curve: props.curve,
      place: props.place,
      position: plot.knob.value,
      value: plot.held.value,
    }),
  )

  const named = computed(() => createLabels(marks.value, callout.value?.box ?? null))

  const heights = computed(() =>
    createAxisLabels(
      plot.extent.value,
      {
        places: plot.positions.value,
        marks: marks.value.map((one) => one.at),
        // Stryker disable next-line LogicalOperator: Null coalescing preserves box bounding rect when present.
        calloutBox: callout.value?.box ?? null,
      },
      (value: number) => words.formatGoalCost(props.curve.goal, value),
    ),
  )

  const knobStyle = computed(() => getKnobValueStyle(plot.knob.value))
  const atKnob = computed(() => words.formatGoalValue(props.curve.goal, plot.held.value))
  const counts = computed(() => (props.material ? words.material(props.material) : []))
  const learning = computed(() => {
    const point = props.curve.at[props.place]
    return point ? words.learning(point.learns, point.learned, props.curve.cards) : []
  })

  const grid = computed(() => props.curve.grid)
  const lastIndex = computed(() => plot.places.value - 1)
  const bounds = useCurveBounds(props, grid, lastIndex)

  return {
    isDated,
    isHonest,
    marks,
    callout,
    named,
    heights,
    reading: knobStyle,
    atKnob,
    counts,
    learning,
    ...bounds,
    value: computed(() => plot.held.value),
  }
}
