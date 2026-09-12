/** What each gesture leaves behind. */
import { describe, expect, it } from 'vitest'
import {
  activateTab,
  closeTab,
  dropOnEdge,
  dropTab,
  moveTabWithin,
  openTab,
  openTabBeside,
  resizeBranch,
} from './edit'
import { arrangeWorkspace } from './arrange'
import type { Side, Workspace } from './node'
import { isBranch, isPane, paneWithTab, panesOf } from './tree'
import {
  deep,
  empty,
  createIdFactory,
  oneStack,
  sideBySide,
  split,
  stack,
  withSpare,
  workspaceOf,
} from '../fixtures/build'
import { broken } from '../fixtures/invariants'

const SCREEN = { x: 0, y: 0, width: 1000, height: 600 }

const ids = (workspace: Workspace) =>
  isBranch(workspace.root) ? workspace.root.children.map((child) => child.id) : [workspace.root.id]

const tabsOf = (workspace: Workspace, pane: string) =>
  panesOf(workspace.root).find((each) => each.id === pane)?.tabs

describe('opening and showing', () => {
  it('puts a new tab in the focused pane', () => {
    const after = openTab(sideBySide(), 'notes', 'aside')
    expect(tabsOf(after, 'aside')).toStrictEqual(['chat', 'notes'])
    expect(after.focus).toBe('aside')
    expect(broken(after)).toStrictEqual([])
  })

  it('shows a tab that is already open where it is', () => {
    const after = openTab(sideBySide(), 'chat', 'main')
    expect(tabsOf(after, 'main')).toStrictEqual(['plex'])
    expect(after.focus).toBe('aside')
  })

  it('moves the focus to whatever is shown', () => {
    const after = activateTab(oneStack(), 'chat')
    expect(panesOf(after.root)[0]?.active).toBe('chat')
    expect(after.focus).toBe('main')
  })
})

describe('opening a tab in a pane of its own', () => {
  it('opens one that was nowhere in the tree', () => {
    const after = openTabBeside(sideBySide(), 'notes', 'right', createIdFactory())

    expect(ids(after)).toStrictEqual(['main', 'made-1', 'aside'])
    expect(tabsOf(after, 'made-1')).toStrictEqual(['notes'])
    expect(after.focus).toBe('made-1')
    expect(broken(after)).toStrictEqual([])
  })

  it('takes one that is already open away from where it was', () => {
    const after = openTabBeside(sideBySide(), 'chat', 'right', createIdFactory())

    expect(paneWithTab(after.root, 'chat')?.id).toBe('made-1')
    expect(panesOf(after.root).map((each) => each.id)).toStrictEqual(['main', 'made-1'])
    expect(after.focus).toBe('made-1')
    expect(broken(after)).toStrictEqual([])
  })

  it('leaves one that is alone in the pane it would open beside where it is', () => {
    const before = workspaceOf(stack('main', 'plex'))
    const ids = createIdFactory()

    const after = openTabBeside(before, 'plex', 'left', ids)

    expect(after).toStrictEqual(before)
    // Nothing was made, so nothing was named.
    expect(ids()).toBe('made-1')
  })

  it.each([
    ['left' as Side, 'horizontal', ['made-1', 'main']],
    ['right' as Side, 'horizontal', ['main', 'made-1']],
    ['top' as Side, 'vertical', ['made-1', 'main']],
    ['bottom' as Side, 'vertical', ['main', 'made-1']],
  ])('puts it on the %s', (side, axis, order) => {
    const after = openTabBeside(oneStack(), 'notes', side, createIdFactory())

    expect(ids(after)).toStrictEqual(order)
    expect(after.axis).toBe(axis)
    expect(broken(after)).toStrictEqual([])
  })

  it('joins the parent when the parent already divides that way', () => {
    const after = openTabBeside(withSpare(), 'more', 'left', createIdFactory(), 'aside')

    expect(ids(after)).toStrictEqual(['main', 'made-1', 'aside'])
    expect(broken(after)).toStrictEqual([])
  })

  it('wraps the pane when the parent divides the other way', () => {
    const after = openTabBeside(withSpare(), 'more', 'bottom', createIdFactory(), 'main')

    expect(ids(after)).toStrictEqual(['made-2', 'aside'])
    const wrapper = isBranch(after.root) ? after.root.children[0] : undefined
    expect(wrapper && isBranch(wrapper) && wrapper.children.map((child) => child.id)).toStrictEqual([
      'main',
      'made-1',
    ])
    expect(broken(after)).toStrictEqual([])
  })

  it('takes half of what it opened beside', () => {
    const after = openTabBeside(sideBySide(), 'notes', 'right', createIdFactory())
    const boxes = arrangeWorkspace(after, SCREEN)

    expect(boxes.get('main')?.width).toBeCloseTo(360)
    expect(boxes.get('made-1')?.width).toBeCloseTo(360)
    expect(boxes.get('aside')?.width).toBeCloseTo(280)
  })

  it('leaves the others where they were', () => {
    const before = arrangeWorkspace(withSpare(), SCREEN)
    const after = arrangeWorkspace(
      openTabBeside(withSpare(), 'more', 'right', createIdFactory(), 'main'),
      SCREEN,
    )
    expect(after.get('aside')).toStrictEqual(before.get('aside'))
  })

  it('opens beside the pane a tab opens into when told no other', () => {
    const focused = workspaceOf(sideBySide().root, 'horizontal', 'aside')
    const after = openTabBeside(focused, 'notes', 'right', createIdFactory())

    expect(ids(after)).toStrictEqual(['main', 'aside', 'made-1'])
    expect(broken(after)).toStrictEqual([])
  })

  it('divides a workspace holding nothing, and clears the empty pane away', () => {
    const after = openTabBeside(empty(), 'notes', 'bottom', createIdFactory())

    expect(isPane(after.root) && after.root.id).toBe('made-1')
    expect(tabsOf(after, 'made-1')).toStrictEqual(['notes'])
    expect(broken(after)).toStrictEqual([])
  })

  it('does nothing for the middle, which divides nothing', () => {
    const before = sideBySide()
    expect(openTabBeside(before, 'notes', 'center', createIdFactory())).toStrictEqual(before)
  })

  it('leaves the tree canonical when the pane it opened beside empties', () => {
    const after = openTabBeside(oneStack(), 'plex', 'left', createIdFactory(), 'main')

    expect(panesOf(after.root)).toHaveLength(2)
    expect(tabsOf(after, 'made-1')).toStrictEqual(['plex'])
    expect(broken(after)).toStrictEqual([])
  })
})

