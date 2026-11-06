import { describe, expect, it } from 'vitest'
import { findGroupStarts } from './groups'
import type { TitleBarCommand } from '@numen/plugin-sdk'

const command = (id: string, group?: string): TitleBarCommand => ({
  id,
  title: id,
  icon: {},
  ...(group === undefined ? {} : { group }),
})

describe('findGroupStarts', () => {
  it('marks the first command of each group but the first group', () => {
    const starts = findGroupStarts([
      command('a', 'navigate'),
      command('b', 'navigate'),
      command('c', 'new'),
      command('d', 'new'),
      command('e', 'tools'),
    ])

    expect([...starts]).toStrictEqual(['c', 'e'])
  })

  it('marks nothing when all commands share a group', () => {
    expect(findGroupStarts([command('a', 'x'), command('b', 'x')]).size).toBe(0)
  })

  it('marks the first command with no group after the commands that have one', () => {
    const starts = findGroupStarts([command('a', 'x'), command('b'), command('c')])

    expect([...starts]).toStrictEqual(['b'])
  })

  it('marks nothing when no command names a group', () => {
    expect(findGroupStarts([command('a'), command('b')]).size).toBe(0)
  })

  it('marks nothing for an empty list', () => {
    expect(findGroupStarts([]).size).toBe(0)
  })
})
