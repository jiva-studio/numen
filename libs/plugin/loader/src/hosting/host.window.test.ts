/**
 * The vaults and the window a plugin reads from its context: the ones the window gives the host,
 * the same for every plugin, and none where the window gives none.
 */
import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import { createMockVaultsHost, createMockWindowHost } from '@numen/plugin-sdk/testing'
import { createTestPluginHost, type TestHostDeps } from '../testing/hostDeps'

function createDeps(extra: Partial<TestHostDeps> = {}): TestHostDeps {
  return {
    target: 'desktop.editor',
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    ...extra,
  }
}

const BAR = createMockManifest('org.example.ledger', {
  name: 'Ledger',
  version: '1',
  permissions: ['vault:read'],
})
const TIDE = createMockManifest('org.example.tide', { name: 'Tide', version: '1' })

describe('the vaults and the window of a plugin context', () => {
  it('reach the ones the window gives, for the plugin that holds vault:read', () => {
    const vaults = createMockVaultsHost()
    const window = createMockWindowHost()
    const host = createTestPluginHost(
      createDeps({ getVaults: () => vaults, getWindow: () => window }),
    )

    const context = host.lifecycle.createContext('org.example.ledger', BAR)

    expect(context.host.vault.vaults?.getState()).toStrictEqual(vaults.getState())
    expect(context.host.ui.window?.getState()).toStrictEqual(window.getState())
  })

  it('are the same for every plugin of the window, the vaults only for one that holds vault:read', () => {
    const vaults = createMockVaultsHost()
    const window = createMockWindowHost()
    const host = createTestPluginHost(
      createDeps({ getVaults: () => vaults, getWindow: () => window }),
    )

    const ledger = host.lifecycle.createContext('org.example.ledger', BAR)
    const tide = host.lifecycle.createContext('org.example.tide', TIDE)

    expect(ledger.host.vault.vaults?.getState()).toStrictEqual(vaults.getState())
    expect(() => tide.host.vault.vaults?.getState()).toThrowError('vault:read')
    expect(tide.host.ui.window?.getState()).toStrictEqual(window.getState())
    expect(ledger.host.ui.window?.getState()).toStrictEqual(window.getState())
  })

  it('are absent where the window gives none', () => {
    const host = createTestPluginHost(createDeps())

    const context = host.lifecycle.createContext('org.example.ledger', BAR)

    expect(context.host.vault.vaults).toBeUndefined()
    expect(context.host.ui.window).toBeUndefined()
  })
})
