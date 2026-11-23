/**
 * The calls of the identity: reading it, making recovery words, creating it from them, restoring it from them or from the
 * sheets of a recovery kit, making and checking a kit, and deleting it from this device.
 */
import type { IdentityClient } from '@numen/core-client'
import type { Result } from '@numen/result'
import type { IdentityFailure, IdentityInfo, KitProof, RecoveryKit } from '../lib/types'

/** The calls of the identity over `identity`. */
export const createIdentityApi = (identity: IdentityClient) => ({
  /** The identity of this device, or undefined when it has none. */
  getIdentity: async (): Promise<IdentityInfo | undefined> => {
    const found = await identity.get()
    return found === undefined ? undefined : { ...found }
  },

  /** Twelve new recovery words; the core keeps none of them. */
  generateWords: (): Promise<readonly string[]> => identity.generateWords(),

  /** Keeps an identity on this device for the twelve `words`. */
  createIdentity: (words: readonly string[]): Promise<Result<IdentityInfo, IdentityFailure>> =>
    identity.create(words),

  /** Starts to restore the account of the twelve `words`; the core completes it in the background. */
  startRestore: (words: readonly string[]): Promise<Result<undefined, IdentityFailure>> =>
    identity.startRestore(words),

  /** Removes the identity from this device. */
  deleteIdentity: (): Promise<void> => identity.delete(),

  /** A new kit of three sheets, once `proof` fits the account of this device. */
  createKit: (proof: KitProof): Promise<Result<RecoveryKit, IdentityFailure>> =>
    identity.createKit(proof),

  /** The account fingerprint when two `sheets` fit the account of this device. */
  checkKit: (sheets: readonly (readonly string[])[]): Promise<Result<string, IdentityFailure>> =>
    identity.checkKit(sheets),

  /** Starts to restore the account of two `sheets` of a kit; the core completes it in the background. */
  startRestoreFromKit: (
    sheets: readonly (readonly string[])[],
  ): Promise<Result<undefined, IdentityFailure>> => identity.startRestoreFromKit(sheets),
})

export type IdentityApi = ReturnType<typeof createIdentityApi>
