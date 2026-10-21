/**
 * The strip of three panels: where it stops, what a hand and a wheel do to it,
 * and what a move asked for by a key does to both.
 *
 * jsdom lays nothing out, so the strip is given the widths it would have had: a
 * panel either side of a card the width of the window, with the card standing
 * at 220 and the strip running out at 440.
 */
import { mount } from '@vue/test-utils'
import type { VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, defineComponent, nextTick, ref, shallowRef } from 'vue'
import type { Ref } from 'vue'

import { useCarouselStrip } from './carousel'
import type { PanelPlace } from './carousel'

const mounted: VueWrapper[] = []

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  for (const one of mounted.splice(0)) one.unmount()
  vi.useRealTimers()
})

/** A press on the strip, by the button and at the place a test names. */
const press = (more: { button?: number; clientX?: number; pointerId?: number } = {}) => {
  const event = new MouseEvent('pointerdown', {
    button: more.button ?? 0,
    clientX: more.clientX ?? 0,
  })
  Object.defineProperty(event, 'pointerId', { value: more.pointerId ?? 1 })
  return event as PointerEvent
}

/**
 * The strip, its window and its card, with what the owner holds standing in a
 * ref. Every place the strip is set to is kept with the smoothing it was set
 * under.
 */
const createStrip = (
  at: PanelPlace = 'here',
  more: { hasCard?: boolean; owner?: Ref<PanelPlace> } = {},
) => {
  const window_ = document.createElement('div')
  Object.defineProperty(window_, 'scrollWidth', { value: 1040 })
  Object.defineProperty(window_, 'clientWidth', { value: 600 })
  const moves: { to: number; smoothing: string }[] = []
  let left = 0
  Object.defineProperty(window_, 'scrollLeft', {
    get: () => left,
    set: (to: number) => {
      left = to
      moves.push({ to, smoothing: window_.style.scrollBehavior })
    },
  })
  const capture = vi.fn()
  window_.setPointerCapture = capture

  const card = document.createElement('div')
  Object.defineProperty(card, 'offsetLeft', { value: 220 })

  const windowRef = shallowRef<HTMLElement | null>(window_)
  const middleRef = shallowRef<HTMLElement | null>(more.hasCard === false ? null : card)
  const displayed = more.owner ?? ref<PanelPlace>(at)
  const errors: unknown[] = []

  let target: ReturnType<typeof useCarouselStrip> | null = null
  const one = mount(
    defineComponent({
      setup() {
        target = useCarouselStrip({ window: windowRef, middle: middleRef, shown: displayed })
        return () => null
      },
    }),
    { global: { config: { errorHandler: (why) => errors.push(why) } } },
  )
  mounted.push(one)
  if (!target) throw new Error('the strip was not set up')
  const strip: ReturnType<typeof useCarouselStrip> = target

  /** The strip left where a wheel put it, and the strip told. */
  const scrollTo = (to: number) => {
    window_.scrollLeft = to
    strip.onScroll()
  }

  /** The owner moving the strip, as a key does. */
  const moveTo = async (where: PanelPlace) => {
    displayed.value = where
    await nextTick()
  }

  /** The strip taken off the screen, as the window closing takes it. */
  const takeOff = () => {
    mounted.splice(mounted.indexOf(one), 1)
    one.unmount()
  }

  return { ...strip, takeOff, window_, windowRef, shown: displayed, moves, capture, errors, scrollTo, moveTo }
}

describe('where the strip remains', () => {
  // A window opening on a card puts the card there: one seen sliding into place
  // looks like it is leaving.
  it('is put on its stop as it opens, with the smoothing off and back on after', () => {
    const { window_, moves } = createStrip('after')

    expect(window_.scrollLeft).toBe(440)
    expect(moves).toEqual([{ to: 440, smoothing: 'auto' }])
    expect(window_.style.scrollBehavior).toBe('')
  })

  it('remains every stop at the start while there is no card to read them off', () => {
    const { window_, shown, errors, scrollTo } = createStrip('after', { hasCard: false })
    expect(window_.scrollLeft).toBe(0)

    scrollTo(300)

    expect(shown.value).toBe('before')
    expect(errors).toEqual([])
  })

  it('takes the earlier of two stops it remains exactly between', () => {
    const { shown, scrollTo } = createStrip('here')
    scrollTo(330)
    expect(shown.value).toBe('here')
  })
})

