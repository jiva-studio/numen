/** The account identity of this device: its fingerprints, and the calls that keep or delete it. */
export { createIdentityStore, type IdentityStore } from './model/identity'
export { createAccountApi, createDevicesApi, createLinkApi, type IdentityApi } from './api/identity'
export type { DeviceInfo, IdentityInfo, LinkInfo, LinkMode, LinkPhase, LinkRole } from './lib/types'
