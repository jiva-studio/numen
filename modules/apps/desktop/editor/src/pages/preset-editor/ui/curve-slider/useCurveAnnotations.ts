import { computed } from 'vue'
import type { Curve, Position } from '../../types'
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
  at: Position | null
  value: number
}

function createCalloutData(options: CalloutOptions) {
  const { curve, place, at, value } = options
  const point = curve.at[place]
  if (!curve.isHonest || !at || !point) return null
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
  return { lines, ...createCallout(at) }
}

function createMarks(goal: string, knob: Position | null, suggested: Position | null): Mark[] {
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
      at: plot.knob.value,
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
        over: callout.value?.box ?? null,
      },
      (value) => words.formatGoalCost(props.curve.goal, value),
    ),
  )

  const reading = computed(() => getKnobValueStyle(plot.knob.value))
  const atKnob = computed(() => words.formatGoalValue(props.curve.goal, plot.held.value))
  const counts = computed(() => (props.material ? words.material(props.material) : []))
  const learning = computed(() => {
    const point = props.curve.at[props.place]
    return point ? words.learning(point.learns, point.learned, props.curve.cards) : []
  })

  const grid = computed(() => props.curve.grid)
  const lastIndex = computed(() => plot.places.value - 1)

  return {
    isDated,
    isHonest,
    marks,
    callout,
    named,
    heights,
    reading,
    atKnob,
    counts,
    learning,
    atLeast: computed(() =>
      props.place === 0 ? '' : words.formatGoalValue(props.curve.goal, grid.value[0] ?? 0),
    ),
    atMost: computed(() =>
      props.place === lastIndex.value
        ? ''
        : words.formatGoalValue(props.curve.goal, grid.value[lastIndex.value] ?? 0),
    ),
    least: computed(() => grid.value[0] ?? 0),
    most: computed(() => grid.value[lastIndex.value] ?? 0),
    value: computed(() => plot.held.value),
  }
}
