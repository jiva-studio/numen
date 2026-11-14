/**
 * The transports a window talks to the core over.
 *
 * The application serves the page and answers its requests, so requests go to the origin the page
 * was served from.
 */
import type { Transport } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'

export type { Transport }

const getOrigin = (): string => (typeof window === 'undefined' ? '' : window.location.origin)

/** Sends requests to the origin the page was served from. */
export const createOriginTransport = (): Transport =>
  createConnectTransport({ baseUrl: getOrigin() })

/** Sends requests to the origin with keepalive, so they complete after the page unloads. */
export const createKeepaliveTransport = (): Transport =>
  createConnectTransport({
    baseUrl: getOrigin(),
    fetch: (input, init) => globalThis.fetch(input, { ...init, keepalive: true }),
  })

/** The header that names the part of the window a request comes from, for the list of running plugins. */
export const CALLER_HEADER = 'Numen-Caller'

const addCaller = (header: HeadersInit | undefined, caller: string): Headers => {
  const out = new Headers(header)
  out.set(CALLER_HEADER, caller)
  return out
}

/**
 * Sends every request over `wire` with the tag of the part of the window that makes it: `editor`,
 * `search`, `settings` and the like, or `plugin:` and the id of the plugin whose view asks.
 */
export const createCallerTransport = (caller: string, wire: Transport): Transport => ({
  unary: (...request) => {
    request[3] = addCaller(request[3], caller)
    return wire.unary(...request)
  },
  stream: (...request) => {
    request[3] = addCaller(request[3], caller)
    return wire.stream(...request)
  },
})
