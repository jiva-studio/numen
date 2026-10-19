import { describe, expect, it } from 'vitest'
import { getFileName, getRenamedPath, isNotePath } from '../src/workspace'

describe('a note path', () => {
  it('is a markdown file, or a path without an extension', () => {
    expect(isNotePath('Ideas/Plan.md')).toBe(true)
    expect(isNotePath('Ideas/Plan')).toBe(true)
  })

  it('is not a file of any other extension', () => {
    expect(isNotePath('Books/Guide.pdf')).toBe(false)
    expect(isNotePath('Ideas/Plan.txt')).toBe(false)
  })
})

describe('a vault path taken apart', () => {
  it('is filed under its last segment', () => {
    expect(getFileName('physics/Kelvin.md')).toBe('Kelvin.md')
    expect(getFileName('Root.md')).toBe('Root.md')
  })

  it('went where a rename took it, and nowhere where none did', () => {
    const renames = [{ from: 'a.md', to: 'b.md' }]

    expect(getRenamedPath(renames, 'a.md')).toBe('b.md')
    expect(getRenamedPath(renames, 'c.md')).toBe('')
  })
})
