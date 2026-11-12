import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { flushPromises } from '@vue/test-utils'
import type { PluginInfo } from '@numen/plugin-manager'
import {
  createLostConnection,
  createMockCatalogEntry,
  installCoreAnswers,
} from '@numen/core-client/testing'
import { useCatalogEntries } from './catalogEntries'

let restore: (() => void) | undefined

afterEach(() => {
  restore?.()
  restore = undefined
})

function createPlugin(id: string, isActive: boolean): PluginInfo {
  return {
    id,
    name: id,
    version: '1.0.0',
    description: '',
    isActive,
    capabilities: [],
    hasFrontend: false,
    hasBackend: false,
    settings: [],
  }
}

describe('useCatalogEntries', () => {
  it('starts empty and refreshes only when plugin ids or active states change', async () => {
    const listCatalog = vi.fn(() => ({
      entries: [createMockCatalogEntry('sample.notes')],
      sourceErrors: [{ source: 'community', message: 'timed out' }],
    }))
    restore = installCoreAnswers({ pluginCatalog: { listCatalog } })
    const plugins = ref<readonly PluginInfo[]>([])
    const errors: unknown[] = []
    const entries = useCatalogEntries(plugins, (err) => errors.push(err))

    expect(entries.value).toStrictEqual([])
    expect(listCatalog).not.toHaveBeenCalled()

    plugins.value = [createPlugin('a', true), createPlugin('b', false)]
    await nextTick()
    await vi.waitFor(() =>
      expect(entries.value.map((one) => one.id)).toStrictEqual(['sample.notes']),
    )
    expect(listCatalog).toHaveBeenCalledTimes(1)
    expect(errors).toHaveLength(1)
    expect((errors[0] as Error).message).toBe('The plugin source community failed: timed out')

    // Same ids and active states do not trigger a refresh
    plugins.value = [createPlugin('a', true), createPlugin('b', false)]
    await nextTick()
    await flushPromises()
    expect(listCatalog).toHaveBeenCalledTimes(1)

    // Boundary between plugins matters (joining with newline vs empty string)
    plugins.value = [createPlugin('a:trueb', false)]
    await nextTick()
    await vi.waitFor(() => expect(listCatalog).toHaveBeenCalledTimes(2))
  })

  it('reports why the catalog is unavailable', async () => {
    restore = installCoreAnswers({
      pluginCatalog: {
        listCatalog: () => {
          throw createLostConnection('offline')
        },
      },
    })
    const plugins = ref<readonly PluginInfo[]>([])
    const errors: unknown[] = []
    useCatalogEntries(plugins, (err) => errors.push(err))

    plugins.value = [createPlugin('a', false)]
    await nextTick()
    await vi.waitFor(() => expect(errors).toHaveLength(1))

    expect((errors[0] as Error).message).toBe(
      'The plugin catalog is unavailable: Numen is not responding; retrying',
    )
  })
})
