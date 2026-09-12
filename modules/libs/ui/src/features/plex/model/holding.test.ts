/** A finger left still on a node: when it counts as a rest, and when it does not. */
import { effectScope } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { HOLD, STRAY, useHold } from './holding'

const press = (over: Partial<PointerEvent> = {}) =>
  ({ pointerType: 'touch', clientX: 100, clientY: 100, ...over }) as PointerEvent

/** A hold watched inside a scope, so what it disposes of can be tested too. */
const createHolding = (ready = () => true) => {
  const reached = vi.fn()
  const scope = effectScope()
  const held = scope.run(() => useHold(ready, reached))!
  return { held, reached, scope }
}

afterEach(() => {
  vi.useRealTimers()
})

describe('a rest on a node', () => {
  it('reaches out with the press it began under, once the wait is over', () => {
    vi.useFakeTimers()
    const { held, reached } = createHolding()
    const event = press()

    held.onPointerDown(event)
    expect(reached).not.toHaveBeenCalled()

    vi.advanceTimersByTime(HOLD)
    expect(reached).toHaveBeenCalledWith(event)
  })

  it('is over as soon as it is reached out on', () => {
    vi.useFakeTimers()
    const { held } = createHolding()

    held.onPointerDown(press())
    vi.advanceTimersByTime(HOLD)
    expect(held.isResting()).toBe(false)
  })

  it('is given up on by a finger that strays', () => {
    vi.useFakeTimers()
    const { held, reached } = createHolding()

    held.onPointerDown(press())
    held.onPointerMove(press({ clientX: 100 + STRAY + 1 }))
    vi.advanceTimersByTime(HOLD)

    expect(reached).not.toHaveBeenCalled()
  })

  it('is kept by a finger that only trembles', () => {
    vi.useFakeTimers()
    const { held, reached } = createHolding()

    held.onPointerDown(press())
    held.onPointerMove(press({ clientX: 100 + STRAY - 1 }))
    vi.advanceTimersByTime(HOLD)

    expect(reached).toHaveBeenCalledOnce()
  })

  it('is given up on by a finger lifted before the wait is out', () => {
    vi.useFakeTimers()
    const { held, reached } = createHolding()

    held.onPointerDown(press())
    held.letGo()
    vi.advanceTimersByTime(HOLD)

    expect(reached).not.toHaveBeenCalled()
  })

  it('is not asked for by a mouse, which has the handle', () => {
    vi.useFakeTimers()
    const { held, reached } = createHolding()

    held.onPointerDown(press({ pointerType: 'mouse' }))
    vi.advanceTimersByTime(HOLD)

    expect(reached).not.toHaveBeenCalled()
  })

  it('is not asked for by a node that will not take one', () => {
    vi.useFakeTimers()
    const { held, reached } = createHolding(() => false)

    held.onPointerDown(press())
    vi.advanceTimersByTime(HOLD)

    expect(reached).not.toHaveBeenCalled()
  })

  it('goes with the scope it was made in', () => {
    vi.useFakeTimers()
    const { held, reached, scope } = createHolding()

    held.onPointerDown(press())
    scope.stop()
    vi.advanceTimersByTime(HOLD)

    expect(reached).not.toHaveBeenCalled()
  })
})
