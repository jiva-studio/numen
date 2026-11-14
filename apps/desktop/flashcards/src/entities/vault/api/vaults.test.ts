import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createCoreTransport } from '@numen/core-client/testing'
import { createWindowClients } from '@/shared/api/clients'
import { createVaultLister } from './vaults'

const core = { listVaults: vi.fn() }

const listVaults = createVaultLister(
  createWindowClients(createCoreTransport({ vaultsService: core })).vaultsService,
)

beforeEach(() => {
  vi.resetAllMocks()
})

describe('the vaults of the installation', () => {
  it('are listed with their identity, name and folder', async () => {
    core.listVaults.mockResolvedValue({
      vaults: [{ id: 'roots', name: 'Roots', path: '/vaults/Roots', isMissing: true }],
    })

    expect(await listVaults()).toStrictEqual([
      { id: 'roots', name: 'Roots', path: '/vaults/Roots' },
    ])
  })
})
