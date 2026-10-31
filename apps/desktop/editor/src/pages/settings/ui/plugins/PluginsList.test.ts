import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import type { CatalogEntry } from '@numen/plugin-sdk'
import { CATALOG_STATE } from '@numen/plugin-sdk'
import { createMockCatalogEntry } from '@numen/plugin-sdk/testing'
import PluginsList from './PluginsList.vue'
import { createInstallation, createPluginInfo } from '@/testing/installation'
import type { Installation, SettingsTabState } from '../../types'
import { getSettingAt as at, type PluginInfo, type SettingEdit } from '@/entities/settings'
import { pick } from '@/testing/pick'

const createList = (
  file: Record<string, unknown> = {},
  plugins: readonly PluginInfo[] = [],
  catalog: readonly CatalogEntry[] = [],
) => {
  const recordedWrites: SettingEdit[] = []
  const installation: Installation = createInstallation({
    getSetting: (path) => at(file, path),
    write: (edits) => {
      recordedWrites.push(...edits)
    },
    plugins: ref(plugins),
    catalog: ref(catalog),
  })
  const state: SettingsTabState = { installation }
  const wrapper = mount(PluginsList, {
    props: { state, plugins },
  })
  return { wrapper, recordedWrites, installation }
}

const findRows = (wrapper: ReturnType<typeof createList>['wrapper']) =>
  wrapper.findAllComponents({ name: 'PluginRow' })

describe('PluginsList', () => {
  it('groups the installed plugins under the categories the catalog declares', () => {
    const plugins = [
      createPluginInfo('sample.tree', 'File Tree', { description: 'Tree view' }),
      createPluginInfo('sample.whisper', 'Whisper', { description: 'ASR engine' }),
    ]
    const { wrapper } = createList({}, plugins, [
      createMockCatalogEntry('sample.tree', { category: 'views' }),
      createMockCatalogEntry('sample.whisper', { category: 'engines' }),
    ])

    expect(findRows(wrapper)).toHaveLength(2)
    expect(wrapper.findAll('.plugin-row__category').map((one) => one.text())).toStrictEqual([
      'Engines /',
      'Views /',
    ])
    expect(wrapper.text()).toContain('System')
  })

  it('lists a plugin the catalog does not know under other', () => {
    const { wrapper } = createList({}, [createPluginInfo('md.numen.views.tree', 'File Tree')])

    expect(wrapper.find('.plugin-row__category').text()).toBe('Other /')
  })

  it('lists a plugin the catalog offers and the host does not have', () => {
    const { wrapper } = createList(
      {},
      [],
      [
        createMockCatalogEntry('sample.offer', {
          name: 'Offered Reader',
          state: CATALOG_STATE.AVAILABLE,
        }),
      ],
    )

    expect(findRows(wrapper)[0]?.props('plugin').name).toBe('Offered Reader')
    expect(wrapper.find('.plugin-row__badge--available').exists()).toBe(true)
  })

  it('renders empty state when plugins list is empty', () => {
    const { wrapper } = createList({}, [])

    expect(wrapper.find('.plugins-list__empty').text()).toBe('No plugins installed.')
    expect(findRows(wrapper)).toHaveLength(0)
  })

  it('filters plugins by feature and category from the catalog, not from the id', async () => {
    const plugins = [
      createPluginInfo('md.numen.notes', 'Notes', {
        readers: [{ id: 'markdown', extensions: ['md'] }],
      }),
      createPluginInfo('md.numen.views.file.tree', 'File Tree Explorer'),
    ]
    const { wrapper } = createList({ plugins: { 'md.numen.notes': { enabled: false } } }, plugins, [
      createMockCatalogEntry('md.numen.notes', { category: 'editors', feature: 'Markdown notes' }),
      createMockCatalogEntry('md.numen.views.file.tree', {
        category: 'views',
        feature: 'file browser',
      }),
    ])
    const searchInput = wrapper.find('input[type="search"]')

    await searchInput.setValue('disabled:true')
    expect(findRows(wrapper).map((row) => row.props('plugin').id)).toStrictEqual(['md.numen.notes'])

    await searchInput.setValue('reader:markdown')
    expect(findRows(wrapper).map((row) => row.props('plugin').id)).toStrictEqual(['md.numen.notes'])

    await searchInput.setValue('feature:browser')
    expect(findRows(wrapper).map((row) => row.props('plugin').id)).toStrictEqual([
      'md.numen.views.file.tree',
    ])

    await searchInput.setValue('feature:tree')
    expect(findRows(wrapper)).toHaveLength(0)

    await searchInput.setValue('category:editors')
    expect(findRows(wrapper).map((row) => row.props('plugin').id)).toStrictEqual(['md.numen.notes'])
  })

  it('passes enabled state from installation settings to plugin row', () => {
    const plugins = [createPluginInfo('plugin.disabled', 'Disabled Plugin')]
    const { wrapper } = createList({ plugins: { 'plugin.disabled': { enabled: false } } }, plugins)

    expect(findRows(wrapper)[0]?.props('isEnabled')).toBe(false)
  })

  it('emits selectPlugin with pluginId when a plugin row emits selectPlugin', () => {
    const plugins = [
      createPluginInfo('plugin.one', 'Plugin One'),
      createPluginInfo('plugin.two', 'Plugin Two'),
    ]
    const { wrapper } = createList({}, plugins)

    pick(findRows(wrapper), 1).vm.$emit('select-plugin')

    expect(wrapper.emitted('select-plugin')).toStrictEqual([['plugin.two']])
  })

  it('writes setting when plugin row emits toggle-enabled', () => {
    const plugins = [createPluginInfo('plugin.one', 'Plugin One')]
    const { wrapper, recordedWrites } = createList({}, plugins)

    pick(findRows(wrapper), 0).vm.$emit('toggle-enabled', false)

    expect(recordedWrites).toEqual([{ at: ['plugins', 'plugin.one', 'enabled'], value: 'false' }])
  })
})
