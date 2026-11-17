import { describe, expect, it } from 'vitest'
import { getTagColor } from './color'

const PALETTE_SIZE = 8

const WORDS = Array.from({ length: 200 }, (_, index) => `Tag ${index}`)

describe('the color of a tag', () => {
  it('is the number of one of the eight color tokens', () => {
    for (const word of WORDS) {
      const color = getTagColor(word)
      expect(Number.isInteger(color)).toBe(true)
      expect(color).toBeGreaterThanOrEqual(1)
      expect(color).toBeLessThanOrEqual(PALETTE_SIZE)
    }
  })

  it('is the same for the same text', () => {
    const first = WORDS.map(getTagColor)

    expect(WORDS.map(getTagColor)).toStrictEqual(first)
  })

  it('is the same for a text written with other letter case', () => {
    expect(getTagColor('PORT')).toBe(getTagColor('Port'))
    expect(getTagColor('port')).toBe(getTagColor('Port'))
  })

  it('is the same for a text written with a leading hash', () => {
    expect(getTagColor('#Port')).toBe(getTagColor('Port'))
    expect(getTagColor('#port')).toBe(getTagColor('Port'))
  })

  it('is the same for a text in another script written with other letter case', () => {
    expect(getTagColor('ПОРТ')).toBe(getTagColor('порт'))
    expect(getTagColor('Порт')).toBe(getTagColor('порт'))
  })

  it('tells apart tags that differ only in characters outside the basic plane', () => {
    const emoji = ['🔥', '🚀', '🐟', '😀', '👍', '💡', '🔔', '📌']

    expect(new Set(emoji.map(getTagColor)).size).toBeGreaterThan(1)
  })

  it('uses every color of the palette for a long list of different tags', () => {
    const used = new Set(WORDS.map(getTagColor))

    expect([...used].sort()).toStrictEqual([1, 2, 3, 4, 5, 6, 7, 8])
  })

  it('gives different tags more than one color', () => {
    expect(
      new Set(['Port', 'Sea', 'Quay', 'Dock', 'Pier', 'Harbour'].map(getTagColor)).size,
    ).toBeGreaterThan(1)
  })
})
