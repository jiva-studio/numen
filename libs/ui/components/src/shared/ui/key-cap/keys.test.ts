import { describe, expect, it } from 'vitest'
import { keyChord, overlayIcon } from './keys'

describe('a keystroke of one letter and the key alongside the space bar', () => {
  it('is retained with the key the keyboard in hand puts there', () => {
    expect(overlayIcon('MacIntel')).toBe('command')
    expect(overlayIcon('Linux x86_64')).toBe('control')
    expect(keyChord('g', 'MacIntel')).toEqual({ icons: ['command'], letter: 'G' })
    expect(keyChord('g', 'Linux x86_64')).toEqual({ icons: ['control'], letter: 'G' })
  })

  it('is drawn in capitals, whichever case it was named in', () => {
    expect(keyChord('G', 'Linux x86_64')).toEqual({ icons: ['control'], letter: 'G' })
  })

  it('carries Shift between the two where the chord holds it', () => {
    expect(keyChord('p', 'MacIntel', true)).toEqual({
      icons: ['command', 'shift'],
      letter: 'P',
    })
    expect(keyChord('p', 'Linux x86_64', true)).toEqual({
      icons: ['control', 'shift'],
      letter: 'P',
    })
  })
})
