/**
 * Asking the application about the vault it is showing.
 *
 * Nothing here describes what an answer looks like: that is the schema, and
 * both halves are generated from it.
 */
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { VaultService } from '@numen/protocol'
import type { Core } from './showing'

export const vault = createClient(
  VaultService,
  createConnectTransport({ baseUrl: window.location.origin }),
)

/** The same questions, in the shape the window asks them. */
export const core: Core = {
  neighbourhood: (path) => vault.neighbourhood({ path }),
  opening: async () => (await vault.opening({})).note ?? null,
  state: () => vault.state({}),
  changes: (signal) => vault.changes({}, { signal }),
  focus: (signal) => vault.focus({}, { signal }),
}
