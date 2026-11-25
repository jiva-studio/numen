/** The scanner reports why a camera did not open, and stops a camera it opened when it unmounts. */
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { encode } from 'uqr'
import type { Frame } from './decodeFrame'
import QrScanner from './QrScanner.vue'

const CELL = 6
const QUIET = 4

const makeFrame = (text: string): Frame => {
  const { data: modules } = encode(text, { ecc: 'M', border: 0 })
  const side = (modules.length + QUIET * 2) * CELL
  const data = new Uint8ClampedArray(side * side * 4).fill(255)
  for (let y = 0; y < side; y += 1) {
    for (let x = 0; x < side; x += 1) {
      if (modules[Math.floor(y / CELL) - QUIET]?.[Math.floor(x / CELL) - QUIET] !== true) continue
      data.fill(0, (y * side + x) * 4, (y * side + x) * 4 + 3)
    }
  }
  return { data, width: side, height: side }
}

const makeStream = () => {
  const stop = vi.fn()
  return { stream: { getTracks: () => [{ stop }] } as unknown as MediaStream, stop }
}

describe('a QR scanner', () => {
  it('reports a camera that is not allowed to the listener', async () => {
    const scanner = mount(QrScanner, {
      props: { camera: () => Promise.reject(new DOMException('no', 'NotAllowedError')) },
    })
    await flushPromises()
    expect(scanner.get('[role="alert"]').text()).toBe('The camera is not allowed.')
    expect(scanner.emitted('unavailable')).toStrictEqual([['The camera is not allowed.']])
  })

  it('gives up on a camera that does not open in ten seconds', async () => {
    vi.useFakeTimers()
    try {
      const scanner = mount(QrScanner, { props: { camera: () => new Promise<MediaStream>(() => undefined) } })
      await vi.advanceTimersByTimeAsync(10_500)
      expect(scanner.get('[role="alert"]').text()).toBe('The camera did not open.')
      expect(scanner.emitted('unavailable')).toHaveLength(1)
    } finally {
      vi.useRealTimers()
    }
  })

  it('reports that no camera was found', async () => {
    const scanner = mount(QrScanner, {
      props: { camera: () => Promise.reject(new DOMException('none', 'NotFoundError')) },
    })
    await flushPromises()
    expect(scanner.get('[role="alert"]').text()).toBe('No camera found.')
  })

  it.each([
    ['NotReadableError', 'The camera is used by another program.'],
    ['OverconstrainedError', 'No camera found.'],
    ['SecurityError', 'The camera did not open.'],
  ])('reports %s as a state and does not throw', async (name, text) => {
    const scanner = mount(QrScanner, {
      props: { camera: () => Promise.reject(new DOMException('x', name)) },
    })
    await flushPromises()
    expect(scanner.emitted('unavailable')).toStrictEqual([[text]])
  })

  it('reports a camera that throws at once', async () => {
    const scanner = mount(QrScanner, {
      props: {
        camera: () => {
          throw new TypeError('no media')
        },
      },
    })
    await flushPromises()
    expect(scanner.emitted('unavailable')).toStrictEqual([['The camera did not open.']])
  })

  it('shows a spinner in the box of the picture while the camera has not answered', () => {
    const scanner = mount(QrScanner, {
      props: { camera: () => new Promise<MediaStream>(() => undefined) },
    })
    expect(scanner.find('video').exists()).toBe(true)
    expect(scanner.get('[role="status"]').element.parentElement).toBe(scanner.element)
    expect(scanner.text()).not.toContain('Waiting for the camera')
  })

  it('stops the camera when the scanner unmounts', async () => {
    const { stream, stop } = makeStream()
    const scanner = mount(QrScanner, { props: { camera: () => Promise.resolve(stream) } })
    await flushPromises()
    scanner.unmount()
    expect(stop).toHaveBeenCalledOnce()
  })

  it('stops a camera that opens after the ten seconds are over', async () => {
    vi.useFakeTimers()
    try {
      const { stream, stop } = makeStream()
      let open: (opened: MediaStream) => void = () => undefined
      const scanner = mount(QrScanner, {
        props: { camera: () => new Promise<MediaStream>((resolve) => (open = resolve)) },
      })
      await vi.advanceTimersByTimeAsync(10_500)
      expect(scanner.emitted('unavailable')).toHaveLength(1)
      open(stream)
      await flushPromises()
      expect(stop).toHaveBeenCalledOnce()
    } finally {
      vi.useRealTimers()
    }
  })

  it('stops a camera that opens after the scanner unmounted', async () => {
    const { stream, stop } = makeStream()
    let open: (opened: MediaStream) => void = () => undefined
    const scanner = mount(QrScanner, {
      props: { camera: () => new Promise<MediaStream>((resolve) => (open = resolve)) },
    })
    scanner.unmount()
    open(stream)
    await flushPromises()
    expect(stop).toHaveBeenCalledOnce()
  })

  it('stops the camera and arms no timer when it unmounts while the video starts', async () => {
    vi.useFakeTimers()
    try {
      const { stream, stop } = makeStream()
      let play: () => void = () => undefined
      vi.spyOn(HTMLMediaElement.prototype, 'play').mockReturnValue(
        new Promise<void>((resolve) => (play = resolve)),
      )
      const scanner = mount(QrScanner, { props: { camera: () => Promise.resolve(stream) } })
      await flushPromises()
      const before = vi.getTimerCount()
      scanner.unmount()
      play()
      await flushPromises()
      expect(stop).toHaveBeenCalledOnce()
      expect(vi.getTimerCount()).toBe(before)
    } finally {
      vi.useRealTimers()
      vi.restoreAllMocks()
    }
  })

  it('draws a video that plays on its own, muted and inline', () => {
    const scanner = mount(QrScanner, {
      props: { camera: () => new Promise<MediaStream>(() => undefined) },
    })
    const video = scanner.get('video')
    expect(video.attributes()).toHaveProperty('autoplay')
    expect(video.attributes()).toHaveProperty('playsinline')
    expect(video.element.muted).toBe(true)
  })

  it('asks for the camera only when it mounts', async () => {
    const camera = vi.fn(() => new Promise<MediaStream>(() => undefined))
    expect(camera).not.toHaveBeenCalled()
    mount(QrScanner, { props: { camera } })
    await flushPromises()
    expect(camera).toHaveBeenCalledOnce()
  })

  it('has no frame over the picture and no button', () => {
    const scanner = mount(QrScanner, {
      props: { camera: () => new Promise<MediaStream>(() => undefined) },
    })
    expect(scanner.find('.qr-scanner__frame').exists()).toBe(false)
    expect(scanner.find('button').exists()).toBe(false)
  })
})

