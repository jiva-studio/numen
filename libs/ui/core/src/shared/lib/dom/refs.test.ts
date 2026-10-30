import { describe, expect, it, vi } from 'vitest'
import { createRefRegistry } from './refs'

describe('the refs of a loop', () => {
  it('passes a key the same function until its element goes', () => {
    const { getRef } = createRefRegistry<string>(vi.fn())
    const first = getRef('a')

    expect(getRef('a')).toBe(first)
    expect(getRef('b')).not.toBe(first)

    first(null)

    expect(getRef('a')).not.toBe(first)
  })

  it('notifies of an element as it is set, and of null as it goes', () => {
    const calls: [string, unknown][] = []
    const { getRef } = createRefRegistry<string>((key, element) => calls.push([key, element]))
    const element = {}

    getRef('a')(element)
    getRef('a')(null)

    expect(calls).toStrictEqual([
      ['a', element],
      ['a', null],
    ])
  })

  it('keeps the function of a key whose element is set again', () => {
    const { getRef } = createRefRegistry<string>(vi.fn())
    const first = getRef('a')

    first({})
    first({})

    expect(getRef('a')).toBe(first)
  })
})
