/**
 * The wire every client in a window talks over.
 *
 * The application serves the page and answers it, so it is asked at the origin
 * the page came from.
 */
import { createConnectTransport } from '@connectrpc/connect-web'

/** The origin the page was served from, or empty where there is no window to ask. */
export const getOrigin = (): string =>
  typeof window === 'undefined' || !window.location ? '' : window.location.origin

export const transport = createConnectTransport({ baseUrl: getOrigin() })
