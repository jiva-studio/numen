import { describe, expect, it } from 'vitest'
import { useRefRegistry } from './refs'

describe('the refs of a loop', () => {
  it('passes a key the same function until its element goes', () => {
    const { getRef } = useRefRegistry<string>(() => {})
    const first = getRef('a')

    expect(getRef('a')).toBe(first)
    expect(getRef('b')).not.toBe(first)

    first(null)

    expect(getRef('a')).not.toBe(first)
  })

  it('notifies of an element as it is set, and of null as it goes', () => {
    const calls: [string, unknown][] = []
    const { getRef } = useRefRegistry<string>((key, element) => calls.push([key, element]))
    const element = {}

    getRef('a')(element)
    getRef('a')(null)

    expect(calls).toStrictEqual([
      ['a', element],
      ['a', null],
    ])
  })

  it('keeps the function of a key whose element is set again', () => {
    const { getRef } = useRefRegistry<string>(() => {})
    const first = getRef('a')

    first({})
    first({})

    expect(getRef('a')).toBe(first)
  })
})
