import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PluginsList from './PluginsList.vue'
import type { PluginHealth } from './pluginInfo'
import PluginRow from './PluginRow.vue'
import { createManagedPlugin, createAvailablePlugin } from './testing/managed'

const plugins = [
  createManagedPlugin(
    'sample.notes',
    'Notes',
    { readers: [{ id: 'markdown', extensions: ['md'] }] },
    {
      category: 'editors',
      feature: 'Markdown notes',
      tabKinds: ['note'],
    },
  ),
  createManagedPlugin('sample.tree', 'File Tree', {}, { category: 'views', feature: 'file tree' }),
  createManagedPlugin('sample.agent', 'Agent Chat', {}, { category: 'views', feature: 'agent chat' }),
  createAvailablePlugin('sample.offer', 'Offered Reader'),
]

const createHealth = (
  key: string,
  severity: PluginHealth['severity'],
  text: string,
): PluginHealth => ({ key, severity, text, detail: '', actionLabel: '' })

const mountList = (props: Record<string, unknown> = {}) =>
  mount(PluginsList, { props: { plugins, enabled: {}, ...props } })

const listNames = (wrapper: ReturnType<typeof mountList>) =>
  wrapper.findAllComponents(PluginRow).map((row) => row.props('plugin').name)

describe('PluginsList', () => {
  it('lists the plugins by the label of their category, then by name', () => {
    const wrapper = mountList()

    expect(listNames(wrapper)).toStrictEqual(['Notes', 'Agent Chat', 'File Tree', 'Offered Reader'])
  })

  it('shows the empty state when there are no plugins', () => {
    const wrapper = mountList({ plugins: [] })

    expect(wrapper.text()).toContain('No plugins installed.')
  })

  it('filters by what the query names and says when nothing matches', async () => {
    const wrapper = mountList()

    await wrapper.find('input[type="search"]').setValue('agent')
    expect(listNames(wrapper)).toStrictEqual(['Agent Chat'])

    await wrapper.find('input[type="search"]').setValue('zzz')
    expect(wrapper.text()).toContain('No plugins match “zzz”')
  })

  it('filters by feature from the manifest, and by category', async () => {
    const wrapper = mountList()

    await wrapper.find('input[type="search"]').setValue('feature:markdown')
    expect(listNames(wrapper)).toStrictEqual(['Notes'])

    await wrapper.find('input[type="search"]').setValue('category:views')
    expect(listNames(wrapper)).toStrictEqual(['Agent Chat', 'File Tree'])
  })

  it('filters the plugins switched off and the plugins not installed', async () => {
    const wrapper = mountList({ enabled: { 'sample.tree': false } })

    await wrapper.find('input[type="search"]').setValue('@disabled')
    expect(listNames(wrapper)).toStrictEqual(['File Tree'])

    await wrapper.find('input[type="search"]').setValue('@available')
    expect(listNames(wrapper)).toStrictEqual(['Offered Reader'])
  })

  it('passes the enabled state to each row, on unless the map says off', () => {
    const wrapper = mountList({ enabled: { 'sample.tree': false } })

    const states = Object.fromEntries(
      wrapper
        .findAllComponents(PluginRow)
        .map((row) => [row.props('plugin').id, row.props('isEnabled')]),
    )
    expect(states).toMatchObject({ 'sample.tree': false, 'sample.notes': true })
  })

  it('takes and gives the query through v-model', async () => {
    const wrapper = mountList({ query: 'notes' })
    expect(listNames(wrapper)).toStrictEqual(['Notes'])

    await wrapper.find('input[type="search"]').setValue('tree')

    expect(wrapper.emitted('update:query')).toStrictEqual([['tree']])
  })

  it('emits select-plugin with the id of the row, and the page of a tile', async () => {
    const wrapper = mountList()

    wrapper.findAllComponents(PluginRow)[0]?.vm.$emit('select-plugin')
    await wrapper.get('button[aria-label="System"]').trigger('click')
    await wrapper.get('button[aria-label="Connections"]').trigger('click')

    expect(wrapper.emitted('select-plugin')).toStrictEqual([
      ['sample.notes'],
      ['system'],
      ['connections'],
    ])
  })

  it('keeps both tiles while the query matches no plugin', () => {
    const wrapper = mountList({ query: 'zzz' })

    expect(wrapper.find('button[aria-label="System"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="Connections"]').exists()).toBe(true)
  })

  it('draws a stopped plugin as a button named for it and described by the reason', async () => {
    const start = createHealth('start', 'error', 'Model missing')
    const wrapper = mountList({ health: { 'sample.tree': [start] } })

    const card = wrapper.get('button[aria-label="File Tree stopped."]')
    const reason = wrapper.get(`#${card.attributes('aria-describedby')}`)
    expect(reason.text()).toBe('Model missing')
    await card.trigger('click')

    expect(wrapper.emitted('select-plugin')).toStrictEqual([['sample.tree']])
  })

  it('draws no stopped card for a plugin whose health holds no failed start', () => {
    const quota = createHealth('quota', 'warning', 'Half used')
    const wrapper = mountList({ health: { 'sample.tree': [quota] } })

    expect(wrapper.find('button[aria-label="File Tree stopped."]').exists()).toBe(false)
  })

  it('clears the query with the clear button', async () => {
    const wrapper = mountList({ query: 'notes' })

    await wrapper.get('button[aria-label="Clear search"]').trigger('click')

    expect(wrapper.emitted('update:query')).toStrictEqual([['']])
  })

  it('emits toggle-enabled with the id and the new state', () => {
    const wrapper = mountList()

    wrapper.findAllComponents(PluginRow)[1]?.vm.$emit('toggle-enabled', false)

    expect(wrapper.emitted('toggle-enabled')).toStrictEqual([['sample.agent', false]])
  })
})
