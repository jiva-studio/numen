/**
 * The owner a full command id starts with.
 */
import { describe, expect, it } from 'vitest'
import { getCommandOwner } from './commandId'

describe('the owner of a full command id', () => {
  it('is the text before the first colon', () => {
    expect(getCommandOwner('md.numen.x:a.b')).toBe('md.numen.x')
    expect(getCommandOwner('md.numen.x:a.b:c')).toBe('md.numen.x')
  })

  it('is empty for an id with no colon', () => {
    expect(getCommandOwner('a.b')).toBe('')
  })
})
