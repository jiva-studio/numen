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

function createCalloutData(curve: Curve, place: number, at: Position | null, value: number) {
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

export function useCurveAnnotations(
  props: CurveSliderProps,
  plot: ReturnType<typeof useCurvePlot>,
) {
  const isDated = computed(() => props.curve.goal === 'date')
  const isHonest = computed(() => props.curve.isHonest)

  const marks = computed(() => {
    if (!isHonest.value) return []
    const out: Mark[] = []
    if (plot.knob.value) out.push({ key: 'knob', at: plot.knob.value, text: '' })
    if (plot.suggested.value) {
      out.push({
        key: 'suggested',
        at: plot.suggested.value,
        text: words.markName(props.curve.goal),
      })
    }
    return out
  })

  const callout = computed(() =>
    createCalloutData(props.curve, props.place, plot.knob.value, plot.held.value),
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

  const atLeast = computed(() =>
    props.place === 0 ? '' : words.formatGoalValue(props.curve.goal, props.curve.grid[0] ?? 0),
  )
  const atMost = computed(() =>
    props.place === plot.places.value - 1
      ? ''
      : words.formatGoalValue(props.curve.goal, props.curve.grid[plot.places.value - 1] ?? 0),
  )

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
    atLeast,
    atMost,
    least: computed(() => props.curve.grid[0] ?? 0),
    most: computed(() => props.curve.grid[plot.places.value - 1] ?? 0),
    value: computed(() => plot.held.value),
  }
}
