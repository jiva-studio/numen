/**
 * The default behavior of a plex when no option is set.
 */
import { describe, expect, it } from 'vitest'
import { DEFAULT_BEHAVIOR } from './behavior'
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
