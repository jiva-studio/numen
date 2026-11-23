/** Finding a node in a workspace, and putting one back. */
import { describe, expect, it } from 'vitest'
import { findNode, findPath, replaceAt } from './tree'
import { split, createStack } from '../fixtures/build'

const root = split('root', [
  createStack('a', 'one'),
  split('down', [createStack('b', 'two'), createStack('c', 'three')]),
])

describe('findNode', () => {
  it('finds the root by the empty path', () => {
    expect(findNode(root, [])).toBe(root)
  })

  it('finds a node by the indices down to it', () => {
    expect(findNode(root, [1, 1])?.id).toBe('c')
  })

  it('finds nothing below a pane', () => {
    expect(findNode(root, [0, 0])).toBeNull()
  })

  it('finds nothing at an index no child has', () => {
    expect(findNode(root, [5])).toBeNull()
    expect(findNode(root, [5, 0])).toBeNull()
  })
})

describe('findPath', () => {
  it('names the indices down to a node', () => {
    expect(findPath(root, 'root')).toStrictEqual([])
    expect(findPath(root, 'b')).toStrictEqual([1, 0])
  })

  it('is empty for a node that is not there', () => {
    expect(findPath(root, 'missing')).toBeNull()
    expect(findPath(createStack('a', 'one'), 'b')).toBeNull()
  })
})

describe('replaceAt', () => {
  it('puts the replacement where the path ends', () => {
    const after = replaceAt(root, [1, 0], createStack('x', 'new'))
    expect(findNode(after, [1, 0])?.id).toBe('x')
    expect(findNode(after, [0])?.id).toBe('a')
  })

  it('replaces the root for the empty path', () => {
    const replacement = createStack('x', 'new')
    expect(replaceAt(root, [], replacement)).toBe(replacement)
  })

  it('leaves the tree as it was when the path runs into a pane', () => {
    expect(replaceAt(createStack('a', 'one'), [0], createStack('x', 'new'))).toStrictEqual(
      createStack('a', 'one'),
    )
  })

  it('leaves the tree as it was when no child has that index', () => {
    expect(replaceAt(root, [5], createStack('x', 'new'))).toBe(root)
  })
})
