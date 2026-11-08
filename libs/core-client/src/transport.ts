/**
 * The transport every service client of a window talks over.
 *
 * The application serves the page and answers its requests, so requests go to
 * the origin the page was served from.
 */
import { createConnectTransport } from '@connectrpc/connect-web'

const getOrigin = (): string =>
  typeof window === 'undefined' || !window.location ? '' : window.location.origin

export const transport = createConnectTransport({ baseUrl: getOrigin() })

/** Sends requests with keepalive, so they complete after the page unloads. */
export const keepaliveTransport = createConnectTransport({
  baseUrl: getOrigin(),
  fetch: (input, init) => globalThis.fetch(input, { ...init, keepalive: true }),
})
