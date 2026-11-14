import { describe, expect, it } from 'vitest'
import { cutToFit } from './cutToFit'

const perLetter = (label: string) => [...label].length

describe('cutting a text to a room', () => {
  it('keeps a label that fills the room exactly', () => {
    expect(cutToFit('abc', 3, perLetter)).toBe('abc')
  })

  it('trims the end of the cut and leaves its start as written', () => {
    expect(cutToFit('  ab cd', 4, perLetter)).toBe('  a…')
  })

  it('returns the ellipsis alone where no text fits', () => {
    expect(cutToFit('abc', 0, perLetter)).toBe('…')
  })

  describe('with a width in UTF-16 units, where half an emoji still takes room', () => {
    const perUnit = (label: string) => label.length

    it('never cuts inside an emoji made of several code points', () => {
      expect(cutToFit('a👨‍👩‍👧b', 5, perUnit)).toBe('a…')
    })

    it('never cuts inside a flag', () => {
      expect(cutToFit('ab🇩🇪c', 5, perUnit)).toBe('ab…')
    })

    it('keeps a whole emoji that fits', () => {
      expect(cutToFit('a👨‍👩‍👧bc', 10, perUnit)).toBe('a👨‍👩‍👧…')
    })
  })
})
