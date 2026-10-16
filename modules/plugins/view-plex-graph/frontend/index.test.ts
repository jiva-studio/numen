import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin, { activate, PlexGraphView } from './index'

describe('view-plex-graph plugin', () => {
  it('registers plex-graph tab kind upon activation', () => {
    const ctx = createMockPluginContext({ pluginId: 'view-plex-graph' })
    activate(ctx)

    const tabKinds = ctx.getRegisteredTabKinds()
    expect(tabKinds).toHaveLength(1)
    expect(tabKinds[0]?.kind).toBe('plex-graph')
    expect(tabKinds[0]?.component).toBe(PlexGraphView)
    expect(tabKinds[0]?.title).toBe('Knowledge Graph')
  })

  it('subscribes to vault change and file select events', () => {
    const ctx = createMockPluginContext({ pluginId: 'view-plex-graph' })
    activate(ctx)

    expect(ctx.subscriptions.length).toBeGreaterThanOrEqual(2)

    ctx.dispose()
    expect(ctx.subscriptions).toHaveLength(0)
  })

  it('activates and deactivates correctly using default plugin lifecycle', async () => {
    const ctx = createMockPluginContext({ pluginId: 'view-plex-graph' })
    await plugin.activate(ctx)

    expect(ctx.getRegisteredTabKinds()).toHaveLength(1)
    expect(ctx.getRegisteredTabKinds()[0]?.kind).toBe('plex-graph')

    await plugin.deactivate?.()
  })
})
