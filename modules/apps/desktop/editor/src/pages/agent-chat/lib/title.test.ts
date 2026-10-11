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

  it('skips a line of nothing but blanks', () => {
    expect(firstLine('   \t \nReal question')).toBe('Real question')
  })

  it('keeps a line that is exactly as long as the limit', () => {
    expect(firstLine('abcdefghij', 10)).toBe('abcdefghij')
    expect(firstLine('abcdefghijk', 10)).toBe('abcdefghij…')
  })

  it('cuts at the last blank only when it falls past a third of the limit', () => {
    expect(firstLine('1234567 89abcdefghijklmnopqrstuvwxyz')).toBe('1234567 89abcdefghijklmn…')
    expect(firstLine('12345678 9abcdefghijklmnopqrstuvwxyz')).toBe('12345678 9abcdefghijklmn…')
    expect(firstLine('123456789 abcdefghijklmnopqrstuvwxyz')).toBe('123456789…')
  })

  it('drops the blanks that end the cut', () => {
    expect(firstLine('aaaaaaaaaaaaaaaaaaa  bbbbbbbbbbbbbbbb')).toBe('aaaaaaaaaaaaaaaaaaa…')
  })
})
