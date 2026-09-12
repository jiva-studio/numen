/** Where the tree puts each node inside a box. */
import { describe, expect, it } from 'vitest'
import { arrangeWorkspace } from './arrange'
import { deep, sideBySide, split, stack, workspaceOf } from '../fixtures/build'

const SCREEN = { x: 0, y: 0, width: 1000, height: 600 }

describe('a row divides the width', () => {
  const boxes = arrangeWorkspace(sideBySide(), SCREEN)

  it('gives each pane its share', () => {
    expect(boxes.get('main')).toStrictEqual({ x: 0, y: 0, width: 720, height: 600 })
    expect(boxes.get('aside')).toStrictEqual({ x: 720, y: 0, width: 280, height: 600 })
  })

  it('leaves the whole box to the root', () => {
    expect(boxes.get('root')).toStrictEqual(SCREEN)
  })
})

describe('depth turns the orientation', () => {
  const boxes = arrangeWorkspace(
    workspaceOf(split('root', [stack('a', 'one'), split('down', [stack('b', 'two'), stack('c', 'three')])])),
    SCREEN,
  )

  it('divides the height one level down', () => {
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
  const boxes = arrangeWorkspace(sideBySide(), SCREEN, { gap: 10 })

  it('leaves it between the two', () => {
    const main = boxes.get('main')
    const aside = boxes.get('aside')
    expect((main?.x ?? 0) + (main?.width ?? 0) + 10).toBe(aside?.x)
  })

  it('fills the box with what is left', () => {
    const main = boxes.get('main')
    const aside = boxes.get('aside')
    expect((main?.width ?? 0) + (aside?.width ?? 0)).toBeCloseTo(SCREEN.width - 10)
  })
})

describe('a workspace holding one pane', () => {
  it('gives it everything', () => {
    const boxes = arrangeWorkspace(workspaceOf(stack('main', 'one')), SCREEN)
    expect(boxes.get('main')).toStrictEqual(SCREEN)
  })
})
