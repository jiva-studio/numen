/**
 * What the core answers about the vaults and the open vault, from `mockVaultState` and
 * `mockVaultList`. The answers go through the router transport, so the window runs its real
 * calls against them.
 */
import type { CoreAnswers } from '@numen/core-client/testing'
import { requests } from '../requests'
import { mockVaultList, mockVaultState, pendingStream } from '../answers'

export const vaultAnswers = {
  vault: {
    getVaultState: () => ({
      id: 'physics',
      name: 'Vault',
      path: '/vaults/Physics',
      scan: { isReady: mockVaultState.isReady, error: mockVaultState.error, watchError: '' },
      coverage: {
        chunkCount: 0n,
        embeddedCount: BigInt(mockVaultState.embedded),
        isEmbedding: false,
      },
    }),
    watchVaultChanges: pendingStream,
  },
  agentService: { getAgentState: () => ({ unreachable: '' }) },
  vaultsService: {
    listVaults: () => {
      if (!mockVaultState.listable) throw new Error('the vaults are not there')
      return { vaults: mockVaultList.vaults }
    },
    chooseFolder: () => {
      requests.chose += 1
      return { isChosen: false, path: '' }
    },
    addVault: () => ({}),
    renameVault: () => ({}),
    removeVault: () => ({}),
    openVault: (request) => {
      requests.opened.push(request.id)
      return {}
    },
  },
} satisfies CoreAnswers
