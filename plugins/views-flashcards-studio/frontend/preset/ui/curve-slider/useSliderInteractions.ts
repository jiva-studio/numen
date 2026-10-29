import { ref, type ComputedRef } from 'vue'
import { getPlaceAtX, getKeyPlace } from '../../lib/plot'

export interface SliderInteractionDeps {
  getPlace: () => number
  places: ComputedRef<number>
  emit: {
    (event: 'move', place: number): void
    (event: 'commit'): void
  }
}

export function useSliderInteractions({ getPlace, places, emit }: SliderInteractionDeps) {
  const picture = ref<SVGSVGElement | null>(null)

  function getPlaceUnder(event: PointerEvent): number {
    const ctm = picture.value?.getScreenCTM?.()
    if (!ctm || ctm.a === 0) return getPlace()
    return getPlaceAtX((event.clientX - ctm.e) / ctm.a, places.value)
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
    emit('commit')
  }

  function onKeyDown(event: KeyboardEvent): void {
    const step = getKeyPlace(event.key, getPlace(), places.value)
    if (step === null) return
    event.preventDefault()
    emit('move', step)
  }

  function onKeyUp(event: KeyboardEvent): void {
    if (getKeyPlace(event.key, getPlace(), places.value) === null) return
    emit('commit')
  }

  function setChart(element: SVGSVGElement | null): void {
    picture.value = element
  }

  return {
    picture,
    setChart,
    onPointerDown,
    onPointerMove,
    onPointerUp,
    onKeyDown,
    onKeyUp,
  }
}
