import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin from './index'

describe('editor-markdown plugin', () => {
  it('registers reader for markdown and text extensions on activation', async () => {
    const ctx = createMockPluginContext({ pluginId: 'editor-markdown' })
    await plugin.activate(ctx)

    const readers = ctx.getRegisteredReaders()
    expect(readers).toHaveLength(1)

    const reader = readers[0]
    expect(reader?.id).toBe('editor-markdown')
    expect(reader?.fileExtensions).toEqual(['.md', '.txt', '.markdown'])
    expect(reader?.component).toBeDefined()
  })
})
