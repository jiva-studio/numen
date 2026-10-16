import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin from './index'
import { isPdfPluginActive, setPdfPluginActive } from '@/pages/document-viewer/model/pluginState'

describe('reader-pdf plugin frontend', () => {
  it('registers reader for pdf extension on activation', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.pdf' })
    await plugin.activate(ctx)

    expect(isPdfPluginActive.value).toBe(true)

    const readers = ctx.getRegisteredReaders()
    expect(readers).toHaveLength(1)

    const reader = readers[0]
    expect(reader?.id).toBe('reader-pdf')
    expect(reader?.fileExtensions).toEqual(['.pdf'])
    expect(reader?.component).toBeDefined()
  })

  it('updates plugin active state on deactivation', async () => {
    setPdfPluginActive(true)
    if (plugin.deactivate) {
      await plugin.deactivate()
      expect(isPdfPluginActive.value).toBe(false)
    }
    // Restore
    setPdfPluginActive(true)
  })
})
