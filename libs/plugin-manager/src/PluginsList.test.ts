import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PluginsList from './PluginsList.vue'
import PluginRow from './PluginRow.vue'
import { createManaged, createOffered } from './testing/managed'

const plugins = [
  createManaged(
    'sample.notes',
    'Notes',
    { readers: [{ id: 'markdown', extensions: ['md'] }] },
    {
      category: 'editors',
      feature: 'Markdown notes',
      tabKinds: ['note'],
    },
  ),
  createManaged('sample.tree', 'File Tree', {}, { category: 'views', feature: 'file tree' }),
  createManaged('sample.agent', 'Agent Chat', {}, { category: 'views', feature: 'agent chat' }),
  createOffered('sample.offer', 'Offered Reader'),
]

const mountList = (props: Record<string, unknown> = {}) =>
  mount(PluginsList, { props: { plugins, enabled: {}, ...props } })

const listNames = (wrapper: ReturnType<typeof mountList>) =>
  wrapper.findAllComponents(PluginRow).map((row) => row.props('plugin').name)

describe('PluginsList', () => {
  it('lists the plugins under the categories their manifests declare', () => {
    const wrapper = mountList()

    expect(wrapper.findAll('.plugin-row__category').map((one) => one.text())).toStrictEqual([
      'Editors /',
      'Views /',
      'Views /',
      'Other /',
    ])
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
    expect(wrapper.text()).toContain('No plugins found.')
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

  it('emits select-plugin with the id of the row, and system for the system row', async () => {
    const wrapper = mountList()

    wrapper.findAllComponents(PluginRow)[0]?.vm.$emit('select-plugin')
    await wrapper.find('.system-row').trigger('click')

    expect(wrapper.emitted('select-plugin')).toStrictEqual([['sample.notes'], ['system']])
  })

  it('emits toggle-enabled with the id and the new state', () => {
    const wrapper = mountList()

    wrapper.findAllComponents(PluginRow)[1]?.vm.$emit('toggle-enabled', false)

    expect(wrapper.emitted('toggle-enabled')).toStrictEqual([['sample.agent', false]])
  })
})
