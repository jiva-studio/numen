/** An identity client of the core that refuses every call; a test overrides the calls it needs. */
import type { IdentityClient, IdentityFailure, LinkInfo } from '@numen/core-client'
import type { Result } from '@numen/result'
import {
  createAccountApi,
  createDevicesApi,
  createDeviceLinkApi,
  type DeviceInfo,
  type IdentityApi,
} from '@/entities/identity'

const asDone = (): Result<undefined, IdentityFailure> => ({ ok: true, value: undefined })
const asRefused = (): Result<never, IdentityFailure> => ({
  ok: false,
  error: { error: 'noLink', message: 'No link.' },
})

export const createIdentityClient = (overrides: Partial<IdentityClient> = {}): IdentityClient => ({
  get: () => Promise.resolve(undefined),
  generateWords: () => Promise.resolve([]),
  create: () => Promise.resolve(asRefused()),
  startRestore: () => Promise.resolve(asRefused()),
  delete: () => Promise.resolve(),
  startLink: () => Promise.resolve(asRefused()),
  getLink: () => Promise.resolve(asRefused()),
  decideLink: () => Promise.resolve(asDone()),
  enterLinkCode: () => Promise.resolve(asDone()),
  confirmJoin: () => Promise.resolve(asDone()),
  cancelLink: () => Promise.resolve(asDone()),
  listDevices: () => Promise.resolve({ ok: true, value: [] }),
  revokeDevice: () => Promise.resolve(asDone()),
  ...overrides,
})

/** A link in the phase and with the fields the test gives; the rest is empty. */
export const createLinkInfo = (overrides: Partial<LinkInfo> = {}): LinkInfo => ({
  phase: 'waitingForPeer',
  mode: 'show',
  role: 'host',
  qrText: 'numen://link?v=1&c=AAAA&k=BBBB&s=CCCC',
  code: '',
  peerName: '',
  peerFingerprint: '',
  accountFingerprint: '3F2A 91C0 0000 1111 2222 3333 4444 5555',
  triesLeft: 3,
  reason: undefined,
  deviceName: 'Laptop',
  deviceFingerprint: 'AAAA BBBB CCCC DDDD 0000 1111 2222 3333',
  isApproved: false,
  ...overrides,
})

export const createDeviceInfo = (overrides: Partial<DeviceInfo> = {}): DeviceInfo => ({
  name: 'Laptop',
  fingerprint: 'AAAA BBBB CCCC DDDD 0000 1111 2222 3333',
  addedAt: '2026-11-22T12:00:00Z',
  isThis: true,
  isRevoked: false,
  ...overrides,
})

/** The calls of the identity over a client of the core. */
export const wrapIdentityClient = (client: IdentityClient): IdentityApi => ({
  account: createAccountApi(client),
  link: createDeviceLinkApi(client),
  devices: createDevicesApi(client),
})
