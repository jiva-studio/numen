/** A drag of something along an order, by the pointer and by the keyboard. */
import { describe, expect, it, vi } from 'vitest'
import { useDrag, type DragDeps } from './drag'
import type { InsertionPoint } from '../lib/order'

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
  const move = vi.fn<(dragged: string, at: InsertionPoint) => void>()
  const deps: DragDeps<InsertionPoint | undefined> = {
    order: () => ['a', 'b', 'c'],
    nowhere: undefined,
    isMoved: () => isMoved,
    move,
  }
  return { drag: useDrag(deps), move }
}

describe('useDrag, the pointer', () => {
  it('holds what was lifted, and nothing before', () => {
    const { drag } = setup()
    expect(drag.dragged.value).toBeNull()
    drag.lift('a', dragEvent().press)
    expect(drag.dragged.value).toBe('a')
  })

  it('passes the browser the name of what was lifted as plain text', () => {
    const { drag } = setup()
    const lifted = dragEvent()
    drag.lift('a', lifted.press)
    expect(lifted.data()).toEqual([['text/plain', 'a']])
  })

  it('answers a hover over a place that takes it, and says the drop may follow', () => {
    const { drag } = setup()
    const overTarget = dragEvent()
    drag.lift('a', dragEvent().press)
    drag.hover('b', overTarget.press)
    expect(drag.at.value).toBe('b')
    expect(overTarget.prevented()).toBe(1)
  })

  it('answers no hover while nothing is dragged', () => {
    const { drag } = setup()
    const overTarget = dragEvent()
    drag.hover('b', overTarget.press)
    expect(drag.at.value).toBeUndefined()
    expect(overTarget.prevented()).toBe(0)
  })

  it('lets go of everything on release', () => {
    const { drag } = setup()
    drag.lift('a', dragEvent().press)
    drag.hover('b', dragEvent().press)
    drag.release()
    expect(drag.dragged.value).toBeNull()
    expect(drag.at.value).toBeUndefined()
  })

  it('moves what was dropped where it was let go, and holds nothing after', () => {
    const { drag, move } = setup()
    drag.lift('a', dragEvent().press)
    drag.hover('c', dragEvent().press)
    drag.drop()
    expect(move.mock.calls).toEqual([['a', 'c']])
    expect(drag.dragged.value).toBeNull()
    expect(drag.at.value).toBeUndefined()
  })

  it('moves nothing dropped where letting go moves nothing', () => {
    const { drag, move } = setup(false)
    drag.lift('a', dragEvent().press)
    drag.hover('c', dragEvent().press)
    drag.drop()
    expect(move).not.toHaveBeenCalled()
  })

  it('moves nothing dropped over nothing that takes it', () => {
    const { drag, move } = setup()
    drag.lift('a', dragEvent().press)
    drag.drop()
    expect(move).not.toHaveBeenCalled()
  })

  it('moves nothing dropped when nothing is retained, whatever remains for nowhere', () => {
    const move = vi.fn<(dragged: string, at: InsertionPoint) => void>()
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

describe('useDrag, the keyboard', () => {
  it('moves what is retained one place and says the key is taken', () => {
    const { drag, move } = setup()
    const key = keyEvent()
    drag.step('c', 'up', key.press)
    expect(move.mock.calls).toEqual([['c', 'b']])
    expect(key.prevented()).toBe(1)
  })

  it('leaves the key alone where there is no place that way', () => {
    const { drag, move } = setup()
    const key = keyEvent()
    drag.step('a', 'up', key.press)
    expect(move).not.toHaveBeenCalled()
    expect(key.prevented()).toBe(0)
  })

  it('leaves the key alone where the step moves nothing', () => {
    const { drag, move } = setup(false)
    const key = keyEvent()
    drag.step('c', 'up', key.press)
    expect(move).not.toHaveBeenCalled()
    expect(key.prevented()).toBe(0)
  })
})
