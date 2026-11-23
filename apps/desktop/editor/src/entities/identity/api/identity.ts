/**
 * The calls of the identity: reading it, making recovery words, creating it from them and deleting
 * it from this device; the link of a device; the devices of the account.
 */
import type { IdentityClient } from '@numen/core-client'
import type { Result } from '@numen/result'
import type { IdentityFailure, IdentityInfo, LinkMode, LinkRole } from '../lib/types'

type AccountClient = Pick<IdentityClient, 'get' | 'generateWords' | 'create' | 'startRestore' | 'delete'>
type LinkClient = Pick<
  IdentityClient,
  'startLink' | 'getLink' | 'decideLink' | 'enterLinkCode' | 'confirmJoin' | 'cancelLink'
>
type DevicesClient = Pick<IdentityClient, 'listDevices' | 'revokeDevice'>

/** The calls that keep the identity of this device. */
export const createAccountApi = (identity: AccountClient) => ({
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
})

/** The calls of a link of this device to an account. */
export const createLinkApi = (identity: LinkClient) => ({
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
})

/** The calls of the devices of the account. */
export const createDevicesApi = (identity: DevicesClient) => ({
  /** The devices of the account. */
  listDevices: () => identity.listDevices(),

  /** Removes a device from the account. */
  revokeDevice: (fingerprint: string) => identity.revokeDevice(fingerprint),
})

type AccountApi = ReturnType<typeof createAccountApi>
export type LinkApi = ReturnType<typeof createLinkApi>
export type DevicesApi = ReturnType<typeof createDevicesApi>

/** The calls of the identity of this device, in the three groups the store keeps apart. */
export interface IdentityApi {
  readonly account: AccountApi
  readonly link: LinkApi
  readonly devices: DevicesApi
}
