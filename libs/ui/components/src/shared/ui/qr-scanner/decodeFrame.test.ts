/** A synthetic frame of a QR code is read back to its text, and a frame of no code is read as none. */
import { describe, expect, it } from 'vitest'
import { encode } from 'uqr'
import { decodeFrame, type Frame } from './decodeFrame'

const CELL = 6
const QUIET = 4
const LINK_TEXT = 'numen://link?v=1&c=AAAAAAAAAAAAAAAAAAAAAA&k=BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB'

const makeFrame = (text: string): Frame => {
  const { data: modules } = encode(text, { ecc: 'M', border: 0 })
  const side = (modules.length + QUIET * 2) * CELL
  const data = new Uint8ClampedArray(side * side * 4).fill(255)
  for (let y = 0; y < side; y += 1) {
    for (let x = 0; x < side; x += 1) {
      const row = Math.floor(y / CELL) - QUIET
      const column = Math.floor(x / CELL) - QUIET
      if (modules[row]?.[column] !== true) continue
      const at = (y * side + x) * 4
      data.fill(0, at, at + 3)
    }
  }
  return { data, width: side, height: side }
}

describe('decodeFrame', () => {
  it('reads the text of a QR code in a frame', () => {
    expect(decodeFrame(makeFrame(LINK_TEXT))).toBe(LINK_TEXT)
  })

  it('reads a frame of one grey as no code', () => {
    const data = new Uint8ClampedArray(64 * 64 * 4).fill(136)
    expect(decodeFrame({ data, width: 64, height: 64 })).toBeNull()
  })
})
