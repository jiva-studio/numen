import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { installCoreAnswers } from '@numen/core-client/testing'
import { listPlugins } from './plugins'

const core = { getPlugins: vi.fn() }

let restore: () => void

beforeEach(() => {
  vi.resetAllMocks()
  restore = installCoreAnswers({ pluginService: core })
})

afterEach(() => {
  restore()
})

describe('the installed plugins', () => {
  it('are listed with their identity and name', async () => {
    core.getPlugins.mockResolvedValue({
      plugins: [{ id: 'org.example.themes', name: 'Sample Themes', version: '1.0.0' }],
    })

    expect(await listPlugins()).toStrictEqual([{ id: 'org.example.themes', name: 'Sample Themes' }])
  })
})
