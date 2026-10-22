import { describe, expect, it } from 'vitest'
import { parseFileConflict } from '../src/buffers'

describe('which conflict a state names', () => {
  it('names a file that is gone', () => {
    expect(parseFileConflict('gone')).toBe('gone')
  })

  it('names a file that is stale', () => {
    expect(parseFileConflict('stale')).toBe('stale')
  })

  it('is empty for any other state', () => {
    expect(parseFileConflict('clean')).toBeNull()
    expect(parseFileConflict('')).toBeNull()
  })
})
