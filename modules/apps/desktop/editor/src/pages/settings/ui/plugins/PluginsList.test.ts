import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import PluginsList from './PluginsList.vue'
import PluginRow from './PluginRow.vue'
import { createInstallation, createPluginInfo } from '@/testing/installation'
import type { Installation, SettingsTabState } from '../../types'
import { getSettingAt as at, type PluginInfo, type SettingEdit } from '@/entities/settings'
import { pick } from '@/testing/pick'
import { WORDS as words } from '../../words'

const createList = (file: Record<string, unknown> = {}, plugins: readonly PluginInfo[] = []) => {
  const recordedWrites: SettingEdit[] = []
  const installation: Installation = createInstallation({
    getSetting: (path) => at(file, path),
    write: (edits) => {
      recordedWrites.push(...edits)
    },
    plugins: ref(plugins),
  })
  const state: SettingsTabState = { installation }
  const wrapper = mount(PluginsList, {
    props: { state, plugins },
  })
  return { wrapper, recordedWrites, installation }
}

describe('PluginsList', () => {
  it('renders installed plugins grouped under category headings', () => {
    const plugins = [
      createPluginInfo('md.numen.views.tree', 'File Tree', { description: 'Tree view' }),
      createPluginInfo('md.numen.engines.whisper', 'Whisper', { description: 'ASR engine' }),
    ]
    const { wrapper } = createList({}, plugins)

    const rows = wrapper.findAllComponents(PluginRow)
    expect(rows).toHaveLength(2)
    expect(wrapper.text()).toContain('File Tree')
    expect(wrapper.text()).toContain('Whisper')
    expect(wrapper.text()).toContain(words.categoryEngines)
    expect(wrapper.text()).toContain(words.categoryViews)
    expect(wrapper.text()).toContain(words.systemSettings)
  })

  it('renders empty state when plugins list is empty', () => {
    const { wrapper } = createList({}, [])

    const empty = wrapper.find('.plugins-list__empty')
    expect(empty.exists()).toBe(true)
    expect(empty.text()).toBe(words.noPlugins)
    expect(wrapper.findAllComponents(PluginRow)).toHaveLength(0)
  })

  it('filters plugins dynamically when search query is typed', async () => {
    const plugins = [
      createPluginInfo('md.numen.views.tree', 'File Tree', { description: 'Explorer' }),
      createPluginInfo('md.numen.engines.whisper', 'Whisper', {
        description: 'Voice transcription',
      }),
      createPluginInfo('md.numen.agents.chat', 'Agent Chat', {
        description: 'AI conversational assistant',
      }),
    ]
    const { wrapper } = createList({}, plugins)

    expect(wrapper.findAllComponents(PluginRow)).toHaveLength(3)

    const searchInput = wrapper.find('input[type="search"]')
    expect(searchInput.exists()).toBe(true)

    // Search by name
    await searchInput.setValue('whisper')
    let rows = wrapper.findAllComponents(PluginRow)
    expect(rows).toHaveLength(1)
    expect(rows[0]!.props('plugin').name).toBe('Whisper')

    // Search by description
    await searchInput.setValue('assistant')
    rows = wrapper.findAllComponents(PluginRow)
    expect(rows).toHaveLength(1)
    expect(rows[0]!.props('plugin').name).toBe('Agent Chat')

    // Search with no matches
    await searchInput.setValue('nonexistent')
    rows = wrapper.findAllComponents(PluginRow)
    expect(rows).toHaveLength(0)
    expect(wrapper.find('.plugins-list__empty').text()).toBe(words.noPluginsFound)

    // Clear search
    await searchInput.setValue('')
    rows = wrapper.findAllComponents(PluginRow)
    expect(rows).toHaveLength(3)
  })

  it('passes enabled state from installation settings to plugin row', () => {
    const plugins = [createPluginInfo('plugin.disabled', 'Disabled Plugin')]
    const { wrapper } = createList({ plugins: { 'plugin.disabled': { enabled: false } } }, plugins)

    const row = wrapper.findComponent(PluginRow)
    expect(row.exists()).toBe(true)
    expect(row.props('isEnabled')).toBe(false)
  })

  it('emits selectPlugin with pluginId when a plugin row emits selectPlugin', () => {
    const plugins = [
      createPluginInfo('plugin.one', 'Plugin One'),
      createPluginInfo('plugin.two', 'Plugin Two'),
    ]
    const { wrapper } = createList({}, plugins)

    const rows = wrapper.findAllComponents(PluginRow)
    pick(rows, 1).vm.$emit('select-plugin')

    expect(wrapper.emitted('select-plugin')).toStrictEqual([['plugin.two']])
  })

  it('writes setting when plugin row emits toggle-enabled', () => {
    const plugins = [createPluginInfo('plugin.one', 'Plugin One')]
    const { wrapper, recordedWrites } = createList({}, plugins)

    const row = wrapper.findComponent(PluginRow)
    row.vm.$emit('toggle-enabled', false)

    expect(recordedWrites).toEqual([{ at: ['plugins', 'plugin.one', 'enabled'], value: 'false' }])
  })
})
