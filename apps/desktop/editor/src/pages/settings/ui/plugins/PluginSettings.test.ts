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

const PAGE = {
  settingsTabs: [{ id: 'sync', title: 'Sync' }],
  settingsViews: [
    { id: 'vaults', title: 'Vaults', tab: 'sync' },
    { id: 'shares', title: 'Shares', tab: 'sync' },
    { id: 'plain', title: 'Plain' },
  ],
  settings: [toggle],
}

describe('the Settings tab of a plugin page', () => {
  it('draws the views that name no tab and leaves the views of a contributed tab out', () => {
    expect(views(mountSettings(PAGE))).toStrictEqual(['plain'])
  })

  it('draws no strip of tabs of its own', () => {
    const wrapper = mountSettings(PAGE)

    expect(wrapper.find('[role="tablist"], [role="radiogroup"]').exists()).toBe(false)
  })

  it('keeps the plain rows in a card of their own, apart from the views', () => {
    const wrapper = mountSettings(PAGE)

    const rows = wrapper.get('.settings-card')

    expect(rows.text()).toContain('Switched')
    expect(rows.find('.view').exists()).toBe(false)
  })

  it('shows no card for the plain rows when the plugin has none', () => {
    const wrapper = mountSettings({ ...PAGE, settings: [] })

    expect(wrapper.find('.settings-card').exists()).toBe(false)
  })
})
