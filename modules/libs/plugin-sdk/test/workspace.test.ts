import { describe, expect, it } from 'vitest'
import { isNotePath } from '../src/workspace'

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
