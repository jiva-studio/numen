/**
 * How the strip of three panels moves: its stops, where a pointer release
 * leaves it, and where it moves from there.
 *
 * The strip is as wide as all three panels, scrolled to the visible one, and
 * it settles on the nearest of the three when a pointer is released.
 */
import { win } from '@numen/ui'
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Ref, ShallowRef } from 'vue'

/** How long a strip taken somewhere by a key has to get there. */
const SLIDE_DURATION_MS = 600

/** How long a strip left alone has to have stopped before it is settled. */
const SNAP_DELAY_MS = 140

/** The three the strip stops at, in the order they stand. */
const PANEL_PLACES = ['before', 'here', 'after'] as const

/** Which of the three is in the window. */
export type PanelPlace = (typeof PANEL_PLACES)[number]

/** What the strip's movement is given: the elements it measures, and the answer. */
export interface CarouselStripDeps {
  readonly window: Readonly<ShallowRef<HTMLElement | null>>
  readonly middle: Readonly<ShallowRef<HTMLElement | null>>
  readonly currentPlace: Ref<PanelPlace>
}

function createCarouselStops(window_: Ref<HTMLElement | null>, middle: Ref<HTMLElement | null>) {
  const getStops = (): Record<PanelPlace, number> => {
    const place = window_.value
    const card = middle.value
    if (!place || !card) return { before: 0, here: 0, after: 0 }
    return { before: 0, here: card.offsetLeft, after: place.scrollWidth - place.clientWidth }
  }

  const findNearestPanel = (left: number): PanelPlace => {
    const all = getStops()
    let best: PanelPlace = 'before'
    for (const where of PANEL_PLACES) {
      if (Math.abs(all[where] - left) < Math.abs(all[best] - left)) best = where
    }
    return best
  }

  return { getStops, findNearestPanel }
}

/** The elements a press is meant for. A strip that captured the pointer would take their clicks. */
const CONTROLS = 'button, a[href], input, textarea, select, [contenteditable="true"]'

const isOnControl = (target: EventTarget | null): boolean =>
  target instanceof Element && target.closest(CONTROLS) !== null

function useCarouselDrag(deps: {
  readonly window_: Ref<HTMLElement | null>
  readonly currentPlace: Ref<PanelPlace>
  readonly findNearestPanel: (left: number) => PanelPlace
  readonly slideTo: (where: PanelPlace) => void
}) {
  const taking = ref(false)
  let from = 0
  let was = 0

  const onPointerDown = (press: PointerEvent) => {
    const place = deps.window_.value
    if (!place || press.button !== 0 || isOnControl(press.target)) return
    taking.value = true
    from = press.clientX
    was = place.scrollLeft
    place.setPointerCapture(press.pointerId)
  }

  const onPointerMove = (press: PointerEvent) => {
    const place = deps.window_.value
    if (!place || !taking.value) return
    place.scrollLeft = was - (press.clientX - from)
  }

  const letGo = async () => {
    if (!taking.value) return
    const place = deps.window_.value
    const where = place ? deps.findNearestPanel(place.scrollLeft) : deps.currentPlace.value
    taking.value = false
    await nextTick()
    if (where !== deps.currentPlace.value) deps.currentPlace.value = where
    await nextTick()
    deps.slideTo(deps.currentPlace.value)
  }

  return { taking, onPointerDown, onPointerMove, letGo }
}

function createCarouselMover(
  window_: Ref<HTMLElement | null>,
  getStops: () => Record<PanelPlace, number>,
) {
  let sending = 0
  let snapTimer = 0
  let aim: PanelPlace | null = null

  const slideTo = (where: PanelPlace) => {
    win.clearTimeout(snapTimer)
    const place = window_.value
    if (!place || Math.abs(place.scrollLeft - getStops()[where]) <= 1) return
    win.clearTimeout(sending)
    sending = win.setTimeout(() => {
      sending = 0
      aim = null
    }, SLIDE_DURATION_MS)
    aim = where
    place.scrollLeft = getStops()[where]
  }

  const placeAt = (where: PanelPlace) => {
    const place = window_.value
    if (!place) return
    win.clearTimeout(snapTimer)
    win.clearTimeout(sending)
    sending = 0
    aim = null
    place.style.scrollBehavior = 'auto'
    place.scrollLeft = getStops()[where]
    place.style.scrollBehavior = ''
  }

  const snapTo = (where: PanelPlace) => {
    win.clearTimeout(snapTimer)
    snapTimer = win.setTimeout(() => slideTo(aim ?? where), SNAP_DELAY_MS)
  }

  const clearAimOnArrival = (scrollLeft: number) => {
    if (aim && Math.abs(scrollLeft - getStops()[aim]) <= 1) {
      aim = null
      win.clearTimeout(sending)
      sending = 0
    }
  }

  const clear = () => {
    win.clearTimeout(sending)
    win.clearTimeout(snapTimer)
  }

  return { slideTo, placeAt, snapTo, clearAimOnArrival, clear, isAiming: () => aim !== null }
}

export const useCarouselStrip = (deps: CarouselStripDeps) => {
  const window_ = deps.window
  const middle = deps.middle
  const { currentPlace } = deps
  const { getStops, findNearestPanel } = createCarouselStops(window_, middle)
  const mover = createCarouselMover(window_, getStops)

  const { taking, onPointerDown, onPointerMove, letGo } = useCarouselDrag({
    window_,
    currentPlace,
    findNearestPanel,
    slideTo: mover.slideTo,
  })

  const onScroll = () => {
    const place = window_.value
    if (!place) return
    mover.clearAimOnArrival(place.scrollLeft)
    const where = findNearestPanel(place.scrollLeft)
    if (!mover.isAiming() && where !== currentPlace.value) currentPlace.value = where
    if (!taking.value) mover.snapTo(currentPlace.value)
  }

  watch(
    () => currentPlace.value,
    (where) => {
      if (!taking.value) mover.slideTo(where)
    },
  )

  const onResize = () => mover.placeAt(currentPlace.value)

  onMounted(() => {
    mover.placeAt(currentPlace.value)
    win.addEventListener('resize', onResize)
  })

  onBeforeUnmount(() => {
    win.removeEventListener('resize', onResize)
    mover.clear()
  })

  return { taking, onScroll, onPointerDown, onPointerMove, letGo }
}
