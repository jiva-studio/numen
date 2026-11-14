/**
 * The list of vaults the window shows is the vaults of the installation with the one this window
 * has open.
 */
import { describe, expect, it } from 'vitest'
import type { CoreAnswers } from '@numen/core-client/testing'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createVaultListReader } from './getVaultList'

const createAnsweredReader = (answers: CoreAnswers) => {
  const api = createAnsweredApi(answers)
  return createVaultListReader({ vaults: api.vaults, getShownVault: api.shownVault.getShownVault })
}

describe('the vault list', () => {
  it('holds every vault and names the vault this window shows', async () => {
    const getVaultList = createAnsweredReader({
      vaultsService: {
        listVaults: () => ({
          vaults: [
            { id: 'physics', name: 'Physics', path: '/vaults/Physics', isMissing: false },
            { id: 'heat', name: 'Heat', path: '/vaults/Heat', isMissing: true },
          ],
        }),
      },
      windowService: { getCurrentVault: () => ({ vault: 'heat' }) },
    })

    expect(await getVaultList()).toStrictEqual({
      vaults: [
        { id: 'physics', name: 'Physics', path: '/vaults/Physics', isMissing: false },
        { id: 'heat', name: 'Heat', path: '/vaults/Heat', isMissing: true },
      ],
      showing: 'heat',
    })
  })

  it('is not answered where the core does not list the vaults', async () => {
    const getVaultList = createAnsweredReader({
      windowService: { getCurrentVault: () => ({ vault: 'heat' }) },
    })

    await expect(getVaultList()).rejects.toThrow()
  })
})
