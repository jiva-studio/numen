/** Putting a pane beside a node, taking a tab off, and restoring the canonical tree. */
import { describe, expect, it, vi } from 'vitest'
import { commit, detach, mapPane, putBeside } from './splice'
import { pane, type Pane, type WorkspaceNode } from './node'
import { isBranch, isPane, nodeAt, paneById } from './tree'
import { createIdFactory, split, stack, workspaceOf } from '../fixtures/build'

describe('putBeside', () => {
  const two = split('root', [stack('a', 'one'), stack('b', 'two')])

  it('leaves the tree alone for the middle, which is not a side', () => {
    const landing = putBeside(two, 'a', 't', 'center', 'horizontal', createIdFactory())
    expect(landing).toStrictEqual({ root: two, axis: 'horizontal', focus: 'a' })
  })

  it('leaves the tree alone for a node that is not there', () => {
    const landing = putBeside(two, 'missing', 't', 'left', 'horizontal', createIdFactory())
    expect(landing).toStrictEqual({ root: two, axis: 'horizontal', focus: 'missing' })
  })

  it('divides a root that is one pane into two halves, turning the axis', () => {
    const landing = putBeside(stack('a', 'one'), 'a', 't', 'bottom', 'horizontal', createIdFactory())

    expect(landing.axis).toBe('vertical')
    expect(landing.focus).toBe('made-1')
    expect(isBranch(landing.root) && landing.root.sizes).toStrictEqual([0.5, 0.5])
    expect(isBranch(landing.root) && landing.root.children.map((child) => child.id)).toStrictEqual([
      'a',
      'made-1',
    ])
  })

  it('wraps a pane in two halves when the parent divides the other way', () => {
    const landing = putBeside(two, 'a', 't', 'top', 'horizontal', createIdFactory())
    const wrapper = nodeAt(landing.root, [0])

    expect(wrapper && isBranch(wrapper) && wrapper.sizes).toStrictEqual([0.5, 0.5])
    expect(wrapper && isBranch(wrapper) && wrapper.children.map((child) => child.id)).toStrictEqual(
      ['made-1', 'a'],
    )
    expect(landing.focus).toBe('made-1')
  })
})

describe('detach', () => {
  const stackOf = (active: string) =>
    split('root', [pane('main', ['a', 'b', 'c'], active), stack('other', 'x')])
  const activeOf = (root: WorkspaceNode) => paneById(root, 'main')?.active

  it('leaves the shown tab alone when another one goes', () => {
    expect(activeOf(detach(stackOf('a'), 'c'))).toBe('a')
  })

  it('shows the next tab along when the shown one goes', () => {
    expect(activeOf(detach(stackOf('b'), 'b'))).toBe('c')
  })

  it('shows the previous tab when the shown one was last', () => {
    expect(activeOf(detach(stackOf('c'), 'c'))).toBe('b')
  })

  it('shows nothing when the last tab goes', () => {
    const lone = split('root', [pane('main', ['a']), stack('other', 'x')])
    expect(paneById(detach(lone, 'a'), 'main')?.active).toBeNull()
  })

  it('leaves the tree alone for a tab that is not open', () => {
    const root = stackOf('a')
    expect(detach(root, 'missing')).toBe(root)
  })
})

describe('commit', () => {
  const three = workspaceOf(
    split('root', [stack('a', 'one'), stack('b', 'two'), stack('c', 'three')]),
    'horizontal',
    'c',
  )

  it('focuses the pane asked for', () => {
    expect(commit(three, 'b').focus).toBe('b')
  })

  it('keeps the focus it had when the pane asked for is gone', () => {
    expect(commit(three, 'missing').focus).toBe('c')
  })

  it('focuses the first pane when neither survives', () => {
    expect(commit({ ...three, focus: 'gone' }, 'missing').focus).toBe('a')
  })
})

describe('mapPane', () => {
  const root = split('root', [stack('a', 'one'), stack('b', 'two')])

  it('changes the pane with that id', () => {
    const after = mapPane(root, 'b', (held: Pane) => ({ ...held, tabs: [...held.tabs, 'new'] }))
    expect(paneById(after, 'b')?.tabs).toStrictEqual(['two', 'new'])
    expect(paneById(after, 'a')?.tabs).toStrictEqual(['one'])
  })

  it('leaves the tree alone for an id that is not there', () => {
    const change = vi.fn<(held: Pane) => Pane>((held) => held)
    expect(mapPane(root, 'missing', change)).toBe(root)
    expect(change).not.toHaveBeenCalled()
  })

  it('leaves the tree alone for a node that is not a pane', () => {
    const change = vi.fn<(held: Pane) => Pane>((held) => held)
    const after = mapPane(root, 'root', change)
    expect(isPane(after)).toBe(false)
    expect(after).toBe(root)
    expect(change).not.toHaveBeenCalled()
  })
})
