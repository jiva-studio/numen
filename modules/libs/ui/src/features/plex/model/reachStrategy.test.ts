/** Reaching out by resting a finger: how long the rest lasts. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import { byLongPress, type ReachSite } from './reaching'
import { LONG_PRESS_DELAY } from './longPress'

const touch = () => ({ pointerType: 'touch', clientX: 10, clientY: 10 }) as PointerEvent

/** A node resting a finger under the given wait, and what it reached out with. */
function createRest(after?: number) {
  const reach = vi.fn<ReachSite['reach']>()
  const scope = effectScope()
  const listeners = scope.run(() => byLongPress(after).listeners({ ready: () => true, reach }))!
  listeners['pointerdown']!(touch())
  return { reach, scope }
}

afterEach(() => {
  vi.useRealTimers()
})

describe('a finger left still on a node', () => {
  it('reaches out after the wait it was given', () => {
    vi.useFakeTimers()
    const { reach, scope } = createRest(100)
    vi.advanceTimersByTime(99)
    expect(reach).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)
    expect(reach).toHaveBeenCalledTimes(1)
    scope.stop()
  })

  it('reaches out after the usual wait where none was given', () => {
    vi.useFakeTimers()
    const { reach, scope } = createRest()
    vi.advanceTimersByTime(LONG_PRESS_DELAY - 1)
    expect(reach).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)
    expect(reach).toHaveBeenCalledTimes(1)
    scope.stop()
  })
})
