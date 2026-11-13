import type { JsonObject } from '@/shared/api/json'
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import PluginsList from './PluginsList.vue'
import {
  createCatalogEntry,
  createInstallation,
  createPluginInfo,
} from '@/pages/settings/testing/installation'
import type { CatalogEntry } from '@/entities/plugin'
import type { Installation } from '../../types'
import { getSettingAt as at } from '@/pages/settings/testing/settingAt'
import type { PluginInfo, SettingEdit } from '@/entities/settings'
import { pick } from '@/testing/pick'

const createList = (
  file: JsonObject = {},
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
  const wrapper = mount(PluginsList, {
    props: {
      settings: installation.settings,
      catalog: installation.plugin.catalog.value,
      query: '',
      plugins,
    },
  })
  return { wrapper, recordedWrites, installation }
}

const findRows = (wrapper: ReturnType<typeof createList>['wrapper']) =>
  wrapper.findAllComponents({ name: 'PluginRow' })

describe('PluginsList', () => {
  it('orders the installed plugins by the categories the catalog declares', () => {
    const plugins = [
      createPluginInfo('sample.tree', 'File Tree', { description: 'Tree view' }),
      createPluginInfo('sample.whisper', 'Whisper', { description: 'ASR engine' }),
    ]
    const { wrapper } = createList({}, plugins, [
      createCatalogEntry('sample.tree', { category: 'views' }),
      createCatalogEntry('sample.whisper', { category: 'engines' }),
    ])

    expect(findRows(wrapper).map((row) => row.props('plugin').category)).toStrictEqual([
      'engines',
      'views',
    ])
    expect(wrapper.text()).toContain('System')
  })

  it('lists a plugin the catalog does not know under other', () => {
    const { wrapper } = createList({}, [createPluginInfo('md.numen.views.tree', 'File Tree')])

    expect(findRows(wrapper)[0]?.props('plugin').category).toBe('')
  })

  it('lists a plugin the catalog offers and the host does not have', () => {
    const { wrapper } = createList(
      {},
      [],
      [
        createCatalogEntry('sample.offer', {
          name: 'Offered Reader',
          isAvailable: true,
        }),
      ],
    )

    expect(findRows(wrapper)[0]?.props('plugin').name).toBe('Offered Reader')
    expect(wrapper.get('button[aria-label="Offered Reader, Not installed"]').text()).toContain(
      'Not installed',
    )
  })

  it('renders empty state when plugins list is empty', () => {
    const { wrapper } = createList({}, [])

    expect(wrapper.text()).toContain('No plugins installed.')
    expect(findRows(wrapper)).toHaveLength(0)
  })

  it('filters plugins by feature and category from the catalog, not from the id', async () => {
    const plugins = [
      createPluginInfo('md.numen.notes', 'Notes', {
        readers: [{ id: 'markdown', extensions: ['md'] }],
      }),
      createPluginInfo('md.numen.views.file.tree', 'File Tree Explorer'),
    ]
    const { wrapper } = createList({ 'md.numen.notes': { enabled: false } }, plugins, [
      createCatalogEntry('md.numen.notes', { category: 'editors', feature: 'Markdown notes' }),
      createCatalogEntry('md.numen.views.file.tree', {
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
    const { wrapper } = createList({ 'plugin.disabled': { enabled: false } }, plugins)

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

    expect(recordedWrites).toEqual([{ at: ['plugin.one', 'enabled'], value: 'false' }])
  })
})
