import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin, { activate, MediaUrlTab } from './index'

describe('downloader frontend plugin', () => {
  it('registers url tab kind upon activation', () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    activate(ctx)

    const tabKinds = ctx.getRegisteredTabKinds()
    expect(tabKinds).toHaveLength(1)
    expect(tabKinds[0]?.kind).toBe('url')
    expect(tabKinds[0]?.component).toBe(MediaUrlTab)
    expect(tabKinds[0]?.title).toBe('Download Media')
  })

  it('registers downloader.open command that opens url tab', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    activate(ctx)

    const commands = ctx.getRegisteredCommands()
    const openCmd = commands.find((c) => c.id === 'downloader.open')
    expect(openCmd).toBeDefined()
    expect(openCmd?.title).toBe('Media: Download from URL')

    let openedKind = ''
    ctx.openTab = (kind: string) => {
      openedKind = kind
      return Promise.resolve(kind)
    }

    await ctx.triggerCommand('downloader.open')
    expect(openedKind).toBe('url')
  })

  it('activates using default plugin lifecycle', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    await plugin.activate(ctx)

    expect(ctx.getRegisteredTabKinds()).toHaveLength(1)
    expect(ctx.getRegisteredTabKinds()[0]?.kind).toBe('url')
  })
})
