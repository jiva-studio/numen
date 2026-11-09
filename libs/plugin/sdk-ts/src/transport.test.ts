/**
 * The origin a client uses, with and without a window to read it from.
 *
 * This suite runs in Node, where there is no window, so the plain import below
 * already represents every caller without one.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'

afterEach(() => {
  vi.unstubAllGlobals()
  vi.resetModules()
})

describe('the wire transport', () => {
  it('loads where there is no window', async () => {
    await expect(import('./transport')).resolves.toHaveProperty('transport')
  })

  it('asks at no origin where there is no window', async () => {
    const { getOrigin } = await import('./transport')
    expect(getOrigin()).toBe('')
  })

  it('asks at the origin a window was served from', async () => {
    vi.stubGlobal('window', { location: { origin: 'https://example.test' } })
    const { getOrigin, transport } = await import('./transport')
    expect(getOrigin()).toBe('https://example.test')
    expect(transport).toBeDefined()
  })
})
