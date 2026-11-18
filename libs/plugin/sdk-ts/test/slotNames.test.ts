/** The names of the slots a plugin places its views in. */
import { describe, expect, it } from 'vitest'
import { SLOT } from '../src'

describe('the slots of the plex graph', () => {
  it('include one over a node, one under it and one over the whole of it', () => {
    expect(SLOT).toMatchObject({
      plexGraphNodeAbove: 'plexGraph.nodeAbove',
      plexGraphNodeBelow: 'plexGraph.nodeBelow',
      plexGraphNode: 'plexGraph.node',
    })
  })
})
