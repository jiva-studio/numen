/**
 * The requests the window makes about the vaults of the installation, and how it words what the
 * core answers.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { VAULTS_ERROR_CODE } from '@numen/core-client'
import { installCoreHandlers } from '@numen/core-client/testing'
import { vaults } from './vaults'

const requestLog = {
  listVaults: vi.fn(),
  chooseFolder: vi.fn(),
  addVault: vi.fn(),
  renameVault: vi.fn(),
  removeVault: vi.fn(),
  openVault: vi.fn(),
}

let restore: () => void

beforeEach(() => {
  vi.clearAllMocks()
  restore = installCoreHandlers(requestLog)
})

afterEach(() => {
  restore()
})

describe('a vault requested for', () => {
  it('is listed as the vaults the installation holds', async () => {
    requestLog.listVaults.mockResolvedValue({
      vaults: [{ id: 'v1', name: 'Vault 1', path: '/v1', isMissing: true }],
    })

    expect(await vaults.list()).toStrictEqual([
      { id: 'v1', name: 'Vault 1', path: '/v1', isMissing: true },
    ])
  })

  it('is chosen from a folder dialog that starts nowhere in particular', async () => {
    requestLog.chooseFolder.mockResolvedValue({ isChosen: false, path: '/ignored' })
    expect(await vaults.choose('Pick')).toBe('')
    expect(requestLog.chooseFolder.mock.calls.at(-1)?.[0]).toMatchObject({
      title: 'Pick',
      startingAt: '',
    })

    requestLog.chooseFolder.mockResolvedValue({ isChosen: true, path: '/v2' })
    expect(await vaults.choose('Pick')).toBe('/v2')
  })

  it('is added and renamed, and answers the vault the core names', async () => {
    requestLog.addVault.mockResolvedValue({
      vault: { id: 'v2', name: 'V2', path: '/v2', isMissing: false },
    })
    const added = await vaults.add('/v2', 'V2')
    expect(requestLog.addVault.mock.calls.at(-1)?.[0]).toMatchObject({ path: '/v2', name: 'V2' })
    expect(added).toStrictEqual({
      ok: true,
      value: { id: 'v2', name: 'V2', path: '/v2', isMissing: false },
    })

    requestLog.renameVault.mockResolvedValue({
      vault: { id: 'v2', name: 'Other', path: '/v2', isMissing: false },
    })
    const answer = await vaults.rename('v2', 'Other')
    expect(requestLog.renameVault.mock.calls.at(-1)?.[0]).toMatchObject({ id: 'v2', name: 'Other' })
    expect(answer.ok ? answer.value?.name : null).toBe('Other')
  })

  it('leaves the list as it was where the core names no vault', async () => {
    requestLog.addVault.mockResolvedValue({})

    expect(await vaults.add('/v', 'Name')).toStrictEqual({ ok: true, value: null })
  })

  it('is removed and opened by what the schema names', async () => {
    requestLog.removeVault.mockResolvedValue({})
    expect(await vaults.remove('v1', true)).toBeNull()
    expect(requestLog.removeVault.mock.calls.at(-1)?.[0]).toMatchObject({ id: 'v1', trash: true })

    requestLog.openVault.mockResolvedValue({})
    expect(await vaults.open('v1')).toBeNull()
    expect(requestLog.openVault.mock.calls.at(-1)?.[0]).toMatchObject({ id: 'v1' })
  })

  it('fails in the words of the error', async () => {
    requestLog.addVault.mockResolvedValue({ error: VAULTS_ERROR_CODE.NAME_TAKEN })
    expect(await vaults.add('/v', 'Name')).toStrictEqual({ ok: false, error: 'nameTaken' })

    requestLog.removeVault.mockResolvedValue({ error: VAULTS_ERROR_CODE.LAST_VAULT })
    expect(await vaults.remove('v1', false)).toBe('lastVault')

    requestLog.openVault.mockResolvedValue({ error: VAULTS_ERROR_CODE.NOT_FOUND })
    expect(await vaults.open('gone')).toBe('notFound')
  })
})
