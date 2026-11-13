import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { Button, KeyCap, Switch } from '@numen/ui'
import PluginDetail from './PluginDetail.vue'
import PluginData from './PluginData.vue'
import PluginNotice from './PluginNotice.vue'
import type { PluginContribution, PluginHealth } from './pluginInfo'
import type { PluginDetailProps } from './PluginDetail.types'
import { createAvailablePlugin, createManagedPlugin } from './testing/managed'

const CONTRIBUTIONS: readonly PluginContribution[] = [
  {
    kind: 'commands',
    items: [{ text: 'New note', detail: 'notes.create', keys: 'mod+shift+n' }],
  },
  { kind: 'permissions', items: [{ text: 'vault:read', detail: '', keys: '' }] },
]

const baseProps = (overrides: Partial<PluginDetailProps> = {}): PluginDetailProps => ({
  plugin: createManagedPlugin('sample.notes', 'Notes', {
    version: '2.1.0',
    description: 'Takes notes.',
    contributions: CONTRIBUTIONS,
  }),
  isEnabled: true,
  readme: '',
  dataFileCount: 0,
  problem: '',
  canInstall: false,
  hasSettings: false,
  ...overrides,
})

const SettingsStub = defineComponent({
  render: () => h('div', { class: 'settings-stub' }, 'fields'),
})

const mountDetail = (overrides: Partial<PluginDetailProps> = {}, withSettings = false) =>
  mount(PluginDetail, {
    props: baseProps(overrides),
    ...(withSettings ? { slots: { settings: SettingsStub } } : {}),
  })

const getTab = (wrapper: ReturnType<typeof mountDetail>, name: string) =>
  wrapper.findAll('[role="tab"]').find((one) => one.text() === name)

const openTab = async (wrapper: ReturnType<typeof mountDetail>, name: string) => {
  await getTab(wrapper, name)?.trigger('click')
}

const health = (key: string, severity: PluginHealth['severity'], detail = ''): PluginHealth => ({
  key,
  severity,
  text: `The ${key} does not work`,
  detail,
  actionLabel: key === 'connection' ? 'Open settings' : '',
})

