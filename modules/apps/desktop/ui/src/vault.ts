/**
 * Asking the application about the vault it is showing.
 *
 * Nothing here describes what an answer looks like: that is the schema, and
 * both halves are generated from it.
 */
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { Refusal, VaultService } from '@numen/protocol'
import type { Answered, Core, Refused } from './showing'

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
  read: async (path) => answered(await vault.read({ path })),
  write: async (path, body) => answered(await vault.write({ path, body })),
  quitting: (signal) => vault.quitting({}, { signal }),
  flushed: async (token) => {
    await vault.flushed({ token })
  },
}

/** The schema's answer in the words the window uses. */
const answered = (from: { body?: string | undefined; refusal?: Refusal | undefined }): Answered => ({
  body: from.body ?? '',
  refusal: from.refusal === undefined ? null : refused[from.refusal],
})

const refused: Record<Refusal, Refused> = {
  [Refusal.UNSPECIFIED]: 'unreadable',
  [Refusal.MISSING]: 'missing',
  [Refusal.NOT_A_NOTE]: 'notANote',
  [Refusal.NOT_TEXT]: 'notText',
  [Refusal.TOO_LARGE]: 'tooLarge',
  [Refusal.BODY_REFUSED]: 'bodyRefused',
  [Refusal.UNREADABLE]: 'unreadable',
}
