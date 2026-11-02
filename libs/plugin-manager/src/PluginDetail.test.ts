import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { Button, Switch } from '@numen/ui'
import PluginDetail from './PluginDetail.vue'
import PluginData from './PluginData.vue'
import PluginStopped from './PluginStopped.vue'
import type { PluginDetailProps } from './PluginDetail.types'
import { createManaged, createOffered } from './testing/managed'

const baseProps = (overrides: Partial<PluginDetailProps> = {}): PluginDetailProps => ({
  plugin: createManaged('sample.notes', 'Notes', { version: '2.1.0', description: 'Takes notes.' }),
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

describe('PluginDetail', () => {
  it('renders the name, the version and the description of the plugin', () => {
    const wrapper = mountDetail()

    expect(wrapper.find('h2').text()).toContain('Notes')
    expect(wrapper.find('.plugin-version-badge').text()).toBe('v2.1.0')
    expect(wrapper.text()).toContain('Takes notes.')
  })

  it('goes back when the back button is clicked', async () => {
    const wrapper = mountDetail()

    await wrapper.findComponent(Button).trigger('click')

    expect(wrapper.emitted('back')).toHaveLength(1)
  })

  it('shows the enabled switch of an installed plugin and emits its change', () => {
    const wrapper = mountDetail({ isEnabled: false })
    const toggle = wrapper.findComponent(Switch)

    expect(toggle.props('modelValue')).toBe(false)
    expect(wrapper.find('[aria-label="Enabled Notes"]').exists()).toBe(true)

    toggle.vm.$emit('update:modelValue', true)

    expect(wrapper.emitted('toggle-enabled')).toStrictEqual([[true]])
  })

  it('splits the README on Read more and expands the rest', async () => {
    const wrapper = mountDetail({ readme: '# Notes\n\nSummary.\n\n<!-- more -->\n\nDetails here.' })

    expect(wrapper.text()).toContain('Summary.')
    expect(wrapper.text()).not.toContain('Details here.')

    await wrapper.find('.plugin-detail__toggle').trigger('click')

    expect(wrapper.text()).toContain('Details here.')
    expect(wrapper.find('.plugin-detail__toggle').attributes('aria-expanded')).toBe('true')
    expect(wrapper.find('.plugin-detail__toggle').text()).toBe('Show less')
  })

  it('draws the failure card of a suspended plugin and emits retry and disable', async () => {
    const failure = { reason: 'Failed to start', detail: 'exec error', failedAt: 0 }
    const wrapper = mountDetail({ plugin: createManaged('sample.notes', 'Notes', { failure }) })
    const failureCard = wrapper.findComponent(PluginStopped)

    expect(failureCard.text()).toContain('Failed to start')
    expect(failureCard.find('details').text()).toContain('exec error')

    const [retry, disable] = failureCard.findAllComponents(Button)
    await retry?.trigger('click')
    await disable?.trigger('click')

    expect(wrapper.emitted('retry')).toHaveLength(1)
    expect(wrapper.emitted('toggle-enabled')).toStrictEqual([[false]])
  })

  it('draws no failure card and no details disclosure without a failure detail', () => {
    expect(mountDetail().findComponent(PluginStopped).exists()).toBe(false)

    const failure = { reason: 'Failed to start', detail: '', failedAt: 0 }
    const wrapper = mountDetail({ plugin: createManaged('sample.notes', 'Notes', { failure }) })
    expect(wrapper.find('details').exists()).toBe(false)
  })

  it('draws the consent switch of a plugin that asks for models and emits its change', () => {
    const wrapper = mountDetail({
      plugin: createManaged('sample.notes', 'Notes', { consent: 'allowed' }),
    })
    const toggles = wrapper.findAllComponents(Switch)
    const consent = toggles[1]

    expect(wrapper.text()).toContain('Allow model use')
    expect(consent?.props('modelValue')).toBe(true)

    consent?.vm.$emit('update:modelValue', false)

    expect(wrapper.emitted('consent-change')).toStrictEqual([[false]])
  })

  it('draws no settings block without consent and without settings', () => {
    expect(mountDetail().find('.plugin-detail__settings').exists()).toBe(false)
  })

  it('draws the settings slot only where the application says it has settings', () => {
    expect(mountDetail({}, true).find('.settings-stub').exists()).toBe(false)

    const wrapper = mountDetail({ hasSettings: true }, true)

    expect(wrapper.find('.plugin-detail__settings .settings-stub').exists()).toBe(true)
  })
})

describe('PluginDetail for a plugin that is not installed', () => {
  it('offers install where the application can install', async () => {
    const wrapper = mountDetail({
      plugin: createOffered('sample.offer', 'Offered'),
      canInstall: true,
    })

    expect(wrapper.findComponent(Switch).exists()).toBe(false)
    const install = wrapper.findAllComponents(Button)[1]
    expect(install?.text()).toBe('Install')

    await install?.trigger('click')

    expect(wrapper.emitted('install')).toHaveLength(1)
  })

  it('says it is not installed where the application cannot install', () => {
    const wrapper = mountDetail({ plugin: createOffered('sample.offer', 'Offered') })

    expect(wrapper.findAllComponents(Button)).toHaveLength(1)
    expect(wrapper.find('.plugin-detail__absent').text()).toBe('Not installed')
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

    await wrapper.find('button').trigger('click')
    expect(wrapper.find('[role="alert"]').text()).toContain('Delete 3 files of Notes?')
    expect(wrapper.emitted('delete-data')).toBeUndefined()

    await wrapper.findAll('[role="alert"] button')[0]?.trigger('click')

    expect(wrapper.emitted('delete-data')).toHaveLength(1)
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('keeps the data when the person keeps it', async () => {
    const wrapper = mountData(1)
    expect(wrapper.text()).toContain('1 file in the plugin')

    await wrapper.find('button').trigger('click')
    await wrapper.findAll('[role="alert"] button')[1]?.trigger('click')

    expect(wrapper.emitted('delete-data')).toBeUndefined()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('shows why the data was not deleted', () => {
    expect(mountData(2, 'permission denied').find('[role="alert"]').text()).toBe(
      'permission denied',
    )
  })
})
