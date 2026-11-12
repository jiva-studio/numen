import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PluginRow from './PluginRow.vue'
import { createManagedPlugin as createPluginInfo, createAvailablePlugin } from './testing/managed'

describe('PluginRow', () => {
  it('renders plugin name, version, and description', () => {
    const plugin = createPluginInfo('test.engine', 'Speech Engine', {
      version: '2.1.0',
      description: 'Converts speech to text accurately.',
    })
    const wrapper = mount(PluginRow, {
      props: { plugin, isEnabled: true },
    })

    expect(wrapper.text()).toContain('Speech Engine v2.1.0')
    expect(wrapper.text()).toContain('Converts speech to text accurately.')
  })

  it('renders plugin name without version suffix when version is empty', () => {
    const plugin = createPluginInfo('test.engine', 'Speech Engine', {
      version: '',
      description: 'Basic engine.',
    })
    const wrapper = mount(PluginRow, {
      props: { plugin, isEnabled: true },
    })

    expect(wrapper.text()).toContain('Speech Engine')
    expect(wrapper.text()).not.toContain('Speech Engine v')
  })

  it('emits selectPlugin when row is clicked', async () => {
    const plugin = createPluginInfo('test.engine', 'Speech Engine')
    const wrapper = mount(PluginRow, {
      props: { plugin, isEnabled: true },
    })

    await wrapper.find('.plugin-row').trigger('click')
    expect(wrapper.emitted('select-plugin')).toHaveLength(1)
  })

  it('emits selectPlugin on Enter keydown', async () => {
    const plugin = createPluginInfo('test.engine', 'Speech Engine')
    const wrapper = mount(PluginRow, {
      props: { plugin, isEnabled: true },
    })

    await wrapper.find('.plugin-row').trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('select-plugin')).toHaveLength(1)
  })

  it('shows Stopped for a plugin whose start failed', () => {
    const plugin = createPluginInfo('test.engine', 'Speech Engine')
    const wrapper = mount(PluginRow, {
      props: {
        plugin,
        isEnabled: false,
        health: [
          { key: 'start', severity: 'error', text: 'Failed to start', detail: '', actionLabel: '' },
        ],
      },
    })

    const badge = wrapper.find('.plugin-row__badge--stopped')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toBe('Stopped')
  })

  it('shows the badge of the worst health item of a plugin that runs', () => {
    const plugin = createPluginInfo('test.engine', 'Speech Engine')
    const quota = {
      key: 'quota',
      severity: 'warning',
      text: 'Half used',
      detail: '',
      actionLabel: '',
    } as const
    const connection = {
      ...quota,
      key: 'connection',
      severity: 'error',
      text: 'No connection',
    } as const

    const warned = mount(PluginRow, { props: { plugin, isEnabled: true, health: [quota] } })
    const failed = mount(PluginRow, {
      props: { plugin, isEnabled: true, health: [quota, connection] },
    })
    const healthy = mount(PluginRow, { props: { plugin, isEnabled: true, health: [] } })

    expect(warned.find('.plugin-row__badge').text()).toBe('Warning')
    expect(failed.find('.plugin-row__badge').text()).toBe('Error')
    expect(healthy.find('.plugin-row__badge').exists()).toBe(false)
  })

  it('applies disabled class and has no disabled badge when plugin is disabled', () => {
    const plugin = createPluginInfo('test.engine', 'Speech Engine')
    const wrapper = mount(PluginRow, {
      props: { plugin, isEnabled: false },
    })

    expect(wrapper.find('.plugin-row--disabled').exists()).toBe(true)
    expect(wrapper.find('.plugin-row__badge--disabled').exists()).toBe(false)
  })

  it('does not apply disabled class when plugin is enabled', () => {
    const plugin = createPluginInfo('test.engine', 'Speech Engine')
    const wrapper = mount(PluginRow, {
      props: { plugin, isEnabled: true },
    })

    expect(wrapper.find('.plugin-row--disabled').exists()).toBe(false)
  })

  it('renders category prefix when provided', () => {
    const plugin = createPluginInfo('test.engine', 'Speech Engine')
    const wrapper = mount(PluginRow, {
      props: { plugin, isEnabled: true, categoryPrefix: 'Engines' },
    })

    expect(wrapper.find('.plugin-row__category').text()).toBe('Engines /')
  })

  it('opens context menu on right click and emits toggle-enabled when chosen', async () => {
    const plugin = createPluginInfo('test.engine', 'Speech Engine')
    const wrapper = mount(PluginRow, {
      props: { plugin, isEnabled: true },
    })

    await wrapper.find('.plugin-row').trigger('contextmenu', { clientX: 100, clientY: 200 })
    const menu = wrapper.findComponent({ name: 'Menu' })
    expect(menu.exists()).toBe(true)

    menu.vm.$emit('choose', 'toggle')
    expect(wrapper.emitted('toggle-enabled')).toStrictEqual([[false]])
  })

  it('badges a plugin that is not installed and offers no switch', async () => {
    const wrapper = mount(PluginRow, {
      props: { plugin: createAvailablePlugin('test.offer', 'Offered Reader'), isEnabled: false },
    })

    expect(wrapper.find('.plugin-row__badge--available').text()).toBe('Not installed')

    await wrapper.find('.plugin-row').trigger('contextmenu', { clientX: 1, clientY: 2 })
    const menu = wrapper.findComponent({ name: 'Menu' })
    expect(menu.props('items')).toStrictEqual([{ id: 'configure', text: 'Settings' }])
  })
})
