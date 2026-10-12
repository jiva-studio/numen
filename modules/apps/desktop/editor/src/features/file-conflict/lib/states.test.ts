import { describe, expect, it } from 'vitest'
import { parseFileConflict } from './states'

describe('which conflict a screen’s word names', () => {
  it('names a file that is gone', () => {
    expect(parseFileConflict('gone')).toBe('gone')
  })

  it('names a file that is stale', () => {
    expect(parseFileConflict('stale')).toBe('stale')
  })

  it('names nothing for any other word', () => {
    expect(parseFileConflict('ready')).toBeNull()
    expect(parseFileConflict('')).toBeNull()
  })
})
