/**
 * Putting a tree back into the one form its picture has, without moving
 * anything that is still on the screen.
 */
import { describe, expect, it } from 'vitest'
import { normalize } from './normalize'
import { pane } from './node'
import { isBranch, isPane } from './tree'
import { arrangeWorkspace } from '../fixtures/arrange'
import { split, stack, createWorkspace } from '../fixtures/build'
import { getFaults } from '../fixtures/invariants'

const SCREEN = { x: 0, y: 0, width: 1280, height: 800 }

describe('what does not draw is cleared away', () => {
  it('takes out a pane holding nothing', () => {
    const before = createWorkspace(
      split('root', [stack('a', 'one'), pane('gone', []), stack('b', 'two')]),
      'horizontal',
      'a',
    )
    const after = normalize(before)

    expect(after.root).toSatisfy(isBranch)
    expect(isBranch(after.root) && after.root.children.map((child) => child.id)).toStrictEqual([
      'a',
      'b',
    ])
    expect(getFaults(after)).toStrictEqual([])
  })

  it('replaces a branch left with one pane by that pane', () => {
    const before = createWorkspace(split('root', [stack('only', 'one')]), 'horizontal', 'only')
    expect(normalize(before).root.id).toBe('only')
  })

  it('keeps a root that holds nothing, so there is always one', () => {
    const before = createWorkspace(pane('main', []))
    const after = normalize(before)

    expect(after.root).toSatisfy(isPane)
    expect(isPane(after.root) && after.root.tabs).toStrictEqual([])
  })

  it('keeps the first pane of a branch that holds nothing at all, so there is a root', () => {
    const after = normalize(createWorkspace(split('root', [pane('x', []), pane('y', [])])))

    expect(after.root).toStrictEqual(pane('x', []))
  })

  it('keeps a branch without children as a pane under its own id', () => {
    const after = normalize(createWorkspace(split('root', [])))

    expect(after.root).toStrictEqual(pane('root', []))
  })

  it('keeps the shares of a child that is reduced to one pane', () => {
    const before = createWorkspace(
      split(
        'root',
        [split('lone', [stack('a', 'one'), pane('gone', [])]), stack('b', 'two')],
        [0.3, 0.7],
      ),
    )
    const after = normalize(before)

    expect(isBranch(after.root) && after.root.sizes).toStrictEqual([0.3, 0.7])
  })

  it('keeps the shares of a child that stays a branch', () => {
    const before = createWorkspace(
      split(
        'root',
        [split('pair', [stack('a', 'one'), stack('b', 'two')]), stack('c', 'three')],
        [0.4, 0.6],
      ),
    )
    const after = normalize(before)

    expect(isBranch(after.root) && after.root.sizes).toStrictEqual([0.4, 0.6])
  })

  it('is idempotent', () => {
    const once = normalize(
      createWorkspace(split('root', [stack('a', 'one'), pane('gone', [])]), 'horizontal', 'a'),
    )
    expect(normalize(once)).toStrictEqual(once)
  })
})

/** Grandchildren rise two levels at once, and keep the way they divide. */
describe('a branch that keeps one child passes its grandchildren up', () => {
  const withEmpty = createWorkspace(
    split(
      'root',
      [
        stack('a', 'one'),
        split('down', [
          pane('gone', []),
          split('across', [stack('c', 'three'), stack('d', 'four')]),
        ]),
      ],
      [0.5, 0.5],
    ),
    'horizontal',
    'a',
  )

  it('leaves the grandchildren where the branch was', () => {
    const after = normalize(withEmpty)
    expect(isBranch(after.root) && after.root.children.map((child) => child.id)).toStrictEqual([
      'a',
      'c',
      'd',
    ])
    expect(getFaults(after)).toStrictEqual([])
  })

  it('leaves them dividing their length the way they did', () => {
    const boxes = arrangeWorkspace(normalize(withEmpty), SCREEN)
    const three = boxes.get('c')
    const four = boxes.get('d')

    expect(three?.y).toBe(four?.y)
    expect(three?.height).toBe(four?.height)
    expect(three?.x).toBeLessThan(four?.x ?? 0)
  })

  it('gives them the room the branch had, and takes none from anyone else', () => {
    const before = arrangeWorkspace(withEmpty, SCREEN)
    const after = arrangeWorkspace(normalize(withEmpty), SCREEN)

    expect(after.get('a')).toStrictEqual(before.get('a'))
    expect(after.get('c')?.x).toBe(before.get('c')?.x)
    expect(after.get('c')?.width).toBe(before.get('c')?.width)
    expect(after.get('d')?.x).toBe(before.get('d')?.x)
  })
})

describe('a root left with one child turns the axis with it', () => {
  const workspace = createWorkspace(
    split('root', [split('down', [stack('c', 'three'), stack('d', 'four')])]),
    'horizontal',
    'c',
  )

  it('takes the grandchildren for the root', () => {
    const after = normalize(workspace)
    expect(isBranch(after.root) && after.root.children.map((child) => child.id)).toStrictEqual([
      'c',
      'd',
    ])
    expect(after.axis).toBe('vertical')
  })

  it('keeps the shares the grandchildren had', () => {
    const uneven = createWorkspace(
      split('root', [split('down', [stack('c', 'three'), stack('d', 'four')], [0.25, 0.75])]),
      'horizontal',
      'c',
    )

    const after = normalize(uneven)

    expect(isBranch(after.root) && after.root.sizes).toStrictEqual([0.25, 0.75])
  })

  it('draws every pane in exactly the box it had', () => {
    const before = arrangeWorkspace(workspace, SCREEN)
    const after = arrangeWorkspace(normalize(workspace), SCREEN)

    for (const id of ['c', 'd']) {
      expect(after.get(id)).toStrictEqual(before.get(id))
    }
  })
})
