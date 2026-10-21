/** Two taps counted here: when they make a second tap, and what they leave behind. */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope, type EffectScope } from 'vue'
import { APART, byDoubleTap, TAP, type ShowSite } from './destination'

const tap = (x: number, y: number, pointerType = 'touch') =>
  new PointerEvent('pointerup', { clientX: x, clientY: y, pointerType })

/** A node counting taps, and what it was asked to show. */
function createTapper(ready: () => boolean = () => true, within?: number) {
  const show = vi.fn<ShowSite['show']>()
  const scope: EffectScope = effectScope()
  const listeners = scope.run(() => byDoubleTap(within).listeners({ ready, show }))!
  return { show, scope, tap: (event: PointerEvent) => listeners['pointerup']!(event) }
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('two taps', () => {
  it('show the node, here, when the second lands close to the first', () => {
    const { tap: press, show } = createTapper()
    press(tap(100, 100))
    press(tap(110, 100))
    expect(show).toHaveBeenCalledTimes(1)
    expect(show).toHaveBeenCalledWith(false)
  })

  it('count a second that lands exactly as far as the allowance', () => {
    const { tap: press, show } = createTapper()
    press(tap(100, 100))
    press(tap(100 + APART, 100))
    expect(show).toHaveBeenCalledTimes(1)
  })

  it('wait as long as they are given', () => {
    const { tap: press, show } = createTapper(() => true, 100)
    press(tap(100, 100))
    vi.advanceTimersByTime(101)
    press(tap(100, 100))
    expect(show).not.toHaveBeenCalled()
  })

  it('wait the allowance of time by default', () => {
    const { tap: press, show } = createTapper()
    press(tap(100, 100))
    vi.advanceTimersByTime(TAP - 1)
    press(tap(100, 100))
    expect(show).toHaveBeenCalledTimes(1)
  })

  it('are not made by a mouse, which has the second click', () => {
    const { tap: press, show } = createTapper()
    press(tap(100, 100, 'mouse'))
    press(tap(100, 100, 'mouse'))
    expect(show).not.toHaveBeenCalled()
  })

  it('are not made on a node that is not ready', () => {
    const { tap: press, show } = createTapper(() => false)
    press(tap(100, 100))
    press(tap(100, 100))
    expect(show).not.toHaveBeenCalled()
  })

  it('are not counted from a mouse towards a finger', () => {
    const { tap: press, show } = createTapper()
    press(tap(100, 100, 'mouse'))
    press(tap(100, 100))
    expect(show).not.toHaveBeenCalled()
  })

  it('start over after they have shown the node', () => {
    const { tap: press, show } = createTapper()
    press(tap(100, 100))
    press(tap(100, 100))
    press(tap(100, 100))
    expect(show).toHaveBeenCalledTimes(1)
  })

  it('are not cut short by the wait of a tap they replaced', () => {
    const { tap: press, show } = createTapper()
    press(tap(100, 100))
    vi.advanceTimersByTime(200)
    press(tap(500, 500))
    vi.advanceTimersByTime(101)
    press(tap(500, 500))
    expect(show).toHaveBeenCalledTimes(1)
  })

  it('are not cut short by the wait of a pair that already showed the node', () => {
    const { tap: press, show } = createTapper()
    press(tap(100, 100))
    vi.advanceTimersByTime(10)
    press(tap(100, 100))
    vi.advanceTimersByTime(240)
    press(tap(100, 100))
    vi.advanceTimersByTime(100)
    press(tap(100, 100))
    expect(show).toHaveBeenCalledTimes(2)
  })

  it('leave no wait running once the node is gone', () => {
    const { tap: press, scope } = createTapper()
    press(tap(100, 100))
    expect(vi.getTimerCount()).toBe(1)
    scope.stop()
    expect(vi.getTimerCount()).toBe(0)
  })
})
