import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PluginRow from './PluginRow.vue'
import { createPluginInfo } from '@/testing/installation'

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

  it('renders failure badge when plugin failure is present', () => {
    const plugin = createPluginInfo('test.engine', 'Speech Engine', {
      failure: {
        reason: 'Failed to start',
        detail: 'process exited with status 1',
        failedAt: 1_700_000_000_000,
      },
    })
    const wrapper = mount(PluginRow, {
      props: { plugin, isEnabled: false },
    })

    const badge = wrapper.find('.plugin-row__badge--stopped')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toBe('Stopped')
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
})
