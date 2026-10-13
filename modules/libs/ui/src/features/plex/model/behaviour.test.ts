/**
 * The behaviour a plex has when nobody says otherwise.
 */
import { describe, expect, it } from 'vitest'
import { DEFAULT_BEHAVIOUR } from './behaviour'
import { DWELL } from './dwell'
import { byHandle } from './reaching'
import { byDoubleClick } from './showing'

describe('the behaviour a plex has by default', () => {
  it('waits the usual while, offers the handle and asks for a node by its second click', () => {
    expect(DEFAULT_BEHAVIOUR.dwell).toBe(DWELL)
    expect(DEFAULT_BEHAVIOUR.reaching).toBe(byHandle)
    expect(DEFAULT_BEHAVIOUR.showing).toBe(byDoubleClick)
  })

  it('opens a box on a clock that tells the time', () => {
    expect(typeof DEFAULT_BEHAVIOUR.clock.now()).toBe('number')
  })
})
