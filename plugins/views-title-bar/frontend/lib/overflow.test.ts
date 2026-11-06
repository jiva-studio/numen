import { describe, expect, it } from 'vitest'
import { splitCommands } from './overflow'
import type { TitleBarCommand } from '@numen/plugin-sdk'

const command = (id: string): TitleBarCommand => ({ id, title: id, icon: {} })

const five = ['a', 'b', 'c', 'd', 'e'].map(command)
const ids = (list: readonly TitleBarCommand[]): string[] => list.map((one) => one.id)

describe('splitCommands', () => {
  it('keeps every command visible when they all fit', () => {
    const lists = splitCommands(five, 5)

    expect(ids(lists.visible)).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
    expect(lists.overflow).toStrictEqual([])
  })

  it('keeps every command visible when there is more room than commands', () => {
    expect(splitCommands(five, 9).overflow).toStrictEqual([])
  })

  it('leaves one slot for the menu button when some commands do not fit', () => {
    const lists = splitCommands(five, 3)

    expect(ids(lists.visible)).toStrictEqual(['a', 'b'])
    expect(ids(lists.overflow)).toStrictEqual(['c', 'd', 'e'])
  })

  it('keeps the order of the commands across the visible and the overflow lists', () => {
    const lists = splitCommands(five, 4)

    expect([...ids(lists.visible), ...ids(lists.overflow)]).toStrictEqual(ids(five))
  })

  it('moves every command to the menu when only one slot fits', () => {
    const lists = splitCommands(five, 1)

    expect(lists.visible).toStrictEqual([])
    expect(ids(lists.overflow)).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
  })

  it('moves every command to the menu when no slot fits', () => {
    const lists = splitCommands(five, 0)

    expect(lists.visible).toStrictEqual([])
    expect(lists.overflow).toHaveLength(5)
  })

  it('has nothing to split when there are no commands', () => {
    expect(splitCommands([], 0)).toStrictEqual({ visible: [], overflow: [] })
  })

  it('keeps every command visible while the room is not measured yet', () => {
    expect(splitCommands(five, Infinity).overflow).toStrictEqual([])
  })
})
