import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createCoreTransport } from '@numen/core-client/testing'
import { createWindowClients } from '@/shared/api/clients'
import { createPluginLister } from './plugins'

const core = { getPlugins: vi.fn() }

const listPlugins = createPluginLister(
  createWindowClients(createCoreTransport({ pluginService: core })).pluginService,
)

beforeEach(() => {
  vi.resetAllMocks()
})

describe('the installed plugins', () => {
  it('are listed with their identity and name', async () => {
    core.getPlugins.mockResolvedValue({
      plugins: [{ id: 'org.example.themes', name: 'Sample Themes', version: '1.0.0' }],
    })

    expect(await listPlugins()).toStrictEqual([{ id: 'org.example.themes', name: 'Sample Themes' }])
  })
})
