import { describe, expect, it } from 'vitest'
import { getFileName, getFileStem } from './paths'

describe('what a file is filed as', () => {
  it('is the last segment of the path', () => {
    expect(getFileName('stencils/cards/Animal.md')).toBe('Animal.md')
  })

  it('is the whole of a path standing in no folder', () => {
    expect(getFileName('Animal.md')).toBe('Animal.md')
  })

  it('is nothing for a file filed nowhere', () => {
    expect(getFileName('')).toBe('')
  })
})

describe('the name a link writes', () => {
  it('is the file, without the folders above it and without the extension', () => {
    expect(getFileStem('stencils/cards/Animal.md')).toBe('Animal')
  })

  it('keeps a name a dot remains inside, and one carrying no extension at all', () => {
    expect(getFileStem('stencils/Animal v2.md')).toBe('Animal v2')
    expect(getFileStem('stencils/Animal')).toBe('Animal')
  })

  it('keeps the whole of a name that is only an ending', () => {
    expect(getFileStem('.hidden')).toBe('.hidden')
    expect(getFileStem('notes/.hidden')).toBe('.hidden')
  })

  it('is nothing for a file filed nowhere', () => {
    expect(getFileStem('')).toBe('')
  })
})
