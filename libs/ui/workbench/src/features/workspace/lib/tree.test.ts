/** Finding a node in a workspace, and putting one back. */
import { describe, expect, it } from 'vitest'
import { findNode, pathTo, replaceAt } from './tree'
import { split, stack } from '../fixtures/build'

const root = split('root', [
  stack('a', 'one'),
  split('down', [stack('b', 'two'), stack('c', 'three')]),
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

describe('pathTo', () => {
  it('names the indices down to a node', () => {
    expect(pathTo(root, 'root')).toStrictEqual([])
    expect(pathTo(root, 'b')).toStrictEqual([1, 0])
  })

  it('is empty for a node that is not there', () => {
    expect(pathTo(root, 'missing')).toBeNull()
    expect(pathTo(stack('a', 'one'), 'b')).toBeNull()
  })
})

describe('replaceAt', () => {
  it('puts the replacement where the path ends', () => {
    const after = replaceAt(root, [1, 0], stack('x', 'new'))
    expect(findNode(after, [1, 0])?.id).toBe('x')
    expect(findNode(after, [0])?.id).toBe('a')
  })

  it('replaces the root for the empty path', () => {
    const replacement = stack('x', 'new')
    expect(replaceAt(root, [], replacement)).toBe(replacement)
  })

  it('leaves the tree as it was when the path runs into a pane', () => {
    expect(replaceAt(stack('a', 'one'), [0], stack('x', 'new'))).toStrictEqual(stack('a', 'one'))
  })

  it('leaves the tree as it was when no child has that index', () => {
    expect(replaceAt(root, [5], stack('x', 'new'))).toBe(root)
  })
})
