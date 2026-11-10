/**
 * The vault entity's part of the vault mock: the list of vaults and the state of the open one.
 */
import { asValue } from '@numen/core-client'
import { requests } from '../requests'
import { mockVaultList, mockVaultState, pendingStream } from '../answers'
import type { VaultCore } from '@/app/vault/core'
import type { Vaults } from '@/entities/vault'

export const vaultsMock = {
  list: () => Promise.resolve(mockVaultList),
  choose: () => {
    requests.chose += 1
    return Promise.resolve('')
  },
  add: () => Promise.resolve(asValue(null)),
  rename: () => Promise.resolve(asValue(null)),
  remove: () => Promise.resolve(null),
  open: (id: string) => {
    requests.opened.push(id)
    return Promise.resolve(null)
  },
} satisfies Partial<Vaults>

export const vaultCoreMock = {
  vaults: () =>
    mockVaultState.listable
      ? Promise.resolve(mockVaultList)
      : Promise.reject(new Error('the vaults are not there')),
  state: () =>
    Promise.resolve({
      id: 'physics',
      name: 'Vault',
      path: '/vaults/Physics',
      scan: {
        isReady: mockVaultState.isReady,
        error: mockVaultState.error,
        unwatchedPath: '',
      },
      coverage: {
        chunkCount: 0n,
        embeddedCount: BigInt(mockVaultState.embedded),
        isEmbedding: false,
      },
    }),
  agentUnreachable: () => Promise.resolve(''),
  watchVaultChanges: pendingStream,
} satisfies Partial<VaultCore>
