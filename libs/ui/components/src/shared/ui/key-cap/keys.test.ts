import { describe, expect, it } from 'vitest'
import { getKeyChord } from './keys'
import { overlayIcon } from './overlayIcon'

describe('a keystroke of one letter and the key alongside the space bar', () => {
  it('is retained with the key the keyboard in hand puts there', () => {
    expect(overlayIcon('MacIntel')).toBe('command')
    expect(overlayIcon('Linux x86_64')).toBe('control')
    expect(getKeyChord('g', 'MacIntel')).toEqual({ icons: ['command'], letter: 'G' })
    expect(getKeyChord('g', 'Linux x86_64')).toEqual({ icons: ['control'], letter: 'G' })
  })

  it('is drawn in capitals, whichever case it was named in', () => {
    expect(getKeyChord('G', 'Linux x86_64')).toEqual({ icons: ['control'], letter: 'G' })
  })

  it('carries Shift between the two where the chord holds it', () => {
    expect(getKeyChord('p', 'MacIntel', true)).toEqual({
      icons: ['command', 'shift'],
      letter: 'P',
    })
    expect(getKeyChord('p', 'Linux x86_64', true)).toEqual({
      icons: ['control', 'shift'],
      letter: 'P',
    })
  })
})
