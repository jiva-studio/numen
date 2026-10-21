import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin, { DocumentTab } from './index'

describe('reader-pdf plugin frontend', () => {
  it('activates and updates plugin active state', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.pdf' })
    await plugin.activate(ctx)

    const readers = ctx.getRegisteredReaders()
    expect(readers).toHaveLength(1)

    const reader = readers[0]
    expect(reader?.id).toBe('reader-pdf')
    expect(reader?.fileExtensions).toEqual(['.pdf'])
    expect(reader?.component).toBe(DocumentTab)
    expect(reader?.source).toBe('book')
  })

  it('runs deactivation cleanly', async () => {
    if (plugin.deactivate) {
      await plugin.deactivate()
    }
  })
})
