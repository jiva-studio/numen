import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { VAULTS_ERROR_CODE } from '@numen/core-client'
import { installCoreHandlers } from '@numen/core-client/testing'
import { vaults } from './index'
import { WINDOW } from './clients'

const requestLog = {
  listVaults: vi.fn(),
  getCurrentVault: vi.fn(),
  chooseFolder: vi.fn(),
  addVault: vi.fn(),
  renameVault: vi.fn(),
  removeVault: vi.fn(),
  openVault: vi.fn(),
}

let restore: () => void

beforeEach(() => {
  restore = installCoreHandlers(requestLog)
})

afterEach(() => {
  restore()
})

describe('a vault requested for', () => {
  it('is listed with the vault this window shows', async () => {
    requestLog.listVaults.mockResolvedValue({ vaults: [] })
    requestLog.getCurrentVault.mockResolvedValue({ vault: 'v1' })
    await vaults.list()
    expect(requestLog.listVaults.mock.calls.at(-1)?.[0]).toMatchObject({})
    expect(requestLog.getCurrentVault.mock.calls.at(-1)?.[0]).toMatchObject({ window: WINDOW })
  })

  it('is chosen from a folder dialog that starts nowhere in particular', async () => {
    requestLog.chooseFolder.mockResolvedValue({ isChosen: false, path: '/ignored' })
    expect(await vaults.choose('Pick')).toBe('')
    expect(requestLog.chooseFolder.mock.calls.at(-1)?.[0]).toMatchObject({
      title: 'Pick',
      startingAt: '',
    })
  })

  it('is added, renamed, removed and opened by what the schema names', async () => {
    requestLog.addVault.mockResolvedValue({})
    await vaults.add('/v', 'Name')
    expect(requestLog.addVault.mock.calls.at(-1)?.[0]).toMatchObject({ path: '/v', name: 'Name' })

    requestLog.renameVault.mockResolvedValue({})
    await vaults.rename('v1', 'Other')
    expect(requestLog.renameVault.mock.calls.at(-1)?.[0]).toMatchObject({ id: 'v1', name: 'Other' })

    requestLog.removeVault.mockResolvedValue({})
    await vaults.remove('v1', true)
    expect(requestLog.removeVault.mock.calls.at(-1)?.[0]).toMatchObject({ id: 'v1', trash: true })

    requestLog.openVault.mockResolvedValue({})
    await vaults.open('v1')
    expect(requestLog.openVault.mock.calls.at(-1)?.[0]).toMatchObject({ id: 'v1' })
  })

  it('fails in the words of the error', async () => {
    requestLog.addVault.mockResolvedValue({ error: VAULTS_ERROR_CODE.NAME_TAKEN })
    expect(await vaults.add('/v', 'Name')).toEqual({ ok: false, error: 'nameTaken' })
  })
})

