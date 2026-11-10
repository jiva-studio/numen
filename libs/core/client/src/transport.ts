/**
 * The transport every service client of a window talks over.
 *
 * The application serves the page and answers its requests, so requests go to
 * the origin the page was served from.
 */
import type { Transport } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'

const getOrigin = (): string => (typeof window === 'undefined' ? '' : window.location.origin)

let replacement: Transport | undefined

const createReplaceableTransport = (wire: Transport): Transport => ({
  unary: (...request) => (replacement ?? wire).unary(...request),
  stream: (...request) => (replacement ?? wire).stream(...request),
})

export const transport = createReplaceableTransport(
  createConnectTransport({ baseUrl: getOrigin() }),
)

/** Sends requests with keepalive, so they complete after the page unloads. */
export const keepaliveTransport = createReplaceableTransport(
  createConnectTransport({
    baseUrl: getOrigin(),
    fetch: (input, init) => globalThis.fetch(input, { ...init, keepalive: true }),
  }),
)

/**
 * Sends every request of every client over `next` until the returned function is called, which
 * restores what was installed before.
 */
export const replaceTransport = (next: Transport): (() => void) => {
  const previous = replacement
  replacement = next
  return () => {
    replacement = previous
  }
}
