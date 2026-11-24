/** The scanner reports why a camera did not open, and stops a camera it opened when it unmounts. */
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import QrScanner from './QrScanner.vue'

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

  it('waits while the camera has not answered', () => {
    const scanner = mount(QrScanner, {
      props: { camera: () => new Promise<MediaStream>(() => undefined) },
    })
    expect(scanner.text()).toContain('Waiting for the camera.')
  })

  it('stops the camera when the scanner unmounts', async () => {
    const { stream, stop } = makeStream()
    const scanner = mount(QrScanner, { props: { camera: () => Promise.resolve(stream) } })
    await flushPromises()
    scanner.unmount()
    expect(stop).toHaveBeenCalledOnce()
  })
})
