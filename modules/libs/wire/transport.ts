/**
 * The wire every client in the window talks over.
 *
 * The application serves the page and answers it, so it is asked at the origin
 * the page came from.
 */
import { createConnectTransport } from '@connectrpc/connect-web'

export const transport = createConnectTransport({ baseUrl: window.location.origin })
