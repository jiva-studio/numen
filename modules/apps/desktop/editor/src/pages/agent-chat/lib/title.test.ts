import { describe, expect, it } from 'vitest'
import { createTabTitle } from './title'

describe('createTabTitle', () => {
  it('extracts first non-empty line and truncates with ellipsis when longer than limit', () => {
    expect(createTabTitle('Hello world')).toBe('Hello world')
    expect(createTabTitle('\n\n  First real line\nSecond line')).toBe('First real line')
    expect(
      createTabTitle('This is a very long line that definitely exceeds twenty four characters'),
    ).toBe('This is a very long…')
  })

  it('skips a line of nothing but blanks', () => {
    expect(createTabTitle('   \t \nReal question')).toBe('Real question')
  })

  it('keeps a line that is exactly as long as the limit', () => {
    expect(createTabTitle('abcdefghij', 10)).toBe('abcdefghij')
    expect(createTabTitle('abcdefghijk', 10)).toBe('abcdefghij…')
  })

  it('cuts at the last blank only when it falls past a third of the limit', () => {
    expect(createTabTitle('1234567 89abcdefghijklmnopqrstuvwxyz')).toBe('1234567 89abcdefghijklmn…')
    expect(createTabTitle('12345678 9abcdefghijklmnopqrstuvwxyz')).toBe('12345678 9abcdefghijklmn…')
    expect(createTabTitle('123456789 abcdefghijklmnopqrstuvwxyz')).toBe('123456789…')
  })

  it('drops the blanks that end the cut', () => {
    expect(createTabTitle('aaaaaaaaaaaaaaaaaaa  bbbbbbbbbbbbbbbb')).toBe('aaaaaaaaaaaaaaaaaaa…')
  })
})
