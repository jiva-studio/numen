/** Where the tree puts each node inside a box. */
import { describe, expect, it } from 'vitest'
import { arrangeWorkspace } from './arrange'
import { deep, sideBySide, split, stack, createWorkspace } from '../fixtures/build'

const SCREEN = { x: 0, y: 0, width: 1000, height: 600 }

describe('a row divides the width', () => {
  it('gives each pane its share', () => {
    const boxes = arrangeWorkspace(sideBySide(), SCREEN)
    expect(boxes.get('main')).toStrictEqual({ x: 0, y: 0, width: 720, height: 600 })
    expect(boxes.get('aside')).toStrictEqual({ x: 720, y: 0, width: 280, height: 600 })
  })

  it('leaves the whole box to the root', () => {
    const boxes = arrangeWorkspace(sideBySide(), SCREEN)
    expect(boxes.get('root')).toStrictEqual(SCREEN)
  })
})

describe('depth turns the orientation', () => {
  it('divides the height one level down', () => {
    const boxes = arrangeWorkspace(
      createWorkspace(
        split('root', [stack('a', 'one'), split('down', [stack('b', 'two'), stack('c', 'three')])]),
      ),
      SCREEN,
    )

    expect(boxes.get('b')).toStrictEqual({ x: 500, y: 0, width: 500, height: 300 })
    expect(boxes.get('c')).toStrictEqual({ x: 500, y: 300, width: 500, height: 300 })
  })

  it('turns again at every level below that', () => {
    const below = arrangeWorkspace(deep(), SCREEN)

    // Depth two divides the width.
    const three = below.get('c')
    const again = below.get('again')
    expect(three?.y).toBe(again?.y)
    expect(three?.x).toBeLessThan(again?.x ?? 0)

    // Depth three divides the height.
    const four = below.get('d')
    const five = below.get('e')
    expect(four?.x).toBe(five?.x)
    expect(four?.width).toBe(five?.width)
    expect(four?.y).toBeLessThan(five?.y ?? 0)
  })
})

describe('a gap is taken out before the shares are counted', () => {
  it('leaves it between the two', () => {
    const boxes = arrangeWorkspace(sideBySide(), SCREEN, { gap: 10 })
    const main = boxes.get('main')
    const aside = boxes.get('aside')
    expect((main?.x ?? 0) + (main?.width ?? 0) + 10).toBe(aside?.x)
  })

  it('fills the box with what is left', () => {
    const boxes = arrangeWorkspace(sideBySide(), SCREEN, { gap: 10 })
    const main = boxes.get('main')
    const aside = boxes.get('aside')
    expect((main?.width ?? 0) + (aside?.width ?? 0)).toBeCloseTo(SCREEN.width - 10)
  })
})

describe('a gap between three children', () => {
  const three = () =>
    createWorkspace(split('root', [stack('a', 'one'), stack('b', 'two'), stack('c', 'three')]))

  it('is left once between each neighbour', () => {
    const boxes = arrangeWorkspace(three(), SCREEN, { gap: 10 })
    const [a, b, c] = ['a', 'b', 'c'].map((id) => boxes.get(id))

    expect(a?.width).toBeCloseTo(980 / 3)
    expect(b?.x).toBeCloseTo(980 / 3 + 10)
    expect(c?.x).toBeCloseTo((2 * 980) / 3 + 20)
    expect((c?.x ?? 0) + (c?.width ?? 0)).toBeCloseTo(SCREEN.width)
  })

  it('takes nothing from a gap wider than the box', () => {
    const boxes = arrangeWorkspace(three(), { x: 0, y: 0, width: 10, height: 600 }, { gap: 50 })

    expect(boxes.get('a')?.width).toBe(0)
    expect(boxes.get('b')?.width).toBe(0)
  })
})

describe('a workspace holding one pane', () => {
  it('gives it everything', () => {
    const boxes = arrangeWorkspace(createWorkspace(stack('main', 'one')), SCREEN)
    expect(boxes.get('main')).toStrictEqual(SCREEN)
  })
})
