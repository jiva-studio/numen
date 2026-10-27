import { computed, shallowRef, watch } from 'vue'
import type { Curve } from '../../types'
import { getKnobValue } from '../../lib/curve'
import {
  getExtent,
  createLinePath,
  createPositions,
  getShortPath,
  type Extent,
} from '../../lib/plot'

export function useCurveScale(getCurve: () => Curve) {
  const scale = shallowRef<{ goal: string; extent: Extent } | null>(null)

  watch(
    getCurve,
    (curve) => {
      if (
        scale.value?.goal === curve.goal &&
        (!curve.isHonest || getExtent(curve).max <= scale.value.extent.max)
      ) {
        return
      }
      scale.value = curve.isHonest ? { goal: curve.goal, extent: getExtent(curve) } : null
    },
    { immediate: true },
  )

  const extent = computed<Extent>(() => scale.value?.extent ?? getExtent(getCurve()))
  return { scale, extent }
}

export function useCurvePlot(getCurve: () => Curve, getPlace: () => number) {
  const { extent } = useCurveScale(getCurve)
  const positions = computed(() => createPositions(getCurve(), extent.value))
  const line = computed(() => createLinePath(positions.value))
  const short = computed(() => getShortPath(getCurve(), positions.value))

  const places = computed(() => getCurve().grid.length)
  const knob = computed(() => positions.value[getPlace()] ?? null)
  const suggested = computed(() => positions.value[getCurve().suggested.index] ?? null)
  const target = computed(() => getKnobValue(getCurve(), getPlace()))

  return {
    extent,
    positions,
    line,
    short,
    places,
    knob,
    suggested,
    heldValue: target,
  }
}
