/** Putting a pane beside a node, taking a tab off, and restoring the canonical tree. */
import { describe, expect, it, vi } from 'vitest'
import { commit, detach, mapPane, putAdjacent } from './splice'
import { pane, type Pane, type WorkspaceNode } from './node'
import { isBranch, isPane, findNode, paneById } from './tree'
import { createIdFactory, split, stack, createWorkspace } from '../fixtures/build'

describe('putAdjacent', () => {
  const two = split('root', [stack('a', 'one'), stack('b', 'two')])

  it('leaves the tree alone for the middle, which is not a side', () => {
    const divided = putAdjacent(
      two,
      { onto: 'a', tab: 't', side: 'center', axis: 'horizontal' },
      createIdFactory(),
    )
    expect(divided).toStrictEqual({ root: two, axis: 'horizontal', focus: 'a' })
  })

  it('leaves the tree alone for a node that is not there', () => {
    const divided = putAdjacent(
      two,
      { onto: 'missing', tab: 't', side: 'left', axis: 'horizontal' },
      createIdFactory(),
    )
    expect(divided).toStrictEqual({ root: two, axis: 'horizontal', focus: 'missing' })
  })

  it('divides a root that is one pane into two halves, turning the axis', () => {
    const divided = putAdjacent(
      stack('a', 'one'),
      { onto: 'a', tab: 't', side: 'bottom', axis: 'horizontal' },
      createIdFactory(),
    )

    expect(divided.axis).toBe('vertical')
    expect(divided.focus).toBe('made-1')
    expect(isBranch(divided.root) && divided.root.sizes).toStrictEqual([0.5, 0.5])
    expect(isBranch(divided.root) && divided.root.children.map((child) => child.id)).toStrictEqual([
      'a',
      'made-1',
    ])
  })

  it('wraps a pane in two halves when the parent divides the other way', () => {
    const divided = putAdjacent(
      two,
      { onto: 'a', tab: 't', side: 'top', axis: 'horizontal' },
      createIdFactory(),
    )
    const wrapper = findNode(divided.root, [0])

    expect(wrapper && isBranch(wrapper) && wrapper.sizes).toStrictEqual([0.5, 0.5])
    expect(wrapper && isBranch(wrapper) && wrapper.children.map((child) => child.id)).toStrictEqual(
      ['made-1', 'a'],
    )
    expect(divided.focus).toBe('made-1')
  })
})

describe('detach', () => {
  const createStack = (active: string) =>
    split('root', [pane('main', ['a', 'b', 'c'], active), stack('other', 'x')])
  const getActive = (root: WorkspaceNode) => paneById(root, 'main')?.active

  it('leaves the shown tab alone when another one goes', () => {
    expect(getActive(detach(createStack('a'), 'c'))).toBe('a')
  })

  it('shows the next tab along when the shown one goes', () => {
    expect(getActive(detach(createStack('b'), 'b'))).toBe('c')
  })

  it('shows the previous tab when the shown one was last', () => {
    expect(getActive(detach(createStack('c'), 'c'))).toBe('b')
  })

  it('shows nothing when the last tab goes', () => {
    const lone = split('root', [pane('main', ['a']), stack('other', 'x')])
    expect(paneById(detach(lone, 'a'), 'main')?.active).toBeNull()
  })

  it('leaves the tree alone for a tab that is not open', () => {
    const root = createStack('a')
    expect(detach(root, 'missing')).toBe(root)
  })
})

describe('commit', () => {
  const three = createWorkspace(
    split('root', [stack('a', 'one'), stack('b', 'two'), stack('c', 'three')]),
    'horizontal',
    'c',
  )

  it('focuses the pane requested for', () => {
    expect(commit(three, 'b').focus).toBe('b')
  })

  it('keeps the focus it had when the pane requested for is gone', () => {
    expect(commit(three, 'missing').focus).toBe('c')
  })

  it('focuses the first pane when neither survives', () => {
    expect(commit({ ...three, focus: 'gone' }, 'missing').focus).toBe('a')
  })
})

describe('mapPane', () => {
  const root = split('root', [stack('a', 'one'), stack('b', 'two')])

  it('changes the pane with that id', () => {
    const after = mapPane(root, 'b', (tab: Pane) => ({ ...tab, tabs: [...tab.tabs, 'new'] }))
    expect(paneById(after, 'b')?.tabs).toStrictEqual(['two', 'new'])
    expect(paneById(after, 'a')?.tabs).toStrictEqual(['one'])
  })

  it('leaves the tree alone for an id that is not there', () => {
    const change = vi.fn<(target: Pane) => Pane>((target) => target)
    expect(mapPane(root, 'missing', change)).toBe(root)
    expect(change).not.toHaveBeenCalled()
  })

  it('leaves the tree alone for a node that is not a pane', () => {
    const change = vi.fn<(target: Pane) => Pane>((target) => target)
    const after = mapPane(root, 'root', change)
    expect(isPane(after)).toBe(false)
    expect(after).toBe(root)
    expect(change).not.toHaveBeenCalled()
  })
})
