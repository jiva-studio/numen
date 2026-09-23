/**
 * How the strip of three panels moves: where it stops, where a hand leaves it,
 * and where it is taken from there.
 *
 * The three are a strip the width of all of them, scrolled to the one in the
 * window, and the strip settles on the nearest of the three where a hand lets
 * go.
 */
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Ref, ShallowRef } from 'vue'

/** How long a strip taken somewhere by a key has to get there. */
const TAKES = 600

/** How long a strip left alone has to have stopped before it is settled. */
const SETTLES = 140

/** The three the strip stops at, in the order they stand. */
const PLACES = ['before', 'here', 'after'] as const

/** Which of the three is in the window. */
export type PanelPlace = (typeof PLACES)[number]

/** What the strip's movement is given: the elements it measures, and the answer. */
export interface CarouselStripDeps {
  /** The window the strip is seen through. */
  readonly window: Readonly<ShallowRef<HTMLElement | null>>
  /** The middle of the three, which the stops are read off. */
  readonly middle: Readonly<ShallowRef<HTMLElement | null>>
  /** Which of the three is in the window, moved by a hand as much as by the owner. */
  readonly shown: Ref<PanelPlace>
}

export const useCarouselStrip = (deps: CarouselStripDeps) => {
  const window_ = deps.window
  const middle = deps.middle
  const shown = deps.shown

  /** A hand is on the strip, so nothing takes it anywhere else. */
  const taking = ref(false)

  /**
   * The strip is being taken somewhere it was asked to go. It passes the others
   * on the way, and that is not a person asking for one of them.
   */
  let sending = 0

  /** The strip has stopped somewhere of its own, and is waiting to be settled. */
  let settling = 0

  /** Where a move in flight is going, and nothing while none is. */
  let aim: PanelPlace | null = null

  /**
   * Where the strip stands when each of the three is in the window. They are read
   * off the layout rather than worked out, so the space the three stand apart by
   * is in them already.
   */
  const getStops = (): Record<PanelPlace, number> => {
    const at = window_.value
    const card = middle.value
    if (!at || !card) return { before: 0, here: 0, after: 0 }
    return { before: 0, here: card.offsetLeft, after: at.scrollWidth - at.clientWidth }
  }

  /** Which of the three the strip is closest to standing on. */
  const findNearest = (left: number): PanelPlace => {
    const all = getStops()
    let best: PanelPlace = 'before'
    for (const where of PLACES) {
      if (Math.abs(all[where] - left) < Math.abs(all[best] - left)) best = where
    }
    return best
  }

  const slideTo = (where: PanelPlace) => {
    window.clearTimeout(settling)
    const at = window_.value
    if (!at) return
    // A strip already standing where it is being sent is left alone: a move
    // that moves nothing never arrives.
    if (Math.abs(at.scrollLeft - getStops()[where]) <= 1) return
    window.clearTimeout(sending)
    sending = window.setTimeout(() => {
      sending = 0
      aim = null
    }, TAKES)
    aim = where
    at.scrollLeft = getStops()[where]
  }

  /**
   * The strip put where it belongs rather than taken there. The window opens with
   * the card already in it, and a resize moves the stops under a strip that is
   * standing on one, so neither is something to be seen sliding.
   */
  const placeAt = (where: PanelPlace) => {
    const at = window_.value
    if (!at) return
    window.clearTimeout(settling)
    window.clearTimeout(sending)
    sending = 0
    aim = null
    at.style.scrollBehavior = 'auto'
    at.scrollLeft = getStops()[where]
    at.style.scrollBehavior = ''
  }

  const onResize = () => placeAt(shown.value)

  watch(
    () => shown.value,
    (where) => {
      if (!taking.value) slideTo(where)
    },
  )

  /**
   * Where a hand has taken the strip. Past the halfway mark between two of the
   * stops it has asked for the nearer one, and the window is told as it crosses.
   */
  const onScroll = () => {
    const at = window_.value
    if (!at) return
    // A move in flight is over when it arrives, whatever time it took.
    if (aim && Math.abs(at.scrollLeft - getStops()[aim]) <= 1) {
      aim = null
      window.clearTimeout(sending)
      sending = 0
    }
    const where = findNearest(at.scrollLeft)
    if (!aim && where !== shown.value) shown.value = where

    // A wheel or a trackpad leaves the strip wherever it ran out, and it is taken
    // the rest of the way once it has stopped. A hand still on it is not done.
    //
    // It is armed while a move is in flight too, and aims where that move was
    // going, so a wheel turned during one still lands on a card.
    //
    // Where it settles is the window's answer and not the strip's: a panel the
    // window refused to open is a panel the strip must not be left standing on.
    if (taking.value) return
    window.clearTimeout(settling)
    const going = aim
    settling = window.setTimeout(() => slideTo(going ?? shown.value), SETTLES)
  }

  /** Where the hand went down, and where the strip was under it. */
  let from = 0
  let was = 0

  const onPointerDown = (press: PointerEvent) => {
    const at = window_.value
    if (!at || press.button !== 0) return
    taking.value = true
    from = press.clientX
    was = at.scrollLeft
    at.setPointerCapture(press.pointerId)
  }

  const onPointerMove = (press: PointerEvent) => {
    const at = window_.value
    if (!at || !taking.value) return
    at.scrollLeft = was - (press.clientX - from)
  }

  const letGo = async () => {
    if (!taking.value) return
    const at = window_.value
    const where = at ? findNearest(at.scrollLeft) : shown.value
    taking.value = false
    // A hand moves the strip with the smoothing off, and it is taken the rest of
    // the way with it on, so the letting go is waited for.
    await nextTick()
    if (where !== shown.value) shown.value = where
    // Where the hand asked for and where the window went are two things: a panel
    // the window refused to open is one the strip goes back off.
    await nextTick()
    slideTo(shown.value)
  }

  onMounted(() => {
    placeAt(shown.value)
    window.addEventListener('resize', onResize)
  })

  onBeforeUnmount(() => {
    window.removeEventListener('resize', onResize)
    window.clearTimeout(sending)
    window.clearTimeout(settling)
  })

  return { taking, onScroll, onPointerDown, onPointerMove, letGo }
}
