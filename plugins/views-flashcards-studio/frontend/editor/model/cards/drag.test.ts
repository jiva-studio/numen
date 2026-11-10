import { describe, expect, it, vi } from 'vitest'
import { useDrag, type DragDeps } from './drag'
import type { InsertionPoint } from '../../../lib/cards'

const dragEvent = (): {
  readonly press: DragEvent
  readonly prevented: () => number
  readonly data: () => unknown[][]
} => {
  const preventDefault = vi.fn<() => void>()
  const setData = vi.fn<(format: string, data: string) => void>()
  const press = { preventDefault, dataTransfer: { setData } } as unknown as DragEvent
  return {
    press,
    prevented: () => preventDefault.mock.calls.length,
    data: () => setData.mock.calls,
  }
}

const keyEvent = (): { readonly press: KeyboardEvent; readonly prevented: () => number } => {
  const preventDefault = vi.fn<() => void>()
  const press = { preventDefault } as unknown as KeyboardEvent
  return { press, prevented: () => preventDefault.mock.calls.length }
}

const setup = (isMoved = true) => {
  const move = vi.fn<(dragged: string, target: InsertionPoint) => void>()
  const deps: DragDeps<InsertionPoint | undefined> = {
    order: () => ['a', 'b', 'c'],
    nowhere: undefined,
    isMoved: () => isMoved,
    move,
  }
  return { drag: useDrag(deps), move }
}

describe('useDrag pointer', () => {
  it('sets dragged item on startDrag', () => {
    const { drag } = setup()
    expect(drag.dragged.value).toBeNull()
    drag.startDrag('a', dragEvent().press)
    expect(drag.dragged.value).toBe('a')
  })

  it('writes the dragged item id to dataTransfer as plain text', () => {
    const { drag } = setup()
    const started = dragEvent()
    drag.startDrag('a', started.press)
    expect(started.data()).toEqual([['text/plain', 'a']])
  })

  it('sets hover target and prevents default when dragging', () => {
    const { drag } = setup()
    const overTarget = dragEvent()
    drag.startDrag('a', dragEvent().press)
    drag.hover('b', overTarget.press)
    expect(drag.at.value).toBe('b')
    expect(overTarget.prevented()).toBe(1)
  })

  it('ignores hover when nothing is dragged', () => {
    const { drag } = setup()
    const overTarget = dragEvent()
    drag.hover('b', overTarget.press)
    expect(drag.at.value).toBeUndefined()
    expect(overTarget.prevented()).toBe(0)
  })

  it('clears drag state on release', () => {
    const { drag } = setup()
    drag.startDrag('a', dragEvent().press)
    drag.hover('b', dragEvent().press)
    drag.release()
    expect(drag.dragged.value).toBeNull()
    expect(drag.at.value).toBeUndefined()
  })

  it('moves the dragged item to the target on drop and clears state', () => {
    const { drag, move } = setup()
    drag.startDrag('a', dragEvent().press)
    drag.hover('c', dragEvent().press)
    drag.drop()
    expect(move.mock.calls).toEqual([['a', 'c']])
    expect(drag.dragged.value).toBeNull()
    expect(drag.at.value).toBeUndefined()
  })

  it('does not move on drop when isMoved returns false', () => {
    const { drag, move } = setup(false)
    drag.startDrag('a', dragEvent().press)
    drag.hover('c', dragEvent().press)
    drag.drop()
    expect(move).not.toHaveBeenCalled()
  })

  it('does not move on drop when no target is hovered', () => {
    const { drag, move } = setup()
    drag.startDrag('a', dragEvent().press)
    drag.drop()
    expect(move).not.toHaveBeenCalled()
  })

  it('does not move on drop when nothing is dragged', () => {
    const move = vi.fn<(dragged: string, target: InsertionPoint) => void>()
    const drag = useDrag<InsertionPoint>({
      order: () => ['a'],
      nowhere: null,
      isMoved: () => true,
      move,
    })
    drag.drop()
    expect(move).not.toHaveBeenCalled()
  })
})

describe('useDrag keyboard', () => {
  it('moves the item one step and prevents default', () => {
    const { drag, move } = setup()
    const key = keyEvent()
    drag.step('c', 'up', key.press)
    expect(move.mock.calls).toEqual([['c', 'b']])
    expect(key.prevented()).toBe(1)
  })

  it('does not move when the item is already at the boundary', () => {
    const { drag, move } = setup()
    const key = keyEvent()
    drag.step('a', 'up', key.press)
    expect(move).not.toHaveBeenCalled()
    expect(key.prevented()).toBe(0)
  })

  it('does not move when isMoved returns false', () => {
    const { drag, move } = setup(false)
    const key = keyEvent()
    drag.step('c', 'up', key.press)
    expect(move).not.toHaveBeenCalled()
    expect(key.prevented()).toBe(0)
  })
})