describe('PluginDetail', () => {
  it('draws the name, the id, the version, the state and the description', () => {
    const wrapper = mountDetail()

    expect(wrapper.get('h2').text()).toBe('Notes')
    expect(wrapper.text()).toContain('sample.notes')
    expect(wrapper.text()).toContain('v2.1.0')
    expect(wrapper.text()).toContain('Running')
    expect(wrapper.text()).toContain('Takes notes.')
  })

  it('goes back from the back button and from the Settings segment of the path', async () => {
    const wrapper = mountDetail()

    await wrapper.get('button[aria-label="Back"]').trigger('click')
    await wrapper.get('nav button').trigger('click')

    expect(wrapper.emitted('back')).toHaveLength(2)
  })

  it('opens the category of the plugin from its segment of the path', async () => {
    const wrapper = mountDetail()

    await wrapper.findAll('nav button')[1]?.trigger('click')

    expect(wrapper.emitted('open-category')).toStrictEqual([['other']])
    expect(wrapper.emitted('back')).toBeUndefined()
  })

  it('shows the path from Settings through the category to the plugin, the plugin as the current page', () => {
    const nav = mountDetail().get('nav')

    expect(nav.attributes('aria-label')).toBe('Path')
    expect(nav.findAll('button').map((one) => one.text())).toStrictEqual(['Settings', 'Other'])
    expect(nav.get('[aria-current="page"]').text()).toBe('Notes')
  })

  it('puts the state and the enabled switch in the toolbar', () => {
    const toolbar = mountDetail().get('.settings-toolbar')

    expect(toolbar.text()).toContain('Running')
    expect(toolbar.find('[role="switch"]').exists()).toBe(true)
  })

  it('shows the enabled switch of an installed plugin and emits its change', () => {
    const wrapper = mountDetail({ isEnabled: false })
    const toggle = wrapper.getComponent(Switch)

    expect(toggle.props('modelValue')).toBe(false)
    expect(wrapper.get('[role="switch"]').attributes('aria-label')).toBe('Enabled Notes')
    toggle.vm.$emit('update:modelValue', true)

    expect(wrapper.emitted('toggle-enabled')).toStrictEqual([[true]])
  })

  it('opens on Settings, followed by About and Contributes, and no Data without data', () => {
    const tabs = mountDetail().findAll('[role="tab"]')

    expect(tabs.map((one) => one.text())).toStrictEqual(['Settings', 'About', 'Contributes'])
    expect(tabs[0]?.attributes('aria-selected')).toBe('true')
  })

  it('adds the Data tab while the plugin keeps files or their deletion failed', () => {
    expect(getTab(mountDetail({ dataFileCount: 2 }), 'Data')).toBeDefined()
    expect(getTab(mountDetail({ problem: 'read-only' }), 'Data')).toBeDefined()
  })

  it('ties each tab to its panel by id', async () => {
    const wrapper = mountDetail()
    await openTab(wrapper, 'About')

    const tab = getTab(wrapper, 'About')
    const panel = wrapper.get('[role="tabpanel"]')
    expect(tab?.attributes('aria-controls')).toBe(panel.attributes('id'))
    expect(panel.attributes('aria-labelledby')).toBe(tab?.attributes('id'))
  })

  it('moves between the tabs with the arrow keys, Home and End', async () => {
    const wrapper = mountDetail()
    const selected = () => wrapper.get('[role="tab"][aria-selected="true"]').text()

    await wrapper.findAll('[role="tab"]')[0]?.trigger('keydown', { key: 'ArrowRight' })
    expect(selected()).toBe('About')
    await wrapper.findAll('[role="tab"]')[1]?.trigger('keydown', { key: 'End' })
    expect(selected()).toBe('Contributes')
    await wrapper.findAll('[role="tab"]')[2]?.trigger('keydown', { key: 'Home' })
    expect(selected()).toBe('Settings')
    await wrapper.findAll('[role="tab"]')[0]?.trigger('keydown', { key: 'ArrowLeft' })
    expect(selected()).toBe('Contributes')
  })

  it('draws the README on About without its title and its comments', async () => {
    const wrapper = mountDetail({ readme: '# Notes\n\nSummary.\n\n<!-- more -->\n\nDetails.' })
    await openTab(wrapper, 'About')

    const panel = wrapper.get('[role="tabpanel"]')
    expect(panel.text()).toContain('Summary.')
    expect(panel.text()).toContain('Details.')
    expect(panel.find('h1').exists()).toBe(false)
    expect(panel.html()).not.toContain('more')
  })

  it('draws the description on About where the plugin has no README', async () => {
    const wrapper = mountDetail()
    await openTab(wrapper, 'About')

    expect(wrapper.get('[role="tabpanel"]').text()).toBe('Takes notes.')
  })

  it('lists the sections of the manifest on Contributes with labels and shortcuts', async () => {
    const wrapper = mountDetail({ isApple: true })
    await openTab(wrapper, 'Contributes')

    const panel = wrapper.get('[role="tabpanel"]')
    expect(panel.findAll('dt').map((one) => one.text())).toStrictEqual(['Commands', 'Permissions'])
    expect(panel.text()).toContain('notes.create')
    expect(panel.getComponent(KeyCap).props('keys')).toStrictEqual({
      icons: ['command', 'shift'],
      letter: 'N',
    })
  })

  it('says the plugin contributes nothing where its manifest declares nothing', async () => {
    const wrapper = mountDetail({
      plugin: createManagedPlugin('sample.empty', 'Empty', { contributions: [] }),
    })
    await openTab(wrapper, 'Contributes')

    expect(wrapper.get('[role="tabpanel"]').text()).toBe('Contributes nothing.')
  })

  it('draws the failure card of a suspended plugin with its detail behind a toggle', async () => {
    const wrapper = mountDetail({ health: [health('start', 'error', 'exec error')] })
    const card = wrapper.getComponent(PluginNotice)

    expect(card.text()).toContain('The start does not work')
    expect(card.text()).not.toContain('exec error')
    expect(wrapper.findAllComponents(PluginNotice)).toHaveLength(1)
    expect(wrapper.text()).toContain('Stopped')

    const [retry, details] = card.findAll('button')
    await details?.trigger('click')
    expect(card.text()).toContain('exec error')
    expect(details?.attributes('aria-expanded')).toBe('true')
    await retry?.trigger('click')

    expect(wrapper.emitted('retry')).toHaveLength(1)
  })

  it('offers no details toggle for a failure without a detail', () => {
    const card = mountDetail({ health: [health('start', 'error')] }).getComponent(PluginNotice)

    expect(card.findAll('button').map((one) => one.text())).toStrictEqual(['Retry'])
  })

  it('draws the notice of the worst item and emits its fix', async () => {
    const wrapper = mountDetail({
      health: [health('quota', 'warning'), health('connection', 'error', 'No connection to ask.')],
    })
    const bar = wrapper.getComponent(PluginNotice)

    expect(bar.attributes('role')).toBe('alert')
    expect(bar.text()).toContain('The connection does not work')
    expect(bar.text()).not.toContain('The quota does not work')
    await bar.get('button').trigger('click')

    expect(wrapper.emitted('run-action')).toStrictEqual([['connection']])
  })

  it('draws the settings slot only where the application says it has settings', () => {
    expect(mountDetail({}, true).find('.settings-stub').exists()).toBe(false)
    expect(mountDetail({}, true).text()).toContain('This plugin has no settings.')
    expect(mountDetail({ hasSettings: true }, true).find('.settings-stub').exists()).toBe(true)
  })
})

