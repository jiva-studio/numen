import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin, { DocumentTab } from './index'

describe('reader-pdf plugin frontend', () => {
  it('activates and updates plugin active state', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.readers.pdf' })
    await plugin.activate(ctx)

    const readers = ctx.getRegisteredReaders()
    expect(readers).toHaveLength(1)

    const reader = readers[0]
    expect(reader?.id).toBe('reader-pdf')
    expect(reader?.component).toBe(DocumentTab)
    expect(reader?.source).toBe('book')
  })

  it('declares the reader it registers as the default reader of PDF files', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.readers.pdf' })
    await plugin.activate(ctx)
    const manifest = JSON.parse(
      readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
    ) as {
      readers: { id: string; contentTypes: string[]; rank: string }[]
    }

    expect(manifest.readers).toEqual([
      { id: ctx.getRegisteredReaders()[0]?.id, contentTypes: ['application/pdf'], rank: 'default' },
    ])
  })

  it('runs deactivation cleanly', async () => {
    if (plugin.deactivate) {
      await plugin.deactivate()
    }
  })
})
