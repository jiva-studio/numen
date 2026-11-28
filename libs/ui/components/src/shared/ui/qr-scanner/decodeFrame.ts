import jsQR from 'jsqr'

/** The pixels of one camera frame, four bytes each: red, green, blue and alpha. */
export interface Frame {
  data: Uint8ClampedArray
  width: number
  height: number
}

/** The text of the QR code a frame holds, or null where it holds none. */
export function decodeFrame(frame: Frame): string | null {
  return jsQR(frame.data, frame.width, frame.height)?.data ?? null
}
