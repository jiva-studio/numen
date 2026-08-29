import { describe, expect, it } from 'vitest'

import { saying } from './saying'

describe('what the window has to say', () => {
  it('names each thing once, so putting one away leaves the rest', () => {
    const one = saying()
    one.says('the first', 'caution')
    one.says('the second', 'caution')

    const first = one.notices.value[0]!
    one.putAway(first.id)
    one.says('the third', 'caution')

    expect(one.notices.value.map((said) => said.says)).toEqual(['the second', 'the third'])
    const names = one.notices.value.map((said) => said.id)
    expect(new Set(names).size).toBe(names.length)
  })

  it('says trouble in the person’s own words, and stands until they put it away', () => {
    const one = saying()
    one.failed(new Error('the vault could not be read'))

    const said = one.notices.value[0]!
    expect(said.says).toContain('the vault could not be read')
    expect(said.tone).toBe('alarm')
    expect(said.stay).toBe('kept')
  })

  it('puts away nothing when the name is not one it holds', () => {
    const one = saying()
    one.says('the first', 'caution')
    one.putAway('nothing')

    expect(one.notices.value).toHaveLength(1)
  })
})
