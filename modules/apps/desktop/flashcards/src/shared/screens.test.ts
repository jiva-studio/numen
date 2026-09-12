/**
 * Going back lets go of what the screens being left hold.
 *
 * The order is what decides: a screen standing after the one gone to is left,
 * and one standing before it is not. Nothing here knows what any holding is.
 */
import { describe, expect, it } from 'vitest'

import { useScreens } from './screens'

/** Three screens, each holding one thing that says when it was let go of. */
const three = () => {
  const dropped: string[] = []
  const held = useScreens(['list', 'rows', 'one'] as const, {
    rows: [() => dropped.push('rows')],
    one: [() => dropped.push('one')],
  })
  return { dropped, ...held }
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
