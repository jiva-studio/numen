/**
 * What the core answers about the vaults and the open vault, from `mockVaultState` and
 * `mockVaultList`. The answers go through the router transport, so the window runs its real
 * calls against them.
 */
import { WIRE_ERROR_CODE } from '@numen/core-client'
import type { CoreAnswers } from '@numen/core-client/testing'
import { requests } from '../requests'
import { vaultChanges, mockVaultList, mockVaultState } from '../answers'

export const vaultAnswers = {
  vault: {
    getVaultState: () => ({
      id: 'physics',
      name: 'Vault',
      path: '/vaults/Physics',
      scan: { isReady: mockVaultState.isReady, error: mockVaultState.error, watchError: '' },
    }),
    watchVaultChanges: vaultChanges.stream,
  },
  folderDialog: {
    chooseFolder: () => {
      requests.chose += 1
      return { isChosen: false, path: '' }
    },
  },
  vaultsService: {
    listVaults: () => {
      if (!mockVaultState.listable) throw new Error('the vaults are not there')
      return { vaults: mockVaultList.vaults }
    },
    addVault: () => ({}),
    renameVault: () => ({}),
    removeVault: () => ({}),
    openVault: (request) => {
      requests.opened.push(request.id)
      requests.tabsWhenOpened.push(requests.openTabs.at(-1)?.tabs.length ?? 0)
      return mockVaultState.isOpenRefused ? { error: WIRE_ERROR_CODE.NOT_FOUND } : {}
    },
  },
} satisfies CoreAnswers
