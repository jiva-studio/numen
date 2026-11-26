import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import SyncBadge from './SyncBadge.vue'
import { WORDS } from '../words'
import '../styles.css'

const meta = {
  title: 'NumenSync/Badge',
  component: SyncBadge,
  args: {
    phase: 'synced',
  },
  render: (args) => ({
    components: { SyncBadge },
    setup: () => ({ args }),
    template: `
      <div style="display: inline-flex; align-items: center; height: 2rem; padding: 0 0.5rem; background: var(--numen-raised);">
        <SyncBadge v-bind="args" />
      </div>
    `,
  }),
} satisfies Meta<typeof SyncBadge>

export default meta
type Story = StoryObj<typeof meta>

export const Syncing: Story = {
  tags: ['screenshot'],
  parameters: {
    screenshot: [
      { name: 'sync', width: 240, height: 64, drawn: 500 },
    ],
  },
  args: {
    phase: 'syncing',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const status = canvas.getByRole('status', { name: WORDS.syncingDetail })
    await expect(status).toBeVisible()
    await expect(status).toHaveTextContent(WORDS.syncingLabel)
  },
}

export const Synced: Story = {
  args: {
    phase: 'synced',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const status = canvas.getByRole('status', { name: WORDS.syncedDetail })
    await expect(status).toBeVisible()
    await expect(status).toHaveTextContent(WORDS.syncedLabel)
  },
}

export const Hidden: Story = {
  args: {
    phase: 'hidden',
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.queryByRole('status')).toBeNull()
  },
}