describe('a move requested for', () => {
  it('drops the settling a wheel left pending', async () => {
    const { window_, scrollTo, moveTo } = createStrip('here')

    await moveTo('after')
    scrollTo(300)
    await moveTo('before')
    vi.advanceTimersByTime(200)

    expect(window_.scrollLeft).toBe(0)
  })

  // A move that moves nothing never arrives, and a pixel off is standing there.
  it('leaves a strip standing within a pixel of its stop where it is', () => {
    const { window_, scrollTo } = createStrip('here')

    scrollTo(221)
    vi.advanceTimersByTime(200)

    expect(window_.scrollLeft).toBe(221)
  })

  it('says nothing of where it passes through while the later of two moves is on its way', async () => {
    const { shown, scrollTo, moveTo } = createStrip('here')

    await moveTo('after')
    vi.advanceTimersByTime(500)
    await moveTo('before')
    vi.advanceTimersByTime(200)
    scrollTo(400)

    expect(shown.value).toBe('before')
  })

  it('hears a wheel as soon as the move has arrived', async () => {
    const { shown, scrollTo, moveTo } = createStrip('here')

    await moveTo('after')
    scrollTo(439)
    vi.advanceTimersByTime(100)
    scrollTo(200)

    expect(shown.value).toBe('here')
  })

  // A move that never arrives is over when its time is up.
  it('hears a wheel once the move has had its time, arrived or not', async () => {
    const { shown, scrollTo, moveTo } = createStrip('here')

    await moveTo('after')
    vi.advanceTimersByTime(700)
    scrollTo(200)

    expect(shown.value).toBe('here')
  })

  it('leaves the time of a move that arrived running out on no later move', async () => {
    const { shown, scrollTo, moveTo } = createStrip('here')

    await moveTo('after')
    vi.advanceTimersByTime(100)
    scrollTo(440)
    vi.advanceTimersByTime(100)
    await moveTo('before')
    vi.advanceTimersByTime(450)
    scrollTo(400)

    expect(shown.value).toBe('before')
  })

  it('does not take the strip anywhere while a hand is on it', async () => {
    const { window_, onPointerDown, moveTo } = createStrip('here')

    onPointerDown(press({ clientX: 500 }))
    await moveTo('after')

    expect(window_.scrollLeft).toBe(220)
  })
})

describe('a resize', () => {
  it('remains the strip on its stop, and no settling left pending takes it off', async () => {
    const { window_, onPointerDown, scrollTo, moveTo } = createStrip('here')

    await moveTo('after')
    scrollTo(300)
    onPointerDown(press({ clientX: 500 }))
    await moveTo('before')
    window.dispatchEvent(new Event('resize'))
    vi.advanceTimersByTime(200)

    expect(window_.scrollLeft).toBe(0)
  })

  it('cancels pending snap settling when sliding directly to a new place', async () => {
    const { window_, scrollTo, moveTo } = createStrip('here')
    const setTimeoutSpy = vi.spyOn(window, 'setTimeout')
    const clearTimeoutSpy = vi.spyOn(window, 'clearTimeout')

    scrollTo(200)
    const snapTimer = setTimeoutSpy.mock.results[0]?.value
    clearTimeoutSpy.mockClear()
    await moveTo('after')

    expect(clearTimeoutSpy).toHaveBeenCalledWith(snapTimer)
    expect(window_.scrollLeft).toBe(440)
    setTimeoutSpy.mockRestore()
    clearTimeoutSpy.mockRestore()
  })

  it('leaves no earlier move still running out its time', async () => {
    const { shown, scrollTo, moveTo } = createStrip('here')

    await moveTo('after')
    vi.advanceTimersByTime(100)
    window.dispatchEvent(new Event('resize'))
    vi.advanceTimersByTime(100)
    await moveTo('before')
    vi.advanceTimersByTime(450)
    scrollTo(400)

    expect(shown.value).toBe('before')
  })
})

describe('a wheel', () => {
  it('waits for the strip to stop before it settles it', () => {
    const { window_, scrollTo } = createStrip('here')

    scrollTo(260)
    vi.advanceTimersByTime(100)
    scrollTo(300)
    vi.advanceTimersByTime(60)
    expect(window_.scrollLeft).toBe(300)

    vi.advanceTimersByTime(100)
    expect(window_.scrollLeft).toBe(220)
  })
})

