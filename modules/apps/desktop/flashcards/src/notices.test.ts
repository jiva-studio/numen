import { describe, expect, it } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'

import { raising } from './notices'

describe('what the window has to say', () => {
  it('names each thing once, so putting one away leaves the rest', () => {
    const one = raising()
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
    const one = raising()
    one.failed(new Error('the vault could not be read'))

    const said = one.notices.value[0]!
    expect(said.says).toBe('The vault could not be read.')
    expect(said.tone).toBe('alarm')
    expect(said.stay).toBe('kept')
  })

  // Pressing a tile whose count is a moment stale is refused, and what the
  // application said is what a person needs. How it travelled is not.
  it('says a refusal without the wire it came over', () => {
    const one = raising()
    one.failed(
      new ConnectError('this preset schedules nothing today: it is paused', Code.FailedPrecondition),
    )

    expect(one.notices.value[0]!.says).toBe(
      'This preset schedules nothing today: it is paused.',
    )
  })

  it('leaves a sentence that already ends where it ends', () => {
    const one = raising()
    one.failed(new Error('The deck could not be written.'))

    expect(one.notices.value[0]!.says).toBe('The deck could not be written.')
  })

  it('puts away nothing when the name is not one it holds', () => {
    const one = raising()
    one.says('the first', 'caution')
    one.putAway('nothing')

    expect(one.notices.value).toHaveLength(1)
  })
})
