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
    const drawn = ['Root.md', 'physics/Entropy.md', 'Heat.md'].map(map.getNodeId)

    expect(new Set(drawn).size).toBe(3)
  })

  it('holds nothing at all where there is no path to hold it', () => {
    const map = createNodeIdMap()

    expect(map.getNodeId('')).toBe('')
  })
})

describe('a note whose file moved', () => {
  it('keeps the node ID it held', () => {
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

describe('a node ID handed back', () => {
  it('names the note holding it', () => {
    const map = createNodeIdMap()

    expect(map.getNodePath(map.getNodeId('Entropy.md'))).toBe('Entropy.md')
  })

  it('names nothing where no note holds it', () => {
    const map = createNodeIdMap()
    map.getNodeId('Entropy.md')

    expect(map.getNodePath('999')).toBe('')
    expect(map.getNodePath('Entropy.md')).toBe('')
    expect(map.getNodePath('')).toBe('')
  })
})

describe('the notes one picture drew', () => {
  it('are kept, and every other note is let go of', () => {
    const map = createNodeIdMap()
    const kept = map.getNodeId('Root.md')
    const gone = map.getNodeId('Heat.md')

    map.retainNodeIds([kept])

    expect(map.getNodePath(kept)).toBe('Root.md')
    expect(map.getNodePath(gone)).toBe('')
  })

  it('leave a note that comes back holding an ID of its own', () => {
    const map = createNodeIdMap()
    const root = map.getNodeId('Root.md')
    const before = map.getNodeId('Heat.md')
    map.retainNodeIds([root])

    const after = map.getNodeId('Heat.md')

    expect(after).not.toBe(before)
    expect(after).not.toBe(root)
    expect(map.getNodePath(root)).toBe('Root.md')
  })

  it('are all a person travelling from note to note leaves held', () => {
    const map = createNodeIdMap()
    const minted: string[] = []

    for (let step = 0; step < 50; step++) {
      const focus = map.getNodeId(`Note${step}.md`)
      const beside = map.getNodeId(`Beside${step}.md`)
      map.retainNodeIds([focus, beside])
      minted.push(focus, beside)
    }

    expect(minted.filter((id) => map.getNodePath(id))).toHaveLength(2)
  })
})
