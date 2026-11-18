/**
 * Tests for note file paths to stable node IDs mapping.
 */
import { describe, expect, it } from 'vitest'
import { createNodeIdMap } from './nodeIdMap'

describe('a note being drawn', () => {
  it('holds one node ID for as long as the picture draws it', () => {
    const map = createNodeIdMap()

    expect(map.getNodeId('Entropy.md')).toBe(map.getNodeId('Entropy.md'))
  })

  it('holds an ID no other note in the picture holds', () => {
    const map = createNodeIdMap()
    const nodeIds = ['Root.md', 'physics/Entropy.md', 'Heat.md'].map(map.getNodeId)

    expect(new Set(nodeIds).size).toBe(3)
  })

  it('holds IDs counting up from 1', () => {
    const map = createNodeIdMap()

    expect(['A.md', 'B.md', 'C.md'].map(map.getNodeId)).toEqual(['1', '2', '3'])
  })

  it('holds nothing at all where there is no path to hold it', () => {
    const map = createNodeIdMap()

    expect(map.getNodeId('')).toBe('')
  })
})

describe('a note whose file moved', () => {
  it('keeps the node ID it retained', () => {
    const map = createNodeIdMap()
    const id = map.getNodeId('Entropy.md')

    map.updateRenamedNodes([{ from: 'Entropy.md', to: 'physics/Entropy.md' }])

    expect(map.getNodeId('physics/Entropy.md')).toBe(id)
    expect(map.getNodePath(id)).toBe('physics/Entropy.md')
  })

  it('keeps it through a second move', () => {
    const map = createNodeIdMap()
    const id = map.getNodeId('Entropy.md')

    map.updateRenamedNodes([{ from: 'Entropy.md', to: 'physics/Entropy.md' }])
    map.updateRenamedNodes([{ from: 'physics/Entropy.md', to: 'heat/Entropy.md' }])

    expect(map.getNodePath(id)).toBe('heat/Entropy.md')
  })

  it('is left where it stood by a move of some other note', () => {
    const map = createNodeIdMap()
    const id = map.getNodeId('Entropy.md')

    map.updateRenamedNodes([{ from: 'Heat.md', to: 'physics/Heat.md' }])

    expect(map.getNodePath(id)).toBe('Entropy.md')
  })

  it('moves with every other note of a folder that moved at once', () => {
    const map = createNodeIdMap()
    const one = map.getNodeId('physics/Entropy.md')
    const two = map.getNodeId('physics/Heat.md')

    map.updateRenamedNodes([
      { from: 'physics/Entropy.md', to: 'science/physics/Entropy.md' },
      { from: 'physics/Heat.md', to: 'science/physics/Heat.md' },
    ])

    expect(map.getNodePath(one)).toBe('science/physics/Entropy.md')
    expect(map.getNodePath(two)).toBe('science/physics/Heat.md')
  })

  it('trades paths with a note that took its own', () => {
    const map = createNodeIdMap()
    const one = map.getNodeId('One.md')
    const two = map.getNodeId('Two.md')

    map.updateRenamedNodes([
      { from: 'One.md', to: 'Two.md' },
      { from: 'Two.md', to: 'One.md' },
    ])

    expect(map.getNodePath(one)).toBe('Two.md')
    expect(map.getNodePath(two)).toBe('One.md')
  })
})

describe('a node ID returned', () => {
  it('names the note holding it', () => {
    const map = createNodeIdMap()

    expect(map.getNodePath(map.getNodeId('Entropy.md'))).toBe('Entropy.md')
  })

  it('is empty where no note holds it', () => {
    const map = createNodeIdMap()
    map.getNodeId('Entropy.md')

    expect(map.getNodePath('999')).toBe('')
    expect(map.getNodePath('Entropy.md')).toBe('')
    expect(map.getNodePath('')).toBe('')
  })
})

describe('a note that left the picture', () => {
  it('keeps its node ID and its path, so a node still on the screen shows the same note', () => {
    const map = createNodeIdMap()
    const left = map.getNodeId('Heat.md')

    for (let step = 0; step < 5; step++) map.getNodeId(`Note${step}.md`)

    expect(map.getNodePath(left)).toBe('Heat.md')
    expect(map.getNodeId('Heat.md')).toBe(left)
  })
})
