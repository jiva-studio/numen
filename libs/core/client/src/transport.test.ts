/**
 * The transport a client uses, with and without a window to read the origin from.
 *
 * This suite runs in Node, where there is no window, so the plain import below
 * already represents every caller without one.
 */
import { createClient, createRouterTransport } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { PluginHealthService } from '@numen/protocol'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('@connectrpc/connect-web', { spy: true })

afterEach(() => {
  vi.clearAllMocks()
  vi.unstubAllGlobals()
  vi.resetModules()
})

describe('the wire transport', () => {
  it('loads where there is no window', async () => {
    await expect(import('./transport')).resolves.toHaveProperty('transport')
    expect(createConnectTransport).toHaveBeenCalledWith({ baseUrl: '' })
  })

  it('loads where a window was served from an origin', async () => {
    vi.stubGlobal('window', { location: { origin: 'https://example.test' } })
    const { transport } = await import('./transport')
    expect(transport).toBeDefined()
    expect(createConnectTransport).toHaveBeenCalledWith({ baseUrl: 'https://example.test' })
  })

  it('sends the requests of the keepalive transport with keepalive', async () => {
    const fetch = vi.fn((_input: unknown, _init: unknown) => Promise.resolve(new Response()))
    vi.stubGlobal('fetch', fetch)
    await import('./transport')

    const options = vi.mocked(createConnectTransport).mock.calls.find(([one]) => one.fetch)?.[0]
    await options?.fetch?.('https://example.test/rpc', { method: 'POST' })

    expect(fetch).toHaveBeenCalledWith('https://example.test/rpc', {
      method: 'POST',
      keepalive: true,
    })
  })

  it('routes unary and stream calls through the replaced transport with the caller header', async () => {
    const { CALLER_HEADER, createCallerTransport, replaceTransport, transport } =
      await import('./transport')
    expect(CALLER_HEADER).toBe('Numen-Caller')

    const callers: Array<string | null> = []
    const restore = replaceTransport(
      createRouterTransport(({ service }) => {
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
      }),
    )

    const client = createClient(PluginHealthService, createCallerTransport('search', transport))
    await client.clearHealth({ pluginId: 'org.example.atlas', key: 'quota' })
    for await (const update of client.watchHealth({})) {
      expect(update.items).toStrictEqual([])
    }
    restore()

    expect(callers).toStrictEqual(['search', 'search'])
  })
})
