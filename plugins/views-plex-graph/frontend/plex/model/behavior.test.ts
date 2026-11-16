/**
 * The default behavior of a plex when no option is set.
 */
import { describe, expect, it } from 'vitest'
import { DEFAULT_BEHAVIOR, getInputStrategies } from './behavior'
import { EXPANSION_DELAY } from './sizing/expansion'
import { byHandle } from './interaction/reachStrategy'
import { byDoubleClick } from './interaction/destination'

describe('the behavior a plex has by default', () => {
  it('waits the usual while, offers the handle and asks for a node by its second click', () => {
    expect(DEFAULT_BEHAVIOR.expansionDelay).toBe(EXPANSION_DELAY)
    expect(DEFAULT_BEHAVIOR.reaching).toBe(byHandle)
    expect(DEFAULT_BEHAVIOR.showing).toBe(byDoubleClick)
  })

  it('opens a box on a clock that notifies the time', () => {
    expect(typeof DEFAULT_BEHAVIOR.clock.now()).toBe('number')
  })
})

describe('the input strategies a plex takes for a pointer', () => {
  it('offers the handle and the second click to a mouse', () => {
    expect(getInputStrategies(false)).toEqual({ reaching: byHandle, showing: byDoubleClick })
  })

  it('offers a rest and two taps to a touch, with no handle', () => {
    const { reaching, showing } = getInputStrategies(true)
    expect(reaching.hasHandle).toBe(false)
    expect(showing.isDoubleClick).toBe(false)
  })
})
