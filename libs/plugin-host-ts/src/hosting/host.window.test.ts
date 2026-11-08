/**
 * The vaults and the window a plugin reads from its context: the ones the window gives the host,
 * the same for every plugin, and none where the window gives none.
 */
import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import { createMockVaultsHost, createMockWindowHost } from '@numen/plugin-sdk/testing'
import { createPluginHost } from './host'
import type { PluginHostDeps } from './types'

function createDeps(extra: Partial<PluginHostDeps> = {}): PluginHostDeps {
  return {
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    ...extra,
  }
}

const BAR = createMockManifest('org.example.ledger', { name: 'Ledger', version: '1' })
const TIDE = createMockManifest('org.example.tide', { name: 'Tide', version: '1' })

describe('the vaults and the window of a plugin context', () => {
  it('are the ones the window gives, whatever the permissions of the plugin', () => {
    const vaults = createMockVaultsHost()
    const window = createMockWindowHost()
    const host = createPluginHost(createDeps({ getVaults: () => vaults, getWindow: () => window }))

    const context = host.createContext('org.example.ledger', BAR)

    expect(context.vaults).toBe(vaults)
    expect(context.window).toBe(window)
  })

  it('are the same for every plugin of the window', () => {
    const vaults = createMockVaultsHost()
    const window = createMockWindowHost()
    const host = createPluginHost(createDeps({ getVaults: () => vaults, getWindow: () => window }))

    const ledger = host.createContext('org.example.ledger', BAR)
    const tide = host.createContext('org.example.tide', TIDE)

    expect(tide.vaults).toBe(vaults)
    expect(ledger.vaults).toBe(vaults)
    expect(tide.window).toBe(window)
    expect(ledger.window).toBe(window)
  })

  it('are absent where the window gives none', () => {
    const host = createPluginHost(createDeps())

    const context = host.createContext('org.example.ledger', BAR)

    expect(context.vaults).toBeUndefined()
    expect(context.window).toBeUndefined()
  })
})
