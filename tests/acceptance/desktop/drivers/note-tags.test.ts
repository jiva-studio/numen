import { describe, expect, it } from 'vitest'
import { addTags, groupTags } from './note-tags.js'

describe('addTags', () => {
  it('writes the tags of a note that has no frontmatter in front of its text', () => {
    expect(addTags('The quay.', ['Port', 'Sea'])).toBe(
      '---\ntags:\n  - "Port"\n  - "Sea"\n---\n\nThe quay.',
    )
  })

  it('adds the tags to the frontmatter a note has', () => {
    expect(addTags('---\ntype: place\n---\nThe quay.', ['Port'])).toBe(
      '---\ntags:\n  - "Port"\ntype: place\n---\nThe quay.',
    )
  })

  it('reports an error for a note that has tags already', () => {
    expect(() => addTags('---\ntags: [Port]\n---\n', ['Sea'])).toThrow('tags key')
  })
})

describe('groupTags', () => {
  it('gathers the rows of one note in the order of the table', () => {
    const grouped = groupTags([
      ['note', 'tag'],
      ['Quay', 'Port'],
      ['Dock', 'Sea'],
      ['Quay', 'Sea'],
    ])
    expect([...grouped]).toStrictEqual([
      ['Quay', ['Port', 'Sea']],
      ['Dock', ['Sea']],
    ])
  })

  it('reports an error for a table without the header note and tag', () => {
    expect(() => groupTags([['file', 'tag']])).toThrow('header')
  })
})