describe('PluginDetail for a plugin that is not installed', () => {
  it('offers install where the application can install', async () => {
    const wrapper = mountDetail({
      plugin: createAvailablePlugin('sample.offer', 'Offered'),
      canInstall: true,
    })

    expect(wrapper.findComponent(Switch).exists()).toBe(false)
    expect(wrapper.text()).toContain('Not installed')
    await wrapper.getComponent(Button).trigger('click')

    expect(wrapper.emitted('install')).toHaveLength(1)
  })

  it('offers neither switch nor install where the application cannot install', () => {
    const wrapper = mountDetail({ plugin: createAvailablePlugin('sample.offer', 'Offered') })

    expect(wrapper.findComponent(Switch).exists()).toBe(false)
    expect(wrapper.findComponent(Button).exists()).toBe(false)
  })
})

describe('PluginData', () => {
  const mountData = (fileCount: number, problem = '') =>
    mount(PluginData, {
      props: { pluginId: 'sample.notes', pluginName: 'Notes', fileCount, problem },
    })

  it('offers no delete where the folder holds no files', () => {
    expect(mountData(0).find('button').exists()).toBe(false)
  })

  it('names the files and deletes only after the person confirms', async () => {
    const wrapper = mountData(3)
    expect(wrapper.text()).toContain('3 files in the plugin')

    await wrapper.get('button').trigger('click')
    expect(wrapper.get('[role="alert"]').text()).toContain('Delete 3 files of Notes?')
    expect(wrapper.emitted('delete-data')).toBeUndefined()

    await wrapper.findAll('[role="alert"] button')[0]?.trigger('click')

    expect(wrapper.emitted('delete-data')).toHaveLength(1)
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('keeps the data when the person keeps it', async () => {
    const wrapper = mountData(1)
    expect(wrapper.text()).toContain('1 file in the plugin')

    await wrapper.get('button').trigger('click')
    await wrapper.findAll('[role="alert"] button')[1]?.trigger('click')

    expect(wrapper.emitted('delete-data')).toBeUndefined()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('shows why the data was not deleted', () => {
    expect(mountData(2, 'permission denied').get('[role="alert"]').text()).toBe('permission denied')
  })
})
