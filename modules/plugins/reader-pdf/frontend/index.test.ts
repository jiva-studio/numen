import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin from './index'
import { isPdfPluginActive, setPdfPluginActive } from '@/pages/document-viewer/model/pluginState'

describe('reader-pdf plugin frontend', () => {
  it('activates and updates plugin active state', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.pdf' })
    await plugin.activate(ctx)

    expect(isPdfPluginActive.value).toBe(true)
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
