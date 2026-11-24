/**
 * The Settings tab of a plugin page: the views that name no tab stand as cards of their own above
 * the card of the rows.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import { defineComponent, ref } from 'vue'
import { IslandCard } from '@numen/ui'
import PluginSettings from './PluginSettings.vue'
import type { PluginInfo } from '@/entities/settings'
import { createInstallation, createPluginInfo } from '@/pages/settings/testing/installation'

const createView = (title: string) =>
  defineComponent({
    components: { IslandCard },
    template: `<IslandCard title="${title}"><p style="margin:0">${title} of this plugin</p></IslandCard>`,
  })

const PLAIN: Partial<PluginInfo> = {
  settings: [
    { key: 'server', type: 'string', title: 'Server', description: '', default: 'sync.example' },
    { key: 'interval', type: 'number', title: 'Interval', description: '', default: 5 },
  ],
}

const draw = (overrides: Partial<PluginInfo>) => {
  const plugin = createPluginInfo('sample.sync', 'Sample sync', overrides)
  const installation = createInstallation({
    plugins: ref([plugin]),
    getSettingsViews: () => new Map([['vaults', { component: createView('Synced vaults') }]]),
  })
  return {
    components: { PluginSettings },
    setup: () => ({ installation, plugin }),
    template: `
      <div class="numen" style="display:flex;flex-direction:column;gap:1.25rem;inline-size:720px;margin:1.5rem auto">
        <PluginSettings :settings="installation.settings" :host="installation.plugin" :plugin="plugin" :marked="[]" :focused="null" />
      </div>
    `,
  }
}

const meta: Meta = {
  title: 'Window/Settings/PluginSettings',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

/** A view with no tab stands as a card above the card of the rows. */
export const WithView: Story = {
  render: () => draw({ ...PLAIN, settingsViews: [{ id: 'vaults', title: 'Synced vaults' }] }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByText('Synced vaults')).toBeVisible()
    await expect(canvas.getByText('Server')).toBeVisible()
  },
}
