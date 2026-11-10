import { describe, expect, it } from 'vitest'
import { parseEvents } from './events'

describe('parseEvents', () => {
  it('keeps an incomplete event as the rest', () => {
    expect(parseEvents('data: {"a":1}\n\ndata: {"b"')).toEqual({
      events: [{ name: 'message', data: '{"a":1}' }],
      rest: 'data: {"b"',
    })
  })
})
