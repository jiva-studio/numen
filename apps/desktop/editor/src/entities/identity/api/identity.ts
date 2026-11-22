/**
 * The calls of the identity: reading it, making recovery words, creating it from them and deleting
 * it from this device.
 */
import type { IdentityClient } from '@numen/core-client'
import type { Result } from '@numen/result'
import type { IdentityFailure, IdentityInfo } from '../lib/types'

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

  /** Removes the identity from this device. */
  deleteIdentity: (): Promise<void> => identity.delete(),
})

export type IdentityApi = ReturnType<typeof createIdentityApi>
