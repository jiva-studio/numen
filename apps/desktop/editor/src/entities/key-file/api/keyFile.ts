/** The calls of the key file: reading how the keys are kept, choosing it and giving its passphrase. */
import type { KeyFileClient } from '@numen/core-client'

/** The calls that keep the keys of this computer in a file. */
export const createKeyFileApi = (client: KeyFileClient) => ({
  /** How the keys are kept now. */
  getState: () => client.getState(),

  /** Keeps the keys in the file the way `mode` says, with `passphrase` for the passphrase mode. */
  choose: (mode: 'plaintext' | 'passphrase', passphrase: string) =>
    client.choose(mode, passphrase),

  /** Gives the passphrase that opens the file for this run. */
  unlock: (passphrase: string) => client.unlock(passphrase),
})

export type KeyFileApi = ReturnType<typeof createKeyFileApi>
