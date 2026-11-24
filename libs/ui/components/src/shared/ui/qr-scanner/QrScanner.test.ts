/** The scanner reports why a camera did not open, and stops a camera it opened when it unmounts. */
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { decodeFrame } from './decodeFrame'
import QrScanner from './QrScanner.vue'

vi.mock('./decodeFrame', () => ({ decodeFrame: vi.fn() }))

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

  it('reports that no camera was found', async () => {
    const scanner = mount(QrScanner, {
      props: { camera: () => Promise.reject(new DOMException('none', 'NotFoundError')) },
    })
    await flushPromises()
    expect(scanner.get('[role="alert"]').text()).toBe('No camera found.')
  })

  it.each([
    ['NotReadableError', 'The camera is used by another program.'],
    ['OverconstrainedError', 'The camera did not open.'],
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
  const decode = vi.mocked(decodeFrame)

  beforeEach(() => {
    vi.useFakeTimers()
    decode.mockReset()
    vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue(undefined)
    vi.spyOn(HTMLVideoElement.prototype, 'videoWidth', 'get').mockReturnValue(4)
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({
      drawImage: () => undefined,
      getImageData: () => ({}),
    } as unknown as CanvasRenderingContext2D)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  const open = async (continuous: boolean) => {
    const { stream, stop } = makeStream()
    const scanner = mount(QrScanner, {
      props: { camera: () => Promise.resolve(stream), continuous },
    })
    await flushPromises()
    return { scanner, stop }
  }
  const frames = (count: number) => vi.advanceTimersByTimeAsync(count * 125)

  it('reads the second code with the same mounted scanner and keeps the camera on', async () => {
    const { scanner, stop } = await open(true)
    decode.mockReturnValue('first')
    await frames(1)
    decode.mockReturnValue('second')
    await frames(1)
    expect(scanner.emitted('scanned')).toStrictEqual([['first'], ['second']])
    expect(stop).not.toHaveBeenCalled()
  })

  it('reports a code held in front of the camera once', async () => {
    const { scanner } = await open(true)
    decode.mockReturnValue('first')
    await frames(10)
    expect(scanner.emitted('scanned')).toStrictEqual([['first']])
  })

  it('stops the camera after the first code when it is not continuous', async () => {
    const { scanner, stop } = await open(false)
    decode.mockReturnValue('first')
    await frames(1)
    decode.mockReturnValue('second')
    await frames(1)
    expect(scanner.emitted('scanned')).toStrictEqual([['first']])
    expect(stop).toHaveBeenCalledOnce()
  })
})
