import { describe, expect, it } from 'vitest'
import { splitActions } from './overflow'
import type { TitleBarAction } from './types'

const action = (id: string): TitleBarAction => ({ id, title: id, icon: {} })

const five = ['a', 'b', 'c', 'd', 'e'].map(action)
const ids = (list: readonly TitleBarAction[]): string[] => list.map((one) => one.id)

describe('splitActions', () => {
  it('keeps every action visible when they all fit', () => {
    const lists = splitActions(five, 5)

    expect(ids(lists.visible)).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
    expect(lists.overflow).toStrictEqual([])
  })

  it('keeps every action visible when there is more room than actions', () => {
    expect(splitActions(five, 9).overflow).toStrictEqual([])
  })

  it('leaves one slot for the menu button when some actions do not fit', () => {
    const lists = splitActions(five, 3)

    expect(ids(lists.visible)).toStrictEqual(['a', 'b'])
    expect(ids(lists.overflow)).toStrictEqual(['c', 'd', 'e'])
  })

  it('keeps the order of the actions across the visible and the overflow lists', () => {
    const lists = splitActions(five, 4)

    expect([...ids(lists.visible), ...ids(lists.overflow)]).toStrictEqual(ids(five))
  })

  it('moves every action to the menu when only one slot fits', () => {
    const lists = splitActions(five, 1)

    expect(lists.visible).toStrictEqual([])
    expect(ids(lists.overflow)).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
  })

  it('moves every action to the menu when no slot fits', () => {
    const lists = splitActions(five, 0)

    expect(lists.visible).toStrictEqual([])
    expect(lists.overflow).toHaveLength(5)
  })

  it('has nothing to split when there are no actions', () => {
    expect(splitActions([], 0)).toStrictEqual({ visible: [], overflow: [] })
  })

  it('keeps every action visible while the room is not measured yet', () => {
    expect(splitActions(five, Infinity).overflow).toStrictEqual([])
  })
})
