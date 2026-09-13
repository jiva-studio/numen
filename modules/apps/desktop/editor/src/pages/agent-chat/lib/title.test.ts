import { describe, expect, it } from 'vitest'
import { firstLine } from './title'

describe('firstLine', () => {
  it('extracts first non-empty line and truncates with ellipsis when longer than limit', () => {
    expect(firstLine('Hello world')).toBe('Hello world')
    expect(firstLine('\n\n  First real line\nSecond line')).toBe('First real line')
    expect(
      firstLine('This is a very long line that definitely exceeds twenty four characters'),
    ).toBe('This is a very long…')
  })
})
