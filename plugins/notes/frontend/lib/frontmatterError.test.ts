import { describe, expect, it } from 'vitest'
import { getFrontmatterError } from './frontmatterError'

describe('the frontmatter a reader would refuse', () => {
  it('is none for a note without frontmatter', () => {
    expect(getFrontmatterError('# Tides\n')).toBe('')
  })

  it('is none for lists, nested maps and quoted values', () => {
    const note =
      '---\ntitle: "Tides: a study"\ntags: [moon,\n  sea]\nlinks:\n  - to: "[[Moon]]"\n---\nBody'

    expect(getFrontmatterError(note)).toBe('')
  })

  it('is reported for an unclosed bracket', () => {
    expect(getFrontmatterError('---\ntitle: [unclosed\n---\nBody')).toBe('frontmatter unreadable')
  })

  it('is reported for a key written twice', () => {
    expect(getFrontmatterError('---\ntitle: One\ntitle: Two\n---\nBody')).toBe(
      'frontmatter unreadable',
    )
  })
})
