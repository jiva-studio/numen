import { describe, expect, it } from 'vitest'
import { addLinks, getLinkTitle, readLinks } from './note-links.js'

describe('addLinks and readLinks', () => {
  it('reads what was written', () => {
    const text = addLinks('# Harbour\n', [
      { to: '[[notes/Quay]]', role: 'child', label: '' },
      { to: '[[notes/Jetty]]', role: 'jump', label: 'Daily ferry' },
    ])
    expect(readLinks(text)).toStrictEqual([
      { to: '[[notes/Quay]]', role: 'child', label: '' },
      { to: '[[notes/Jetty]]', role: 'jump', label: 'Daily ferry' },
    ])
    expect(text.endsWith('# Harbour\n')).toBe(true)
  })

  it('reads a note without links', () => {
    expect(readLinks('# Harbour\n')).toStrictEqual([])
  })

  it('refuses a note that has a frontmatter', () => {
    expect(() => addLinks('---\ntype: x\n---\n', [])).toThrow('frontmatter')
  })
})

describe('getLinkTitle', () => {
  it('gives the title of the note', () => {
    expect(getLinkTitle('[[notes/Quay#Dock|the quay]]')).toBe('Quay')
    expect(getLinkTitle('Quay.md')).toBe('Quay')
  })
})
