import type { Curve, PresetCounts } from '../../types'
import { useCurvePlot } from './useCurvePlot'
import { useSliderInteractions } from './useSliderInteractions'
import { useCurveAnnotations } from './useCurveAnnotations'

export interface CurveSliderProps {
  curve: Curve
  material: PresetCounts | null
  place: number
  valueText: string
  isWaiting: boolean
}

/** Everything the slider draws itself from, as its parts read it. */
export type CurveSliderState = ReturnType<typeof useCurveSlider>

export function useCurveSlider(
  props: CurveSliderProps,
  emit: {
    (event: 'move', place: number): void
    (event: 'commit'): void
  },
) {
  const plot = useCurvePlot(
    () => props.curve,
    () => props.place,
  )
  const interactions = useSliderInteractions({
    getPlace: () => props.place,
    places: plot.places,
    emit,
  })
  const annotations = useCurveAnnotations(props, plot)

  return {
    plot,
    interactions,
    annotations,
  }
}
