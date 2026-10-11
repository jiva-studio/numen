import { describe, expect, it } from 'vitest'
import { conflictIn } from './states'

describe('which conflict a screen’s word names', () => {
  it('names a file that is gone', () => {
    expect(conflictIn('gone')).toBe('gone')
  })

  it('names a file that is stale', () => {
    expect(conflictIn('stale')).toBe('stale')
  })

  it('names nothing for any other word', () => {
    expect(conflictIn('ready')).toBeNull()
    expect(conflictIn('')).toBeNull()
  })
})
