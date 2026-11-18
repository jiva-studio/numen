/** The names of the slots a plugin places its views in. */
import { describe, expect, it } from 'vitest'
import { SLOT } from '../src'

describe('the slots of the plex graph', () => {
  it('include one over a node and one under it', () => {
    expect(SLOT).toMatchObject({
      plexGraphNodeAbove: 'plexGraph.nodeAbove',
      plexGraphNodeBelow: 'plexGraph.nodeBelow',
    })
  })
})
