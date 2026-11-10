/**
 * The settings, session and vault calls of the window core, over the real clients and a router in
 * place of the core. The module passes a message through unchanged, and the schema defines its
 * shape.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { installCoreHandlers } from '@numen/core-client/testing'
import { core, vaults } from './index'

const request = {
  getSettings: vi.fn(),
  writeSettings: vi.fn(),
  getVaultState: vi.fn(),
  getAgentState: vi.fn(),
  writeOpenTabs: vi.fn(),
  reportFlush: vi.fn(),
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
  restore = installCoreHandlers(request)
})

afterEach(() => {
  restore()
})

describe('settings domain', () => {
  it('reads and writes the settings', async () => {
    const writtenSettings = JSON.stringify({ app: { naming: { syncTitleAndFilename: true } } })
    request.getSettings.mockResolvedValue({
      written: writtenSettings,
      models: [],
      plugins: [],
      app: [],
      secretSections: [],
      secretMenus: [],
      credentialsPath: '',
      path: 'settings.json',
    })
    request.writeSettings.mockResolvedValue({})

    const config = await core.getSettings()
    expect(config.path).toBe('settings.json')
    await core.updateSettings([])
  })
})

describe('session domain', () => {
  it('handles state, agentUnreachable, writeOpenTabs, and reportFlush', async () => {
    request.getVaultState.mockResolvedValue({
      id: 'v1',
      name: 'V1',
      path: '/vault',
      scan: { isReady: true, error: '', unwatched: '' },
      coverage: { chunkCount: 10n, embeddedCount: 10n, isEmbedding: false },
    })
    const st = await core.state()
    expect(st.name).toBe('V1')

    request.getAgentState.mockResolvedValue({ unreachable: 'offline' })
    expect(await core.agentUnreachable()).toBe('offline')

    request.writeOpenTabs.mockResolvedValue({})
    await core.writeOpenTabs({ tabs: [], activeId: '' })
    expect(request.writeOpenTabs).toHaveBeenCalled()

    request.reportFlush.mockResolvedValue({})
    await core.reportFlush('tok', 'written')
    expect(request.reportFlush).toHaveBeenCalled()
  })
})

describe('vaults domain', () => {
  it('handles list, choose, add, rename, remove, open, and core.vaults', async () => {
    request.listVaults.mockResolvedValue({ vaults: [{ id: 'v1', name: 'Vault 1' }] })
    request.getCurrentVault.mockResolvedValue({ vault: 'v1' })

    const list = await vaults.list()
    expect(list.vaults.length).toBe(1)
    expect(list.showing).toBe('v1')

    const fromCore = await core.vaults()
    expect(fromCore.showing).toBe('v1')

    request.chooseFolder.mockResolvedValue({ isChosen: true, path: '/v2' })
    expect(await vaults.choose('Pick')).toBe('/v2')

    request.addVault.mockResolvedValue({ vault: { id: 'v2', name: 'V2' } })
    const added = await vaults.add('/v2', 'V2')
    expect(added.ok ? added.value?.id : null).toBe('v2')

    request.renameVault.mockResolvedValue({ vault: { id: 'v2', name: 'V2 renamed' } })
    const renamedItems = await vaults.rename('v2', 'V2 renamed')
    expect(renamedItems.ok ? renamedItems.value?.name : null).toBe('V2 renamed')

    request.removeVault.mockResolvedValue({})
    expect(await vaults.remove('v2', false)).toBeNull()

    request.openVault.mockResolvedValue({})
    expect(await vaults.open('v1')).toBeNull()
  })
})