describe('a continuous QR scanner', () => {
  let shown: Frame = { data: new Uint8ClampedArray(64 * 64 * 4).fill(136), width: 64, height: 64 }
  const show = (text: string): void => {
    shown = makeFrame(text)
  }

  beforeEach(() => {
    vi.useFakeTimers()
    vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue(undefined)
    vi.spyOn(HTMLVideoElement.prototype, 'videoWidth', 'get').mockReturnValue(4)
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({
      drawImage: () => undefined,
      getImageData: () => shown,
    } as unknown as CanvasRenderingContext2D)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  const open = async (isContinuous: boolean) => {
    const { stream, stop } = makeStream()
    const scanner = mount(QrScanner, {
      props: { camera: () => Promise.resolve(stream), isContinuous },
    })
    await flushPromises()
    return { scanner, stop }
  }
  const frames = (count: number) => vi.advanceTimersByTimeAsync(count * 125)

  it('reads the second code with the same mounted scanner and keeps the camera on', async () => {
    const { scanner, stop } = await open(true)
    show('first')
    await frames(1)
    show('second')
    await frames(1)
    expect(scanner.emitted('scanned')).toStrictEqual([['first'], ['second']])
    expect(stop).not.toHaveBeenCalled()
  })

  it('reports a code that stays in view once', async () => {
    const { scanner } = await open(true)
    show('first')
    await frames(10)
    expect(scanner.emitted('scanned')).toStrictEqual([['first']])
  })

  it('stops the camera after the first code when it is not continuous', async () => {
    const { scanner, stop } = await open(false)
    show('first')
    await frames(1)
    show('second')
    await frames(1)
    expect(scanner.emitted('scanned')).toStrictEqual([['first']])
    expect(stop).toHaveBeenCalledOnce()
  })
})
