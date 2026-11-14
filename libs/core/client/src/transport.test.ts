/**
 * The transports a window sends its requests over, with and without a window to read the origin
 * from.
 *
 * This suite runs in Node, where there is no window, so a transport made without a stubbed window
 * represents every caller without one.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createConnectTransport } from '@connectrpc/connect-web'
import { createKeepaliveTransport, createOriginTransport } from './transport'

vi.mock('@connectrpc/connect-web', { spy: true })

afterEach(() => {
  vi.clearAllMocks()
  vi.unstubAllGlobals()
})

describe('the origin transport', () => {
  it('is made where there is no window', () => {
    expect(createOriginTransport()).toBeDefined()
    expect(createConnectTransport).toHaveBeenCalledWith({ baseUrl: '' })
  })

  it('sends to the origin the window was served from', () => {
    vi.stubGlobal('window', { location: { origin: 'https://example.test' } })
    createOriginTransport()
    expect(createConnectTransport).toHaveBeenCalledWith({ baseUrl: 'https://example.test' })
  })
})

describe('the keepalive transport', () => {
  it('sends its requests with keepalive', async () => {
    const fetch = vi.fn((_input: unknown, _init: unknown) => Promise.resolve(new Response()))
    vi.stubGlobal('fetch', fetch)
    createKeepaliveTransport()

    const options = vi.mocked(createConnectTransport).mock.calls.find(([one]) => one.fetch)?.[0]
    await options?.fetch?.('https://example.test/rpc', { method: 'POST' })

    expect(fetch).toHaveBeenCalledWith('https://example.test/rpc', {
      method: 'POST',
      keepalive: true,
    })
  })
})
