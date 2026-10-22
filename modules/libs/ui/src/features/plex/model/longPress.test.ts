/** A finger left still on a node: when it counts as a rest, and when it does not. */
import { effectScope } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { LONG_PRESS_DELAY, LONG_PRESS_TOLERANCE, useLongPress } from './longPress'

const press = (over: Partial<PointerEvent> = {}) =>
  ({ pointerType: 'touch', clientX: 100, clientY: 100, ...over }) as PointerEvent

/** A hold watched inside a scope, so what it disposes of can be tested too. */
const createHolding = (ready = () => true) => {
  const reached = vi.fn()
  const scope = effectScope()
  const target = scope.run(() => useLongPress(ready, reached))!
  return { target: target, reached, scope }
}

afterEach(() => {
  vi.useRealTimers()
})

describe('a rest on a node', () => {
  it('reaches out with the press it began under, once the wait is over', () => {
    vi.useFakeTimers()
    const { target, reached } = createHolding()
    const event = press()

    target.onPointerDown(event)
    expect(reached).not.toHaveBeenCalled()

    vi.advanceTimersByTime(LONG_PRESS_DELAY)
    expect(reached).toHaveBeenCalledWith(event)
  })

  it('is over as soon as it is reached out on', () => {
    vi.useFakeTimers()
    const { target } = createHolding()

    target.onPointerDown(press())
    vi.advanceTimersByTime(LONG_PRESS_DELAY)
    expect(target.isResting()).toBe(false)
  })

  it('is given up on by a finger that strays', () => {
    vi.useFakeTimers()
    const { target, reached } = createHolding()

    target.onPointerDown(press())
    target.onPointerMove(press({ clientX: 100 + LONG_PRESS_TOLERANCE + 1 }))
    vi.advanceTimersByTime(LONG_PRESS_DELAY)

    expect(reached).not.toHaveBeenCalled()
  })

  it('is kept by a finger that only trembles', () => {
    vi.useFakeTimers()
    const { target, reached } = createHolding()

    target.onPointerDown(press())
    target.onPointerMove(press({ clientX: 100 + LONG_PRESS_TOLERANCE - 1 }))
    vi.advanceTimersByTime(LONG_PRESS_DELAY)

    expect(reached).toHaveBeenCalledOnce()
  })

  it('is given up on by a finger lifted before the wait is out', () => {
    vi.useFakeTimers()
    const { target, reached } = createHolding()

    target.onPointerDown(press())
    target.letGo()
    vi.advanceTimersByTime(LONG_PRESS_DELAY)

    expect(reached).not.toHaveBeenCalled()
  })

  it('is not requested for by a mouse, which has the handle', () => {
    vi.useFakeTimers()
    const { target, reached } = createHolding()

    target.onPointerDown(press({ pointerType: 'mouse' }))
    vi.advanceTimersByTime(LONG_PRESS_DELAY)

    expect(reached).not.toHaveBeenCalled()
  })

  it('is not requested for by a node that will not take one', () => {
    vi.useFakeTimers()
    const { target, reached } = createHolding(() => false)

    target.onPointerDown(press())
    vi.advanceTimersByTime(LONG_PRESS_DELAY)

    expect(reached).not.toHaveBeenCalled()
  })

  it('waits the default time before it reaches out, and not a moment less', () => {
    vi.useFakeTimers()
    const { target, reached } = createHolding()

    target.onPointerDown(press())
    vi.advanceTimersByTime(LONG_PRESS_DELAY - 1)
    expect(reached).not.toHaveBeenCalled()

    vi.advanceTimersByTime(1)
    expect(reached).toHaveBeenCalledOnce()
  })

  it('reports that it is resting from the press until it is over', () => {
    vi.useFakeTimers()
    const { target } = createHolding()
    expect(target.isResting()).toBe(false)

    target.onPointerDown(press())
    expect(target.isResting()).toBe(true)

    target.letGo()
    expect(target.isResting()).toBe(false)
  })

  it('follows the latest press when a second one arrives before the wait is out', () => {
    vi.useFakeTimers()
    const { target, reached } = createHolding()
    const second = press({ clientX: 140 })

    target.onPointerDown(press())
    vi.advanceTimersByTime(LONG_PRESS_DELAY / 2)
    target.onPointerDown(second)
    vi.advanceTimersByTime(LONG_PRESS_DELAY)

    expect(reached).toHaveBeenCalledOnce()
    expect(reached).toHaveBeenCalledWith(second)
  })

  it('is kept by a finger that strays exactly as far as is allowed', () => {
    vi.useFakeTimers()
    const { target, reached } = createHolding()

    target.onPointerDown(press())
    target.onPointerMove(press({ clientX: 100 + LONG_PRESS_TOLERANCE }))
    vi.advanceTimersByTime(LONG_PRESS_DELAY)

    expect(reached).toHaveBeenCalledOnce()
  })

  it('takes a move with no press before it as nothing', () => {
    const { target } = createHolding()

    expect(() => target.onPointerMove(press())).not.toThrow()
    expect(target.isResting()).toBe(false)
  })

  it('goes with the scope it was made in', () => {
    vi.useFakeTimers()
    const { target, reached, scope } = createHolding()

    target.onPointerDown(press())
    scope.stop()
    vi.advanceTimersByTime(LONG_PRESS_DELAY)

    expect(reached).not.toHaveBeenCalled()
  })
})
