import { describe, expect, it } from 'vitest'
import { splitActions } from './overflow'
import type { TitleBarAction } from './types'

const action = (id: string): TitleBarAction => ({ id, title: id, icon: {} })

const five = ['a', 'b', 'c', 'd', 'e'].map(action)
const ids = (list: readonly TitleBarAction[]): string[] => list.map((one) => one.id)

describe('splitActions', () => {
  it('keeps every action visible when they all fit', () => {
    const split = splitActions(five, 5)

    expect(ids(split.visible)).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
    expect(split.overflow).toStrictEqual([])
  })

  it('keeps every action visible when there is more room than actions', () => {
    expect(splitActions(five, 9).overflow).toStrictEqual([])
  })

  it('leaves one slot for the menu button when some actions do not fit', () => {
    const split = splitActions(five, 3)

    expect(ids(split.visible)).toStrictEqual(['a', 'b'])
    expect(ids(split.overflow)).toStrictEqual(['c', 'd', 'e'])
  })

  it('keeps the order of the actions across the visible and the overflow lists', () => {
    const split = splitActions(five, 4)

    expect([...ids(split.visible), ...ids(split.overflow)]).toStrictEqual(ids(five))
  })

  it('moves every action to the menu when only one slot fits', () => {
    const split = splitActions(five, 1)

    expect(split.visible).toStrictEqual([])
    expect(ids(split.overflow)).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
  })

  it('moves every action to the menu when no slot fits', () => {
    const split = splitActions(five, 0)

    expect(split.visible).toStrictEqual([])
    expect(split.overflow).toHaveLength(5)
  })

  it('has nothing to split when there are no actions', () => {
    expect(splitActions([], 0)).toStrictEqual({ visible: [], overflow: [] })
  })

  it('keeps every action visible while the room is not measured yet', () => {
    expect(splitActions(five, Infinity).overflow).toStrictEqual([])
  })
})
