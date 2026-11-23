/** The scanner says why a camera did not open, and stops a camera it opened when it goes away. */
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import QrScanner from './QrScanner.vue'

const makeStream = () => {
  const stop = vi.fn()
  return { stream: { getTracks: () => [{ stop }] } as unknown as MediaStream, stop }
}

describe('a QR scanner', () => {
  it('says the camera is not allowed and tells whoever listens', async () => {
    const scanner = mount(QrScanner, {
      props: { camera: () => Promise.reject(new DOMException('no', 'NotAllowedError')) },
    })
    await flushPromises()
    expect(scanner.get('[role="alert"]').text()).toBe('The camera is not allowed.')
    expect(scanner.emitted('failed')).toStrictEqual([['The camera is not allowed.']])
  })

  it('says no camera was found', async () => {
    const scanner = mount(QrScanner, {
      props: { camera: () => Promise.reject(new DOMException('none', 'NotFoundError')) },
    })
    await flushPromises()
    expect(scanner.get('[role="alert"]').text()).toBe('No camera was found.')
  })

  it('waits while the camera has not answered', () => {
    const scanner = mount(QrScanner, { props: { camera: () => new Promise<MediaStream>(() => undefined) } })
    expect(scanner.text()).toContain('Waiting for the camera.')
  })

  it('stops the camera when it goes away', async () => {
    const { stream, stop } = makeStream()
    const scanner = mount(QrScanner, { props: { camera: () => Promise.resolve(stream) } })
    await flushPromises()
    scanner.unmount()
    expect(stop).toHaveBeenCalledOnce()
  })
})
