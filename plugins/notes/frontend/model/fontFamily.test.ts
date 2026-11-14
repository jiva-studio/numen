import { describe, expect, it } from 'vitest'
import { getFontFamilyValue } from './fontFamily'

describe('the font family value', () => {
  it('keeps serif bare', () => {
    expect(getFontFamilyValue('serif')).toBe('serif')
  })

  it('keeps every generic family bare', () => {
    for (const name of ['sans-serif', 'monospace', 'system-ui', 'cursive', 'fantasy']) {
      expect(getFontFamilyValue(name)).toBe(name)
    }
  })

  it('quotes a name with spaces', () => {
    expect(getFontFamilyValue('Fira Code')).toBe('"Fira Code"')
  })

  it('keeps a list in order', () => {
    expect(getFontFamilyValue('Fira Code, Menlo ,monospace')).toBe(
      '"Fira Code", "Menlo", monospace',
    )
  })

  it('returns an empty text for an empty text, meaning the interface font', () => {
    expect(getFontFamilyValue('')).toBe('')
  })

  it('returns null for a list with no name in it', () => {
    expect(getFontFamilyValue(' , ')).toBeNull()
  })

  it('rejects a value that holds a semicolon or a brace', () => {
    expect(getFontFamilyValue('serif; color: red')).toBeNull()
    expect(getFontFamilyValue('a{b')).toBeNull()
    expect(getFontFamilyValue('a}b')).toBeNull()
  })

  it('rejects a value that holds a quote or a parenthesis', () => {
    expect(getFontFamilyValue('"Fira Code"')).toBeNull()
    expect(getFontFamilyValue("it's")).toBeNull()
    expect(getFontFamilyValue('url(x)')).toBeNull()
    expect(getFontFamilyValue('a)b')).toBeNull()
  })

  it('rejects a value that holds a backslash, an angle bracket or a control character', () => {
    expect(getFontFamilyValue('a\\b')).toBeNull()
    expect(getFontFamilyValue('a<b')).toBeNull()
    expect(getFontFamilyValue('a>b')).toBeNull()
    expect(getFontFamilyValue('a\nb')).toBeNull()
  })
})
