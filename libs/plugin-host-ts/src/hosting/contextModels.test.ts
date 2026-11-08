/**
 * The model calls of a plugin: each plugin is handed models of its own, and the calls need no
 * permission, because the rows of its settings are the grant.
 */
import { describe, expect, it, vi } from 'vitest'
import type { ModelHost } from '@numen/plugin-sdk'
import { createPluginContext } from './context'
import type { PluginContextHost } from './types'

const createContext = (permissions: readonly string[]) => {
  const models = {
    complete: vi.fn(() => Promise.resolve({ ok: true, value: { text: 'fixed' } })),
  } as unknown as ModelHost
  const getModels = vi.fn(() => models)
  const host = { getModels } as unknown as PluginContextHost
  const context = createPluginContext('md.numen.tools.proofreading', host, {
    id: 'md.numen.tools.proofreading',
    name: 'Proofreading',
    version: '1.0.0',
    permissions,
  })
  return { context, models, getModels }
}

describe('the models of a plugin', () => {
  it('are made for the plugin that asks', () => {
    const { getModels } = createContext([])

    expect(getModels).toHaveBeenCalledWith('md.numen.tools.proofreading')
  })

  it('complete messages without any permission', async () => {
    const { context, models } = createContext([])

    await context.models!.complete({ messages: [] })

    expect(models.complete).toHaveBeenCalledWith({ messages: [] })
  })

  it('are absent where the host has none', () => {
    const context = createPluginContext('md.numen.a', {} as unknown as PluginContextHost, {
      id: 'md.numen.a',
      name: 'A',
      version: '1.0.0',
      permissions: [],
    })

    expect(context.models).toBeUndefined()
  })
})