describe('a hand', () => {
  it('keeps hold of the pointer that took the strip', () => {
    const { capture, onPointerDown } = createStrip('here')
    onPointerDown(press({ pointerId: 7 }))
    expect(capture).toHaveBeenCalledWith(7)
  })

  it('takes the strip with the main button alone', () => {
    const { window_, capture, taking, onPointerDown, onPointerMove } = createStrip('here')

    onPointerDown(press({ button: 2, clientX: 500 }))
    onPointerMove(press({ clientX: 380 }))

    expect(taking.value).toBe(false)
    expect(capture).not.toHaveBeenCalled()
    expect(window_.scrollLeft).toBe(220)
  })

  it('is not settled while the hand is still on it', () => {
    const { window_, onPointerDown, onPointerMove, onScroll } = createStrip('here')

    onPointerDown(press({ clientX: 500 }))
    onPointerMove(press({ clientX: 380 }))
    onScroll()
    vi.advanceTimersByTime(200)

    expect(window_.scrollLeft).toBe(340)
  })

  it('settles nothing when it arrives off a strip it never went down on', async () => {
    const { window_, scrollTo, letGo } = createStrip('here')

    scrollTo(300)
    await letGo()

    expect(window_.scrollLeft).toBe(300)
  })
})

describe('a strip with no window', () => {
  it('does nothing when it is moved, scrolled, pressed or resized', async () => {
    const { windowRef, errors, onScroll, onPointerDown, onPointerMove, letGo, taking, moveTo } =
      createStrip('here')
    const uncaught: unknown[] = []
    const onError = (event: ErrorEvent) => uncaught.push(event.error)
    window.addEventListener('error', onError)

    windowRef.value = null
    await moveTo('after')
    onScroll()
    onPointerDown(press({ clientX: 500 }))
    onPointerMove(press({ clientX: 380 }))
    window.dispatchEvent(new Event('resize'))
    await letGo()
    window.removeEventListener('error', onError)

    expect(taking.value).toBe(false)
    expect(errors).toEqual([])
    expect(uncaught).toEqual([])
  })

  it('does nothing when a hand already on it moves after the window has gone', () => {
    const { windowRef, onPointerDown, onPointerMove } = createStrip('here')

    onPointerDown(press({ clientX: 500 }))
    windowRef.value = null

    expect(() => onPointerMove(press({ clientX: 380 }))).not.toThrow()
  })
})

describe('a strip taken off the screen', () => {
  it('is moved by neither a resize nor a settling left pending', () => {
    const { takeOff, window_, scrollTo } = createStrip('here')

    scrollTo(300)
    takeOff()
    window.dispatchEvent(new Event('resize'))
    vi.advanceTimersByTime(200)

    expect(window_.scrollLeft).toBe(300)
  })
})

/** The owner's hold on where the strip is, which hears of every write made to it. */
const createOwner = (at: PanelPlace) => {
  const target = ref<PanelPlace>(at)
  const writes: PanelPlace[] = []
  const owner = computed<PanelPlace>({
    get: () => target.value,
    set: (to) => {
      writes.push(to)
      target.value = to
    },
  })
  return { owner, writes }
}

describe('what the owner is told', () => {
  it('hears nothing of a strip that is scrolled and still relies on the one it showed', () => {
    const { owner, writes } = createOwner('here')
    const { scrollTo } = createStrip('here', { owner })

    scrollTo(230)

    expect(writes).toEqual([])
  })

  it('hears nothing of a hand that lets go on the one the strip showed', async () => {
    const { owner, writes } = createOwner('here')
    const { onPointerDown, letGo } = createStrip('here', { owner })

    onPointerDown(press({ clientX: 500 }))
    await letGo()

    expect(writes).toEqual([])
  })

  it('hears of the one a hand lets go nearest to', async () => {
    const { owner, writes } = createOwner('here')
    const { window_, onPointerDown, letGo } = createStrip('here', { owner })

    onPointerDown(press({ clientX: 500 }))
    window_.scrollLeft = 400
    await letGo()

    expect(writes).toEqual(['after'])
  })
})

describe('a move in flight when the strip is taken off the screen', () => {
  it('has its time cancelled', async () => {
    const activeSet = vi.spyOn(window, 'setTimeout')
    const { takeOff, moveTo } = createStrip('here')

    await moveTo('after')
    const timer = activeSet.mock.results.at(-1)?.value as number
    const cleared = vi.spyOn(window, 'clearTimeout')
    takeOff()

    expect(cleared).toHaveBeenCalledWith(timer)
    activeSet.mockRestore()
    cleared.mockRestore()
  })
})