describe('a tab let go beside a pane', () => {
  it('joins the parent when the parent already divides that way', () => {
    const after = dropTab(sideBySide(), { tab: 'chat', onto: 'main', side: 'left' }, createIdFactory())

    expect(ids(after)).toStrictEqual(['made-1', 'main'])
    expect(after.axis).toBe('horizontal')
    expect(after.focus).toBe('made-1')
    expect(broken(after)).toStrictEqual([])
  })

  it('wraps the pane when the parent divides the other way', () => {
    const after = dropTab(withSpare(), { tab: 'notes', onto: 'main', side: 'bottom' }, createIdFactory())

    expect(ids(after)).toStrictEqual(['made-2', 'aside'])
    const wrapper = isBranch(after.root) ? after.root.children[0] : undefined
    expect(wrapper && isBranch(wrapper) && wrapper.children.map((child) => child.id)).toStrictEqual([
      'main',
      'made-1',
    ])
    expect(broken(after)).toStrictEqual([])
  })

  it('takes half of what it landed beside', () => {
    const after = dropTab(oneStack(), { tab: 'chat', onto: 'main', side: 'right' }, createIdFactory())
    const boxes = arrangeWorkspace(after, SCREEN)

    expect(boxes.get('main')?.width).toBeCloseTo(500)
    expect(boxes.get('made-1')?.width).toBeCloseTo(500)
  })

  it('turns the axis when the whole workspace is one pane', () => {
    const after = dropTab(oneStack(), { tab: 'chat', onto: 'main', side: 'top' }, createIdFactory())

    expect(after.axis).toBe('vertical')
    expect(ids(after)).toStrictEqual(['made-1', 'main'])
  })

  it('leaves the others where they were', () => {
    const before = arrangeWorkspace(withSpare(), SCREEN)
    const after = arrangeWorkspace(
      dropTab(withSpare(), { tab: 'notes', onto: 'main', side: 'right' }, createIdFactory()),
      SCREEN,
    )
    expect(after.get('aside')).toStrictEqual(before.get('aside'))
  })
})

describe('a tab let go in the middle of a pane', () => {
  it('joins the stack and is shown', () => {
    const after = dropTab(sideBySide(), { tab: 'chat', onto: 'main', side: 'center' }, createIdFactory())

    expect(tabsOf(after, 'main')).toStrictEqual(['plex', 'chat'])
    expect(panesOf(after.root)).toHaveLength(1)
    expect(after.root.id).toBe('main')
    expect(broken(after)).toStrictEqual([])
  })

  it('clears away the pane it came from', () => {
    const after = dropTab(sideBySide(), { tab: 'plex', onto: 'aside', side: 'center' }, createIdFactory())
    expect(panesOf(after.root).map((each) => each.id)).toStrictEqual(['aside'])
  })
})

