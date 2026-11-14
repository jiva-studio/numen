/**
 * The list of vaults the window shows is the vaults of the installation with the one this window
 * has open.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { installCoreAnswers } from '@numen/core-client/testing'
import { getVaultList } from './vaultList'

let restore: () => void = () => undefined

afterEach(() => {
  restore()
})

describe('the vault list', () => {
  it('holds every vault and names the vault this window shows', async () => {
    restore = installCoreAnswers({
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

  it('asks the core for the vault of the editor window', async () => {
    const asked: string[] = []
    restore = installCoreAnswers({
      vaultsService: { listVaults: () => ({ vaults: [] }) },
      windowService: {
        getCurrentVault: (request) => {
          asked.push(request.window)
          return { vault: 'heat' }
        },
      },
    })

    await getVaultList()

    expect(asked).toStrictEqual(['editor'])
  })

  it('is not answered where the core does not list the vaults', async () => {
    restore = installCoreAnswers({
      windowService: { getCurrentVault: () => ({ vault: 'heat' }) },
    })

    await expect(getVaultList()).rejects.toThrow()
  })
})
