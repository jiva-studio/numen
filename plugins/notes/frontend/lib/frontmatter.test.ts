import { describe, expect, it } from 'vitest'
import { parseFrontmatter } from './frontmatter'

describe('parseFrontmatter', () => {
  it('returns empty data when no frontmatter exists', () => {
    const text = '# Hello world\nThis is a note.'
    const result = parseFrontmatter(text)
    expect(result.data).toStrictEqual({})
    expect(result.content).toBe(text)
  })

  it('parses valid frontmatter keys and values', () => {
    const text = `---
title: My First Note
tags: note
pinned: true
order: 42
---
# Content starts here`

    const result = parseFrontmatter(text)
    expect(result.data).toStrictEqual({
      title: 'My First Note',
      tags: 'note',
      pinned: true,
      order: 42,
    })
    expect(result.content).toBe('# Content starts here')
  })

  it('handles unclosed frontmatter gracefully', () => {
    const text = '---\ntitle: Missing closing delimiter\nNo closing'
    const result = parseFrontmatter(text)
    expect(result.data).toStrictEqual({})
    expect(result.content).toBe(text)
  })
})
