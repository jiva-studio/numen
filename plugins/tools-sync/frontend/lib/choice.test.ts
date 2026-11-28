import { describe, expect, it } from 'vitest'
import { getChoice } from './choice'

describe('getChoice', () => {
  it('keeps the option that was picked while it is offered', () => {
    expect(getChoice('b', ['a', 'b'])).toBe('b')
  })

  it('takes the first option when none was picked or the picked one is gone', () => {
    expect(getChoice('', ['a', 'b'])).toBe('a')
    expect(getChoice('c', ['a', 'b'])).toBe('a')
  })

  it('is empty where nothing is offered', () => {
    expect(getChoice('a', [])).toBe('')
  })
})
