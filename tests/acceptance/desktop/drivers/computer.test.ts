import { describe, expect, it } from 'vitest'
import { findConflictCopies, getNotePath, parseComputer, parseHolder, parseInside, parseOwnedVault } from './computer.js'

describe('parseComputer', () => {
  it('names the first computer by the person and the second by its words', () => {
    expect(parseComputer('the person')).toBe('first')
    expect(parseComputer('the second computer')).toBe('second')
  })

  it('fails on a computer the scenarios do not know', () => {
    expect(() => parseComputer('the third computer')).toThrow('does not name a computer')
  })
})

describe('parseOwnedVault', () => {
  it('reads the name after the vault or its vault', () => {
    expect(parseOwnedVault('the vault "Test"')).toBe('Test')
    expect(parseOwnedVault('its vault "Copy"')).toBe('Copy')
  })

  it('fails on text that names no vault', () => {
    expect(() => parseOwnedVault('a vault "Test"')).toThrow('does not name a vault')
  })
})

describe('parseHolder', () => {
  it('names the second computer with no vault', () => {
    expect(parseHolder('the second computer')).toStrictEqual({ computer: 'second', vault: undefined })
  })

  it('names the first computer by the vault of the person', () => {
    expect(parseHolder('the vault "Test"')).toStrictEqual({ computer: 'first', vault: 'Test' })
  })
})

describe('parseInside', () => {
  it('reads the vault after "in"', () => {
    expect(parseInside(' in its vault "Copy"')).toBe('Copy')
    expect(parseInside(' in the vault "Test"')).toBe('Test')
  })

  it('names no vault where the step has none', () => {
    expect(parseInside('')).toBeUndefined()
  })
})

describe('getNotePath', () => {
  it('adds the extension of a note and keeps the folders of the title', () => {
    expect(getNotePath('Shopping list')).toBe('Shopping list.md')
    expect(getNotePath('Food/Shopping list')).toBe('Food/Shopping list.md')
  })
})

describe('findConflictCopies', () => {
  const files = [
    { path: 'Shopping list.md', text: 'eggs' },
    { path: 'Shopping list (conflict 2026-11-23).md', text: 'bread\n' },
    { path: 'Shopping list (conflict 2026-11-23 2).md', text: 'milk' },
    { path: 'Food/Shopping list (conflict 2026-11-23).md', text: 'rice' },
    { path: 'Shopping lists (conflict 2026-11-23).md', text: 'tea' },
  ]

  it('finds the copies of the note in its folder, with the number of a taken name', () => {
    expect(findConflictCopies(files, 'Shopping list')).toStrictEqual(['bread', 'milk'])
  })

  it('finds the copies of a note of a folder and no copy of another note', () => {
    expect(findConflictCopies(files, 'Food/Shopping list')).toStrictEqual(['rice'])
    expect(findConflictCopies(files, 'Groceries')).toStrictEqual([])
  })
})
