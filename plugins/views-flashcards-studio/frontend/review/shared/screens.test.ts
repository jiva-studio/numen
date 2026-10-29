/**
 * Going back releases what the screens being left hold.
 *
 * The order decides: a screen after the target screen is left, and a screen
 * before it is not. This test does not depend on the type of any resource a screen holds.
 */
import { describe, expect, it } from 'vitest'

import { useScreens } from './screens'

/** Three screens, each with one release callback that records when it ran. */
const three = () => {
  const dropped: string[] = []
  const target = useScreens(['list', 'rows', 'one'] as const, {
    rows: [() => dropped.push('rows')],
    one: [() => dropped.push('one')],
  })
  return { dropped, ...target }
}

describe('going to a screen', () => {
  it('opens on the first of them', () => {
    expect(three().on.value).toBe('list')
  })

  it('lets go of everything the screens after it hold', () => {
    const { dropped, goTo, on } = three()

    goTo('one')
    goTo('list')

    expect(dropped).toStrictEqual(['rows', 'one'])
    expect(on.value).toBe('list')
  })

  it('leaves what the screens before it hold', () => {
    const { dropped, goTo } = three()

    goTo('one')
    goTo('rows')

    expect(dropped).toStrictEqual(['one'])
  })

  it('lets go of nothing going further in', () => {
    const { dropped, goTo } = three()

    goTo('rows')
    dropped.length = 0
    goTo('one')

    expect(dropped).toStrictEqual([])
  })
})
