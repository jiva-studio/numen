/**
 * The transport every service client of a window talks over.
 *
 * The application serves the page and answers its requests, so requests go to
 * the origin the page was served from.
 */
import { createConnectTransport } from '@connectrpc/connect-web'

/** The origin the page was served from, or empty where there is no window to ask. */
const getOrigin = (): string =>
  typeof window === 'undefined' || !window.location ? '' : window.location.origin

export const transport = createConnectTransport({ baseUrl: getOrigin() })

/**
 * A transport whose requests complete after the page unloads, for a request
 * sent on `pagehide`. A plain fetch is cancelled with the page.
 */
export const keepaliveTransport = createConnectTransport({
  baseUrl: getOrigin(),
  fetch: (input, init) => globalThis.fetch(input, { ...init, keepalive: true }),
})
