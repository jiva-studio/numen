/**
 * The transport a client uses, with and without a window to read the origin from.
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

  it('loads where a window was served from an origin', async () => {
    vi.stubGlobal('window', { location: { origin: 'https://example.test' } })
    const { transport } = await import('./transport')
    expect(transport).toBeDefined()
  })
})
