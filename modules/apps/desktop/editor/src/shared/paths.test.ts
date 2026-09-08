import { describe, expect, it } from 'vitest'
import { fileOf, nameOf } from './paths'

describe('what a file is filed as', () => {
  it('is the last segment of the path', () => {
    expect(fileOf('stencils/cards/Animal.md')).toBe('Animal.md')
  })

  it('is the whole of a path standing in no folder', () => {
    expect(fileOf('Animal.md')).toBe('Animal.md')
  })

  it('is nothing for a file filed nowhere', () => {
    expect(fileOf('')).toBe('')
  })
})

describe('the name a link writes', () => {
  it('is the file, without the folders above it and without the extension', () => {
    expect(nameOf('stencils/cards/Animal.md')).toBe('Animal')
  })

  it('keeps a name a dot stands inside, and one carrying no extension at all', () => {
    expect(nameOf('stencils/Animal v2.md')).toBe('Animal v2')
    expect(nameOf('stencils/Animal')).toBe('Animal')
  })

  it('is nothing for a file filed nowhere', () => {
    expect(nameOf('')).toBe('')
  })
})
