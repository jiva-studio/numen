import { describe, expect, it } from 'vitest'
import { getFileConflict } from './states'

describe('which conflict a screen’s word names', () => {
  it('names a file that is gone', () => {
    expect(getFileConflict('gone')).toBe('gone')
  })

  it('names a file that is stale', () => {
    expect(getFileConflict('stale')).toBe('stale')
  })

  it('names nothing for any other word', () => {
    expect(getFileConflict('ready')).toBeNull()
    expect(getFileConflict('')).toBeNull()
  })
})
