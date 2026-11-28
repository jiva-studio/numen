/**
 * A camera for jsdom: it opens a stream and the QR code last shown to it is what the scanner decodes on its next frame.
 * Only the browser edge is replaced (the stream, the video and the canvas); the scanner and its decoder are the real ones.
 */
import { vi } from 'vitest'
import { encode } from 'uqr'

const CELL = 6
const QUIET = 4
const FRAME_MS = 125

const createFrame = (text: string): ImageData => {
  const { data: modules } = encode(text, { ecc: 'M', border: 0 })
  const side = (modules.length + QUIET * 2) * CELL
  const data = new Uint8ClampedArray(side * side * 4).fill(255)
  for (let y = 0; y < side; y += 1) {
    for (let x = 0; x < side; x += 1) {
      if (modules[Math.floor(y / CELL) - QUIET]?.[Math.floor(x / CELL) - QUIET] !== true) continue
      data.fill(0, (y * side + x) * 4, (y * side + x) * 4 + 3)
    }
  }
  return { data, width: side, height: side, colorSpace: 'srgb' }
}

/** Replaces the browser edge of the scanner for the test; `vi.restoreAllMocks` takes it away. */
export function createFakeCamera() {
  const blank: ImageData = {
    data: new Uint8ClampedArray(64 * 64 * 4).fill(255),
    width: 64,
    height: 64,
    colorSpace: 'srgb',
  }
  let shown = blank
  const stop = vi.fn()
  const stream: MediaStream = Object.assign(Object.create(null), { getTracks: () => [{ stop }] })
  const context: CanvasRenderingContext2D = Object.assign(Object.create(null), {
    drawImage: () => undefined,
    getImageData: () => shown,
  })
  const camera = vi.fn(() => Promise.resolve(stream))
  vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue(undefined)
  vi.spyOn(HTMLVideoElement.prototype, 'videoWidth', 'get').mockReturnValue(4)
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(context)

  return {
    camera,
    stop,
    show: (text: string): void => {
      shown = createFrame(text)
    },
    hide: (): void => {
      shown = blank
    },
    waitFrames: (count: number) => vi.advanceTimersByTimeAsync(count * FRAME_MS),
  }
}
