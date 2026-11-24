/**
 * The settings page of a plugin that declares tabs: the strip shows over the views, each view is a
 * card of its own, and the default tab holds the plain rows.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
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
    getSettingsViews: () =>
      new Map([
        ['vaults', { component: createView('Synced vaults') }],
        ['shares', { component: createView('Shared folders') }],
      ]),
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

/** One tab: no strip, the views stand as cards above the card of the rows. */
export const SingleTab: Story = {
  render: () => draw({ ...PLAIN, settingsViews: [{ id: 'vaults', title: 'Synced vaults' }] }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.queryByRole('radiogroup')).toBeNull()
    await expect(canvas.getByText('Synced vaults')).toBeVisible()
    await expect(canvas.getByText('Server')).toBeVisible()
  },
}

/** Two tabs: the default tab holds the rows, the declared tab holds its views as separate cards. */
export const WithTabs: Story = {
  render: () =>
    draw({
      ...PLAIN,
      settingsTabs: [{ id: 'sync', title: 'Sync' }],
      settingsViews: [
        { id: 'vaults', title: 'Synced vaults', tab: 'sync' },
        { id: 'shares', title: 'Shared folders', tab: 'sync' },
      ],
    }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('radio', { name: 'Settings' })).toBeChecked()
    await expect(canvas.getByText('Server')).toBeVisible()
    await userEvent.click(canvas.getByRole('radio', { name: 'Sync' }))
    await expect(canvas.getByText('Synced vaults')).toBeVisible()
    await expect(canvas.getByText('Shared folders')).toBeVisible()
    await expect(canvas.queryByText('Server')).toBeNull()
  },
}
