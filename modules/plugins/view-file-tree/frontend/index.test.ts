import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin, { activate, FileTreeView } from './index'

describe('view-file-tree plugin', () => {
  it('registers file-tree tab kind upon activation', () => {
    const ctx = createMockPluginContext({ pluginId: 'view-file-tree' })
    activate(ctx)

    const tabKinds = ctx.getRegisteredTabKinds()
    expect(tabKinds).toHaveLength(1)
    expect(tabKinds[0]?.kind).toBe('file-tree')
    expect(tabKinds[0]?.component).toBe(FileTreeView)
    expect(tabKinds[0]?.title).toBe('File Tree Explorer')
  })

  it('subscribes to vault change and file select events', () => {
    const ctx = createMockPluginContext({ pluginId: 'view-file-tree' })
    activate(ctx)

    expect(ctx.subscriptions.length).toBeGreaterThanOrEqual(3)

    ctx.dispose()
    expect(ctx.subscriptions).toHaveLength(0)
  })

  it('activates correctly using default plugin lifecycle', async () => {
    const ctx = createMockPluginContext({ pluginId: 'view-file-tree' })
    await plugin.activate(ctx)

    expect(ctx.getRegisteredTabKinds()).toHaveLength(1)
    expect(ctx.getRegisteredTabKinds()[0]?.kind).toBe('file-tree')
  })
})
