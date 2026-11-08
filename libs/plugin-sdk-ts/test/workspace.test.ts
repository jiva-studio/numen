import { describe, expect, it } from 'vitest'
import { getFileName, getFileStem, getRenamedPath, isNotePath } from '../src/workspace'

describe('a note path', () => {
  it('is a file whose extension is md, in any case', () => {
    expect(isNotePath('Ideas/Plan.md')).toBe(true)
    expect(isNotePath('Ideas/Plan.MD')).toBe(true)
    expect(isNotePath('Ideas.2024/Plan.md')).toBe(true)
  })

  it('is not a file without an extension, nor a name that is only an extension', () => {
    expect(isNotePath('Ideas/Plan')).toBe(false)
    expect(isNotePath('Ideas.2024/Plan')).toBe(false)
    expect(isNotePath('Data/README')).toBe(false)
    expect(isNotePath('Ideas/.md')).toBe(false)
  })

  it('is not a file of any other extension', () => {
    expect(isNotePath('Books/Guide.pdf')).toBe(false)
    expect(isNotePath('Ideas/Plan.txt')).toBe(false)
    expect(isNotePath('Ideas.2024/Plan.txt')).toBe(false)
  })
})

describe('a vault path taken apart', () => {
  it('is filed under its last segment', () => {
    expect(getFileName('physics/Kelvin.md')).toBe('Kelvin.md')
    expect(getFileName('Root.md')).toBe('Root.md')
  })

  it('is named without its ending', () => {
    expect(getFileStem('physics/Kelvin.md')).toBe('Kelvin')
    expect(getFileStem('.archive')).toBe('.archive')
    expect(getFileStem('Root')).toBe('Root')
  })

  it('went where a rename took it, and nowhere where none did', () => {
    const renames = [{ from: 'a.md', to: 'b.md' }]

    expect(getRenamedPath(renames, 'a.md')).toBe('b.md')
    expect(getRenamedPath(renames, 'c.md')).toBe('')
  })
})
