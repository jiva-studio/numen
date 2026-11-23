import { describe, expect, it } from 'vitest'
import { getNotePath, parseComputer, parseHolder, parseInside, parseOwnedVault, parseWait } from './computer.js'

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

describe('parseWait', () => {
  it('reads the seconds and the vault after them', () => {
    expect(parseWait(' within "5" seconds')).toStrictEqual({ seconds: 5, vault: undefined })
    expect(parseWait(' within "15" seconds in its vault "Copy"')).toStrictEqual({ seconds: 15, vault: 'Copy' })
  })

  it('reads no seconds where the text names none', () => {
    expect(parseWait('')).toStrictEqual({ seconds: undefined, vault: undefined })
    expect(parseWait(' in the vault "Test"')).toStrictEqual({ seconds: undefined, vault: 'Test' })
  })

  it('fails on a text that names no vault', () => {
    expect(() => parseWait(' soon')).toThrow('does not name a vault')
  })
})

describe('getNotePath', () => {
  it('adds the extension of a note and keeps the folders of the title', () => {
    expect(getNotePath('Shopping list')).toBe('Shopping list.md')
    expect(getNotePath('Food/Shopping list')).toBe('Food/Shopping list.md')
  })
})
