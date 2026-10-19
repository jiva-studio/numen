import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin, { activate, PlexGraphView, PLEX } from './index'
import { createHost } from './fixtures/host'
import { settle } from './fixtures'
import type { PlexTabState } from './types'

const createContext = () => {
  const { host, asked } = createHost({ opening: 'Root.md' })
  const ctx = createMockPluginContext({
    pluginId: 'md.numen.view-plex-graph',
    workspace: host.workspace,
    notes: host.notes,
  })
  return { ctx, asked }
}

describe('view-plex-graph plugin', () => {
  it('registers the plex tab kind upon activation', () => {
    const { ctx } = createContext()
    activate(ctx)

    const tabKinds = ctx.getRegisteredTabKinds()
    expect(tabKinds).toHaveLength(1)
    expect(tabKinds[0]?.kind).toBe(PLEX)
    expect(tabKinds[0]?.component).toBe(PlexGraphView)
    expect(tabKinds[0]?.open).toBeTypeOf('function')
  })

  it('holds two subscriptions, the tab kind and the vault, and lets go of both', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(ctx.subscriptions).toHaveLength(2)

    ctx.dispose()
    expect(ctx.subscriptions).toHaveLength(0)
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
  })

  it('asks every plex for its picture again when the vault changes', async () => {
    const { ctx, asked } = createContext()
    activate(ctx)
    const kind = ctx.getRegisteredTabKinds()[0]!
    kind.open!('One.md', 'plex:1') as PlexTabState
    await settle()
    asked.headings.length = 0

    ctx.emitVaultChange({ paths: ['One.md'] })
    await settle()
    await settle()

    expect(asked.headings.length).toBeGreaterThan(0)
  })

  it('refuses to activate in a window that offers no workspace or notes', () => {
    expect(() =>
      activate(createMockPluginContext({ pluginId: 'md.numen.view-plex-graph' })),
    ).toThrow('a plex needs the window it is drawn in and the notes of the vault')
  })

  it('activates through the default plugin lifecycle', async () => {
    const { ctx } = createContext()
    await plugin.activate(ctx)

    expect(ctx.getRegisteredTabKinds()[0]?.kind).toBe(PLEX)
  })
})
