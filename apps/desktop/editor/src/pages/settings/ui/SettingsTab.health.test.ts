/**
 * The settings tab opened at a setting a health item of a plugin names: the page of the plugin, the
 * bar of the item, and the field marked and brought into view.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import type { WindowHandle } from '@/entities/tab'
import type { PluginHealthItem } from '@/entities/plugin'
import { createInstallation, createPluginInfo } from '../testing/installation'
import { useSettingsTab } from '../kind'
import SettingsTab from './SettingsTab.vue'
import SettingRow from './setting-row/SettingRow.vue'

const ATLAS = createPluginInfo('org.example.atlas', 'Atlas', {
  settings: [
    {
      key: 'source',
      type: 'enum',
      title: 'Source',
      description: 'Where the atlas reads its maps',
      default: 'local',
      options: [
        { value: 'local', title: 'Local maps', detail: '' },
        { value: 'connection', title: 'Connection', detail: '' },
      ],
    },
    { key: 'zoom', type: 'number', title: 'Zoom', description: '', default: 3 },
  ],
})

const CONNECTION: PluginHealthItem = {
  pluginId: 'org.example.atlas',
  key: 'connection',
  severity: 'error',
  text: 'Choose a connection',
  detail: '',
  action: {
    command: 'app:settings.open',
    values: { plugin: 'org.example.atlas', setting: 'source' },
    title: '',
  },
}

const scrolled: string[] = []
const originalScroll = HTMLElement.prototype.scrollIntoView

beforeEach(() => {
  scrolled.length = 0
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe = vi.fn()
      unobserve = vi.fn()
      disconnect = vi.fn()
    },
  )
  HTMLElement.prototype.scrollIntoView = function (this: HTMLElement) {
    scrolled.push(this.querySelector('.settings__name')?.textContent ?? '')
  }
})

afterEach(() => {
  HTMLElement.prototype.scrollIntoView = originalScroll
  vi.unstubAllGlobals()
})

describe('the settings opened at a setting of a plugin', () => {
  it('show the page of the plugin with the item and the setting marked and in view', async () => {
    const installation = createInstallation({
      plugins: ref([ATLAS]),
      items: ref([CONNECTION]),
    })
    const handle = { openTab: async (kind: string) => kind } as unknown as WindowHandle
    const settings = useSettingsTab(handle, installation, vi.fn())

    settings.openSettings({ pluginId: 'org.example.atlas', setting: 'source' })
    const tab = mount(SettingsTab, { props: { state: settings.state }, attachTo: document.body })
    await flushPromises()

    expect(tab.get('h2').text()).toBe('Atlas')
    expect(tab.get('[role="alert"]').text()).toContain('Choose a connection')
    expect(tab.get('[role="tab"][aria-selected="true"]').text()).toBe('Settings')
    const marked = tab.findAllComponents(SettingRow).filter((one) => one.props('isMarked'))
    expect(marked.map((one) => one.props('name'))).toStrictEqual(['Source'])
    expect(scrolled).toStrictEqual(['Source'])
    tab.unmount()
  })

  it('mark the setting no longer once the user goes back and opens the plugin again', async () => {
    const installation = createInstallation({ plugins: ref([ATLAS]) })
    const handle = { openTab: async (kind: string) => kind } as unknown as WindowHandle
    const settings = useSettingsTab(handle, installation, vi.fn())

    settings.openSettings({ pluginId: 'org.example.atlas', setting: 'source' })
    const tab = mount(SettingsTab, { props: { state: settings.state }, attachTo: document.body })
    await flushPromises()
    await tab.get('button[aria-label="All plugins"]').trigger('click')
    await tab.get('button[aria-label="Atlas"]').trigger('click')
    await flushPromises()

    const marked = tab.findAllComponents(SettingRow).filter((one) => one.props('isMarked'))
    expect(marked).toStrictEqual([])
    expect(scrolled).toStrictEqual(['Source'])
    tab.unmount()
  })

  it('scroll to the setting again on a repeated request', async () => {
    const installation = createInstallation({ plugins: ref([ATLAS]) })
    const handle = { openTab: async (kind: string) => kind } as unknown as WindowHandle
    const settings = useSettingsTab(handle, installation, vi.fn())

    settings.openSettings({ pluginId: 'org.example.atlas', setting: 'source' })
    const tab = mount(SettingsTab, { props: { state: settings.state }, attachTo: document.body })
    await flushPromises()
    settings.openSettings({ pluginId: 'org.example.atlas', setting: 'source' })
    await flushPromises()

    expect(scrolled).toStrictEqual(['Source', 'Source'])
    tab.unmount()
  })

  it('list the plugin with the label of its worst item', async () => {
    const installation = createInstallation({
      plugins: ref([ATLAS]),
      items: ref([CONNECTION]),
    })
    const handle = { openTab: async (kind: string) => kind } as unknown as WindowHandle
    const settings = useSettingsTab(handle, installation, vi.fn())

    const tab = mount(SettingsTab, { props: { state: settings.state }, attachTo: document.body })
    await flushPromises()

    const row = tab.get('button[aria-label="Atlas, Error"]')
    expect(row.text()).toContain('Error')
    tab.unmount()
  })
})
