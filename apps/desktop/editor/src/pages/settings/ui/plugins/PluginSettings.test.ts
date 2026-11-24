import { describe, expect, it } from 'vitest'
import { defineComponent, ref, type Component } from 'vue'
import { mount } from '@vue/test-utils'
import { IslandCard } from '@numen/ui'
import PluginSettings from './PluginSettings.vue'
import { createInstallation, createPluginInfo } from '@/pages/settings/testing/installation'
import type { PluginInfo } from '@/entities/settings'

const card = (name: string): Component =>
  defineComponent({
    components: { IslandCard },
    template: `<IslandCard title="${name}"><p class="view" data-view="${name}">${name}</p></IslandCard>`,
  })

const toggle = {
  key: 'on',
  type: 'boolean' as const,
  title: 'Switched',
  description: '',
  default: false,
}

const mountSettings = (overrides: Partial<PluginInfo>) => {
  const plugin = createPluginInfo('sync.plugin', 'Sync', overrides)
  const installation = createInstallation({
    plugins: ref([plugin]),
    getSettingsViews: () =>
      new Map([
        ['vaults', { component: card('vaults') }],
        ['shares', { component: card('shares') }],
        ['plain', { component: card('plain') }],
      ]),
  })
  return mount(PluginSettings, {
    props: {
      settings: installation.settings,
      host: installation.plugin,
      plugin,
      marked: [],
      focused: null,
    },
  })
}

const views = (wrapper: ReturnType<typeof mountSettings>) =>
  wrapper.findAll('.view').map((one) => one.attributes('data-view'))

const TABS = {
  settingsTabs: [{ id: 'sync', title: 'Sync' }],
  settingsViews: [
    { id: 'vaults', title: 'Vaults', tab: 'sync' },
    { id: 'shares', title: 'Shares', tab: 'sync' },
    { id: 'plain', title: 'Plain' },
  ],
  settings: [toggle],
}

describe('the tabs of the settings page of a plugin', () => {
  it('shows no tab strip for a plugin that declares no tab', () => {
    const wrapper = mountSettings({
      settingsViews: [{ id: 'vaults', title: 'Vaults' }],
      settings: [toggle],
    })

    expect(wrapper.find('[data-testid="plugin-settings-tabs"]').exists()).toBe(false)
    expect(views(wrapper)).toStrictEqual(['vaults'])
  })

  it('shows the default tab first and the declared tabs after it', () => {
    const wrapper = mountSettings(TABS)

    const labels = wrapper.findAll('[role="radio"]').map((one) => one.text())

    expect(labels).toStrictEqual(['Settings', 'Sync'])
  })

  it('holds the plain rows and the views without a tab in the default tab', () => {
    const wrapper = mountSettings(TABS)

    expect(views(wrapper)).toStrictEqual(['plain'])
    expect(wrapper.text()).toContain('Switched')
  })

  it('draws the views of a declared tab and not the plain rows after the tab is chosen', async () => {
    const wrapper = mountSettings(TABS)

    await wrapper.findAll('[role="radio"]')[1]?.trigger('click')

    expect(views(wrapper)).toStrictEqual(['vaults', 'shares'])
    expect(wrapper.text()).not.toContain('Switched')
  })

  it('draws the views as cards of their own, with no card around them', async () => {
    const wrapper = mountSettings(TABS)
    await wrapper.findAll('[role="radio"]')[1]?.trigger('click')

    const drawn = wrapper.findAll('.view')

    expect(drawn).toHaveLength(2)
    for (const one of drawn) expect(one.element.closest('.settings-card')).toBeNull()
  })

  it('keeps the plain rows in a card of their own, apart from the views', () => {
    const wrapper = mountSettings(TABS)

    const rows = wrapper.get('.settings-card')

    expect(rows.text()).toContain('Switched')
    expect(rows.find('.view').exists()).toBe(false)
  })

  it('shows no card for the plain rows when the plugin has none', () => {
    const wrapper = mountSettings({
      settingsViews: TABS.settingsViews,
      settingsTabs: TABS.settingsTabs,
    })

    expect(wrapper.find('.settings-card').exists()).toBe(false)
  })
})
