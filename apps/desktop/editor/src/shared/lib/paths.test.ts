import { describe, expect, it } from 'vitest'
import { generateUniqueName, getFileName } from './paths'

describe('what a file is filed as', () => {
  it('is the last segment of the path', () => {
    expect(getFileName('forms/samples/Animal.md')).toBe('Animal.md')
  })

  it('is the whole of a path standing in no folder', () => {
    expect(getFileName('Animal.md')).toBe('Animal.md')
  })

  it('is nothing for a file filed nowhere', () => {
    expect(getFileName('')).toBe('')
  })
})

describe('a name nothing in a folder carries', () => {
  it('is the word itself where nothing is called that', () => {
    expect(generateUniqueName(['physics'], 'New folder')).toBe('New folder')
  })

  it('is the word and a count where it is taken', () => {
    expect(generateUniqueName(['New folder'], 'New folder')).toBe('New folder 2')
  })

  it('counts past every one that is taken', () => {
    expect(generateUniqueName(['New folder', 'New folder 2'], 'New folder')).toBe('New folder 3')
  })
})
