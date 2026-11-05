import { describe, expect, it } from 'vitest'
import { findGroupStarts } from './groups'
import type { TitleBarAction } from './types'

const action = (id: string, group?: string): TitleBarAction => ({
  id,
  title: id,
  icon: {},
  ...(group === undefined ? {} : { group }),
})

describe('findGroupStarts', () => {
  it('marks the first action of each group but the first group', () => {
    const starts = findGroupStarts([
      action('a', 'navigate'),
      action('b', 'navigate'),
      action('c', 'new'),
      action('d', 'new'),
      action('e', 'tools'),
    ])

    expect([...starts]).toStrictEqual(['c', 'e'])
  })

  it('marks nothing when all actions share a group', () => {
    expect(findGroupStarts([action('a', 'x'), action('b', 'x')]).size).toBe(0)
  })

  it('marks the first action with no group after the actions that have one', () => {
    const starts = findGroupStarts([action('a', 'x'), action('b'), action('c')])

    expect([...starts]).toStrictEqual(['b'])
  })

  it('marks nothing when no action names a group', () => {
    expect(findGroupStarts([action('a'), action('b')]).size).toBe(0)
  })

  it('marks nothing for an empty list', () => {
    expect(findGroupStarts([]).size).toBe(0)
  })
})
