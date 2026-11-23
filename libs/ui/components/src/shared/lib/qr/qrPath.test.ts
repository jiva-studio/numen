/** The path of a QR code draws each run of dark squares as one rectangle. */
import { describe, expect, it } from 'vitest'
import { getQrPath } from './qrPath'

describe('getQrPath', () => {
  it('draws a run of dark squares as one rectangle, moved in by the quiet zone', () => {
    expect(getQrPath([[true, true, false, true]], 2)).toBe('M2 2h2v1h-2zM5 2h1v1h-1z')
  })

  it('draws nothing for a row of light squares', () => {
    expect(getQrPath([[false, false]], 2)).toBe('')
  })
})
