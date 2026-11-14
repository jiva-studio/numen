import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { Cpu, Puzzle } from '@lucide/vue'
import { Menu } from '@numen/ui'
import type { PluginHealth } from './pluginInfo'
import PluginRow from './PluginRow.vue'
import { createAvailablePlugin, createManagedPlugin } from './testing/managed'

const ENGINE = createManagedPlugin('test.engine', 'Speech Engine', {
  version: '2.1.0',
  description: 'Converts speech to text accurately.',
})

const health = (key: string, severity: PluginHealth['severity']): PluginHealth => ({
  key,
  severity,
  text: `The ${key} does not work`,
  detail: '',
  actionLabel: '',
})

const mountRow = (props: { isEnabled?: boolean; health?: readonly PluginHealth[] } = {}) =>
  mount(PluginRow, { props: { plugin: ENGINE, isEnabled: true, ...props } })

describe('PluginRow', () => {
  it('draws the name, the description and the category icon, and no version', () => {
    const wrapper = mountRow()
    const row = wrapper.get('button')

    expect(row.text()).toContain('Speech Engine')
    expect(row.text()).toContain('Converts speech to text accurately.')
    expect(row.text()).not.toContain('2.1.0')
    expect(wrapper.findComponent(Puzzle).exists()).toBe(true)

    const enginePlugin = createManagedPlugin(
      'test.engine',
      'Speech Engine',
      {},
      { category: 'engines' },
    )
    expect(
      mount(PluginRow, { props: { plugin: enginePlugin, isEnabled: true } })
        .findComponent(Cpu)
        .exists(),
    ).toBe(true)
  })

  it('emits select-plugin when the row is clicked', async () => {
    const wrapper = mountRow()

    await wrapper.get('button').trigger('click')

    expect(wrapper.emitted('select-plugin')).toHaveLength(1)
  })

  it('is named by the plugin alone while it runs', () => {
    expect(mountRow().get('button').attributes('aria-label')).toBe('Speech Engine')
  })

  it('carries Off in its name when the plugin is switched off, and draws no label', () => {
    const row = mountRow({ isEnabled: false }).get('button')

    expect(row.attributes('aria-label')).toBe('Speech Engine, Off')
    expect(row.text()).not.toContain('Off')
  })

  it('carries Stopped in its name for a plugin whose start failed, and draws no label', () => {
    const row = mountRow({ health: [health('start', 'error')] }).get('button')

    expect(row.attributes('aria-label')).toBe('Speech Engine, Stopped')
    expect(row.text()).not.toContain('Stopped')
  })

  it('labels the worst health item of a plugin that runs', () => {
    expect(mountRow({ health: [health('quota', 'warning')] }).text()).toContain('Warning')
    expect(
      mountRow({ health: [health('quota', 'warning'), health('connection', 'error')] }).text(),
    ).toContain('Error')
    expect(mountRow({ health: [] }).text()).not.toMatch(/Warning|Error/)
  })

  it('opens a menu on right click and emits toggle-enabled when Disable is chosen', async () => {
    const wrapper = mountRow()

    await wrapper.get('button').trigger('contextmenu', { clientX: 100, clientY: 200 })
    const menu = wrapper.findComponent(Menu)
    expect(menu.props('name')).toBe('Plugin actions')
    menu.vm.$emit('choose', 'toggle')
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('toggle-enabled')).toStrictEqual([[false]])
    expect(wrapper.findComponent(Menu).exists()).toBe(false)
  })

  it('labels a plugin that is not installed and offers only Settings in its menu', async () => {
    const wrapper = mount(PluginRow, {
      props: { plugin: createAvailablePlugin('test.offer', 'Offered Reader'), isEnabled: false },
    })

    expect(wrapper.text()).toContain('Not installed')
    await wrapper.get('button').trigger('contextmenu', { clientX: 1, clientY: 2 })

    expect(wrapper.findComponent(Menu).props('items')).toStrictEqual([
      { id: 'configure', text: 'Settings' },
    ])
  })
})
