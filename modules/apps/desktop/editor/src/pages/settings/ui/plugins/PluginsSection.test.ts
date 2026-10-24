import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import PluginsSection from './PluginsSection.vue'
import PluginsList from './PluginsList.vue'
import PluginDetail from './PluginDetail.vue'
import { createInstallation, createPluginInfo } from '@/testing/installation'
import type { Installation, SettingsTabState } from '../../types'
import type { PluginInfo } from '@/entities/settings'
import { WORDS as words } from '../../words'

beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
})

const createSection = (plugins: readonly PluginInfo[] = []) => {
  const installation: Installation = createInstallation({
    plugins: ref(plugins),
  })
  const state: SettingsTabState = { installation }
  const wrapper = mount(PluginsSection, {
    props: { state },
  })
  return { wrapper, installation }
}

describe('PluginsSection', () => {
  it('renders nothing when there are no plugins', () => {
    const { wrapper } = createSection([])

    expect(wrapper.find('section').exists()).toBe(false)
  })

  it('renders default list view when plugins exist and no plugin is selected', () => {
    const plugins = [
      createPluginInfo('plugin.one', 'Plugin One'),
      createPluginInfo('plugin.two', 'Plugin Two'),
    ]
    const { wrapper } = createSection(plugins)

    expect(wrapper.find('section').exists()).toBe(true)
    const heading = wrapper.find('.settings__heading')
    expect(heading.exists()).toBe(true)
    expect(heading.text()).toBe(words.plugins)

    expect(wrapper.findComponent(PluginsList).exists()).toBe(true)
    expect(wrapper.findComponent(PluginDetail).exists()).toBe(false)
  })

  it('transitions to detail view when selectPlugin is triggered', async () => {
    const plugins = [
      createPluginInfo('plugin.one', 'Plugin One'),
      createPluginInfo('plugin.two', 'Plugin Two'),
    ]
    const { wrapper } = createSection(plugins)

    const list = wrapper.findComponent(PluginsList)
    list.vm.$emit('selectPlugin', 'plugin.two')
    await wrapper.vm.$nextTick()

    expect(wrapper.findComponent(PluginDetail).exists()).toBe(true)
    expect(wrapper.findComponent(PluginDetail).props('plugin').id).toBe('plugin.two')
    expect(wrapper.findComponent(PluginsList).exists()).toBe(false)
    expect(wrapper.find('.settings__heading').exists()).toBe(false)
  })

  it('returns to list view when back is clicked in detail view', async () => {
    const plugins = [createPluginInfo('plugin.one', 'Plugin One')]
    const { wrapper } = createSection(plugins)

    wrapper.findComponent(PluginsList).vm.$emit('selectPlugin', 'plugin.one')
    await wrapper.vm.$nextTick()

    expect(wrapper.findComponent(PluginDetail).exists()).toBe(true)

    wrapper.findComponent(PluginDetail).vm.$emit('back')
    await wrapper.vm.$nextTick()

    expect(wrapper.findComponent(PluginsList).exists()).toBe(true)
    expect(wrapper.findComponent(PluginDetail).exists()).toBe(false)
    expect(wrapper.find('.settings__heading').exists()).toBe(true)
  })
})
