/**
 * The calls of the identity: reading it, making recovery words, creating it from them and deleting
 * it from this device.
 */
import type { IdentityClient, LinkMode, LinkRole } from '@numen/core-client'
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

  /** Starts to restore the account of the twelve `words`; the core completes it in the background. */
  startRestore: (words: readonly string[]): Promise<Result<undefined, IdentityFailure>> =>
    identity.startRestore(words),

  /** Removes the identity from this device. */
  deleteIdentity: (): Promise<void> => identity.delete(),

  /** Starts a link of this device: the host that has the account, or the joiner that has none. */
  startLink: (role: LinkRole, mode: LinkMode, scannedText: string) => identity.startLink(role, mode, scannedText),

  /** The link in progress. */
  getLink: () => identity.getLink(),

  /** The answer of the person on the host to the new device. */
  decideLink: (approve: boolean) => identity.decideLink(approve),

  /** The confirmation code the person read on the host. */
  enterLinkCode: (code: string) => identity.enterLinkCode(code),

  /** The answer of the person on the new device to the account it is offered. */
  confirmJoin: (approve: boolean) => identity.confirmJoin(approve),

  /** Ends the link in progress. */
  cancelLink: () => identity.cancelLink(),

  /** The devices of the account. */
  listDevices: () => identity.listDevices(),

  /** Removes a device from the account. */
  revokeDevice: (fingerprint: string) => identity.revokeDevice(fingerprint),
})

export type IdentityApi = ReturnType<typeof createIdentityApi>