describe('a tab let go where it started', () => {
  it('is only shown when it is the only one there', () => {
    const before = sideBySide()
    const after = dropTab(before, { tab: 'chat', onto: 'aside', side: 'left' }, createIdFactory())

    expect(after.root).toStrictEqual(before.root)
    expect(after.focus).toBe('aside')
  })

  it('splits its own pane when there are others with it', () => {
    const after = dropTab(oneStack(), { tab: 'chat', onto: 'main', side: 'right' }, createIdFactory())

    expect(tabsOf(after, 'main')).toStrictEqual(['plex'])
    expect(panesOf(after.root)).toHaveLength(2)
    expect(broken(after)).toStrictEqual([])
  })
})

describe('a tab let go on the outer edge', () => {
  it('divides the whole workspace', () => {
    const after = dropOnEdge(withSpare(), 'notes', 'bottom', createIdFactory())

    expect(after.axis).toBe('vertical')
    expect(ids(after)).toStrictEqual(['root', 'made-1'])
    expect(broken(after)).toStrictEqual([])
  })

  it('does nothing for the middle, which is not an edge', () => {
    const before = sideBySide()
    expect(dropOnEdge(before, 'chat', 'center', createIdFactory())).toStrictEqual(before)
  })
})

describe('closing', () => {
  it('shows the next tab along', () => {
    const after = closeTab(oneStack(), 'plex')
    expect(panesOf(after.root)[0]?.active).toBe('chat')
    expect(broken(after)).toStrictEqual([])
  })

  it('clears the pane away and leaves the rest where they were', () => {
    const before = deep()
    const boxes = arrangeWorkspace(before, SCREEN)
    const after = closeTab(before, 'four')

    expect(paneWithTab(after.root, 'four')).toBeNull()
    expect(arrangeWorkspace(after, SCREEN).get('e')).toStrictEqual(boxes.get('again'))
    expect(arrangeWorkspace(after, SCREEN).get('a')).toStrictEqual(boxes.get('a'))
    expect(broken(after)).toStrictEqual([])
  })

  it('leaves an empty workspace when the last tab goes', () => {
    const after = closeTab(closeTab(oneStack(), 'plex'), 'chat')

    expect(isPane(after.root) && after.root.tabs).toStrictEqual([])
    expect(isPane(after.root) && after.root.active).toBeNull()
    expect(broken(after)).toStrictEqual([])
  })

  it('ignores a tab that is not open', () => {
    const before = sideBySide()
    expect(closeTab(before, 'nothing')).toStrictEqual(before)
  })
})

describe('moving a tab along its own strip', () => {
  const three = workspaceOf(stack('main', 'one', 'two', 'three'))

  it.each([
    [0, ['three', 'one', 'two']],
    [1, ['one', 'three', 'two']],
    [3, ['one', 'two', 'three']],
  ])('to slot %i', (slot, expected) => {
    expect(tabsOf(moveTabWithin(three, 'three', slot), 'main')).toStrictEqual(expected)
  })

  it('counts the slot as it stands before the tab is lifted', () => {
    expect(tabsOf(moveTabWithin(three, 'one', 2), 'main')).toStrictEqual(['two', 'one', 'three'])
  })
})

describe('resizing', () => {
  it('takes the shares a handle reports', () => {
    const after = resizeBranch(sideBySide(), 'root', [0.4, 0.6])
    const boxes = arrangeWorkspace(after, SCREEN)

    expect(boxes.get('main')?.width).toBeCloseTo(400)
    expect(broken(after)).toStrictEqual([])
  })

  it('ignores a node that does not divide anything', () => {
    const before = sideBySide()
    expect(resizeBranch(before, 'main', [0.5, 0.5])).toStrictEqual(before)
  })
})

describe('every gesture keeps the tree canonical', () => {
  const start = split('root', [stack('a', 'one', 'two'), split('down', [stack('b', 'three'), stack('c', 'four')])])

  it('through a run of them', () => {
    const made = createIdFactory()
    let workspace = workspaceOf(start, 'horizontal', 'a')

    const run: readonly ((current: Workspace) => Workspace)[] = [
      (current) => dropTab(current, { tab: 'two', onto: 'c', side: 'bottom' }, made),
      (current) => openTab(current, 'five', 'b'),
      (current) => dropOnEdge(current, 'three', 'left', made),
      (current) => closeTab(current, 'four'),
      (current) => dropTab(current, { tab: 'one', onto: 'b', side: 'center' }, made),
      (current) => closeTab(current, 'five'),
      (current) => closeTab(current, 'two'),
    ]

    for (const gesture of run) {
      workspace = gesture(workspace)
      expect(broken(workspace)).toStrictEqual([])
    }
  })
})
