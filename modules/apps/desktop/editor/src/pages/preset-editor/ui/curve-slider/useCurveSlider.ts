import { computed, shallowRef, watch, ref } from 'vue'
import type { Curve, PresetCounts } from '../../types'
import { clearBacklog, valueAt } from '../../lib/curve'
import { calloutOf, heightsOf, labelsOf, readingAt, type Mark } from '../../lib/label'
import {
  extentOf,
  lineOf,
  placeUnder,
  positionsOf,
  runAt,
  shortOf,
  walkGrid,
  type Extent,
} from '../../lib/plot'
import { WORDS as words } from '../../words'

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
    (event: 'settle'): void
  },
) {
  const picture = ref<SVGSVGElement | null>(null)

  const scale = shallowRef<{ goal: string; extent: Extent } | null>(null)

  watch(
    () => props.curve,
    (curve) => {
      if (scale.value?.goal === curve.goal) return
      scale.value = curve.isHonest ? { goal: curve.goal, extent: extentOf(curve) } : null
    },
    { immediate: true },
  )

  const extent = computed<Extent>(() => scale.value?.extent ?? extentOf(props.curve))
  const positions = computed(() => positionsOf(props.curve, extent.value))
  const line = computed(() => lineOf(positions.value))
  const short = computed(() => shortOf(props.curve, positions.value))

  const places = computed(() => props.curve.grid.length)
  const knob = computed(() => positions.value[props.place] ?? null)
  const suggested = computed(() => positions.value[props.curve.suggested.at] ?? null)

  const held = computed(() => valueAt(props.curve, props.place))
  const isDated = computed(() => props.curve.goal === 'date')
  const isHonest = computed(() => props.curve.isHonest)

  const marks = computed(() => {
    if (!isHonest.value) return []
    const out: Mark[] = []
    if (knob.value) out.push({ key: 'knob', at: knob.value, text: '' })
    if (suggested.value) {
      out.push({ key: 'suggested', at: suggested.value, text: words.markName(props.curve.goal) })
    }
    return out
  })

  const callout = computed(() => {
    const at = knob.value
    const point = props.curve.at[props.place]
    if (!isHonest.value || !at || !point) return null
    const backlog = runAt(props.curve, props.place)
    const lines = words.buys(props.curve.goal, {
      value: held.value,
      reviews: point.reviews,
      minutes: point.minutes,
      horizon: backlog.length,
      clears: clearBacklog(backlog),
      short: point.short,
      cards: props.curve.cards,
    })
    return { lines, ...calloutOf(at) }
  })

  const named = computed(() => labelsOf(marks.value, callout.value?.box ?? null))

  const heights = computed(() =>
    heightsOf(
      extent.value,
      positions.value,
      marks.value.map((one) => one.at),
      callout.value?.box ?? null,
      (value) => words.heightAt(props.curve.goal, value),
    ),
  )

  const reading = computed(() => readingAt(knob.value))
  const atKnob = computed(() => words.widthAt(props.curve.goal, held.value))

  const counts = computed(() => {
    const one = props.material
    return one ? words.material(one.decks, one.cards, one.overdue, one.unbegun) : []
  })

  const learning = computed(() => {
    const point = props.curve.at[props.place]
    if (!point) return []
    return words.learning(point.learns, point.learned, props.curve.cards)
  })

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

  function getPlaceUnder(event: PointerEvent): number {
    const at = picture.value?.getScreenCTM?.()
    if (!at || at.a === 0) return props.place
    return placeUnder((event.clientX - at.e) / at.a, places.value)
  }

  function onPointerDown(event: PointerEvent): void {
    if (event.button !== 0) return
    event.preventDefault()
    picture.value?.focus()
    picture.value?.setPointerCapture(event.pointerId)
    emit('move', getPlaceUnder(event))
  }

  function onPointerMove(event: PointerEvent): void {
    if (!picture.value?.hasPointerCapture(event.pointerId)) return
    emit('move', getPlaceUnder(event))
  }

  function onPointerUp(event: PointerEvent): void {
    if (!picture.value?.hasPointerCapture(event.pointerId)) return
    picture.value.releasePointerCapture(event.pointerId)
    emit('settle')
  }

  function onKeyDown(event: KeyboardEvent): void {
    const step = walkGrid(event.key, props.place, places.value)
    if (step === null) return
    event.preventDefault()
    emit('move', step)
  }

  function onKeyUp(event: KeyboardEvent): void {
    if (walkGrid(event.key, props.place, places.value) === null) return
    emit('settle')
  }

  function setPicture(element: SVGSVGElement | null): void {
    picture.value = element
  }

  return {
    picture,
    setPicture,
    line,
    short,
    places,
    knob,
    suggested,
    held,
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
    least,
    most,
    value,
    onPointerDown,
    onPointerMove,
    onPointerUp,
    onKeyDown,
    onKeyUp,
  }
}
