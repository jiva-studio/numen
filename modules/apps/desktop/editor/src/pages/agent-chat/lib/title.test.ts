import { describe, expect, it } from 'vitest'
import { getFirstLine } from './title'

describe('getFirstLine', () => {
  it('extracts first non-empty line and truncates with ellipsis when longer than limit', () => {
    expect(getFirstLine('Hello world')).toBe('Hello world')
    expect(getFirstLine('\n\n  First real line\nSecond line')).toBe('First real line')
    expect(
      getFirstLine('This is a very long line that definitely exceeds twenty four characters'),
    ).toBe('This is a very long…')
  })

  it('skips a line of nothing but blanks', () => {
    expect(getFirstLine('   \t \nReal question')).toBe('Real question')
  })

  it('keeps a line that is exactly as long as the limit', () => {
    expect(getFirstLine('abcdefghij', 10)).toBe('abcdefghij')
    expect(getFirstLine('abcdefghijk', 10)).toBe('abcdefghij…')
  })

  it('cuts at the last blank only when it falls past a third of the limit', () => {
    expect(getFirstLine('1234567 89abcdefghijklmnopqrstuvwxyz')).toBe('1234567 89abcdefghijklmn…')
    expect(getFirstLine('12345678 9abcdefghijklmnopqrstuvwxyz')).toBe('12345678 9abcdefghijklmn…')
    expect(getFirstLine('123456789 abcdefghijklmnopqrstuvwxyz')).toBe('123456789…')
  })

  it('drops the blanks that end the cut', () => {
    expect(getFirstLine('aaaaaaaaaaaaaaaaaaa  bbbbbbbbbbbbbbbb')).toBe('aaaaaaaaaaaaaaaaaaa…')
  })
})
