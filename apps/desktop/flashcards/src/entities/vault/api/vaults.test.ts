import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { installCoreAnswers } from '@numen/core-client/testing'
import { listVaults } from './vaults'

const core = { listVaults: vi.fn() }

let restore: () => void

beforeEach(() => {
  vi.resetAllMocks()
  restore = installCoreAnswers({ vaultsService: core })
})

afterEach(() => {
  restore()
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
