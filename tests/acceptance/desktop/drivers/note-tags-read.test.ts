import { describe, expect, it } from 'vitest'
import { readTags } from './note-tags-read.js'

describe('readTags', () => {
  it('reads a list with one item for each tag', () => {
    expect(readTags('---\ntags:\n  - Port\n  - "Sea"\n---\nThe quay.')).toStrictEqual([
      'Port',
      'Sea',
    ])
  })

  it('reads a list in the flow form', () => {
    expect(readTags("---\ntags: [Port, 'Sea']\n---\nThe quay.")).toStrictEqual(['Port', 'Sea'])
  })

  it('reads a single tag written as a text', () => {
    expect(readTags('---\ntags: Port\n---\nThe quay.')).toStrictEqual(['Port'])
  })

  it('reads the tags of a note that has other keys', () => {
    expect(
      readTags('---\ntype: place\ntags:\n  - Port\nstatus: open\n---\nThe quay.'),
    ).toStrictEqual(['Port'])
  })

  it('reads no tag from a note without the key', () => {
    expect(readTags('---\ntype: place\n---\nThe quay.')).toStrictEqual([])
  })

  it('reads no tag from a note without frontmatter', () => {
    expect(readTags('The quay.')).toStrictEqual([])
  })

  it('reads no tag from an empty list', () => {
    expect(readTags('---\ntags: []\n---\nThe quay.')).toStrictEqual([])
  })

  it('does not read a tags line of the text below the frontmatter', () => {
    expect(readTags('---\ntype: place\n---\ntags: Port')).toStrictEqual([])
  })
})
