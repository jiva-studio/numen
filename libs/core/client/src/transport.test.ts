/**
 * The transports a window sends its requests over, with and without a window to read the origin
 * from.
 *
 * This suite runs in Node, where there is no window, so a transport made without a stubbed window
 * represents every caller without one.
 */
import { createClient, createRouterTransport } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { PluginHealthService } from '@numen/protocol'
import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  CALLER_HEADER,
  createBearerTransport,
  createCallerTransport,
  createKeepaliveTransport,
  createOriginTransport,
} from './transport'

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

describe('the bearer transport', () => {
  it('sends each request to its address with the token', async () => {
    const authorized: (string | null)[] = []
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      authorized.push(new Headers(init?.headers).get('Authorization'))
      throw new Error(`no answer for ${String(input)}`)
    })
    vi.stubGlobal('fetch', fetch)

    const client = createClient(
      PluginHealthService,
      createBearerTransport('http://127.0.0.1:41234', 't0ken'),
    )
    await expect(
      client.clearHealth({ pluginId: 'org.example.atlas', key: 'quota' }),
    ).rejects.toThrow()

    expect(String(fetch.mock.calls[0]?.[0])).toBe(
      'http://127.0.0.1:41234/numen.v1.PluginHealthService/ClearHealth',
    )
    expect(authorized).toEqual(['Bearer t0ken'])
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

describe('the caller transport', () => {
  it('routes unary and stream calls through the wire transport with the caller header', async () => {
    expect(CALLER_HEADER).toBe('Numen-Caller')

    const callers: Array<string | null> = []
    const wire = createRouterTransport(({ service }) => {
      service(PluginHealthService, {
        clearHealth: (_request, context) => {
          callers.push(context.requestHeader.get('Numen-Caller'))
          return {}
        },
        watchHealth: async function* (_request, context) {
          callers.push(context.requestHeader.get('Numen-Caller'))
          yield {}
        },
      })
    })

    const client = createClient(PluginHealthService, createCallerTransport('search', wire))
    await client.clearHealth({ pluginId: 'org.example.atlas', key: 'quota' })
    for await (const update of client.watchHealth({})) {
      expect(update.items).toStrictEqual([])
    }

    expect(callers).toStrictEqual(['search', 'search'])
  })
})
