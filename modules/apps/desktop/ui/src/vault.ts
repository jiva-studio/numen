/**
 * Asking the application about the vault it is showing.
 *
 * Nothing here describes what an answer looks like: that is the schema, and
 * both halves are generated from it.
 */
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { VaultService } from '@numen/protocol'

export const vault = createClient(
  VaultService,
  createConnectTransport({ baseUrl: window.location.origin }),
)
