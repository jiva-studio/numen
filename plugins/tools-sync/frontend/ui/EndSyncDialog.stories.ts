import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import EndSyncDialog from './EndSyncDialog.vue'
import { decorateWithFrame } from './storyFrame'
import '../styles.css'

const meta = {
  title: 'NumenSync/EndSyncDialog',
  component: EndSyncDialog,
  decorators: [decorateWithFrame],
  render: (args) => ({
    components: { EndSyncDialog },
    setup: () => ({ args }),
    template: '<EndSyncDialog v-bind="args" />',
  }),
} satisfies Meta<typeof EndSyncDialog>

export default meta
type Story = StoryObj<typeof meta>

export const StopSyncing: Story = {
  args: { kind: 'stop' },
  play: async ({ canvasElement }) => {
    const dialog = within(canvasElement.ownerDocument.body)
    await expect(await dialog.findByRole('alertdialog', { name: 'Stop syncing' })).toBeVisible()
    await expect(
      dialog.getByText('This device stops syncing. Files stay here and in the cloud.'),
    ).toBeVisible()
    await expect(dialog.getByRole('button', { name: 'Stop syncing' })).toBeVisible()
    await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeVisible()
  },
}

export const LeaveFolder: Story = {
  args: { kind: 'leave' },
  play: async ({ canvasElement }) => {
    const dialog = within(canvasElement.ownerDocument.body)
    await expect(await dialog.findByRole('alertdialog', { name: 'Leave folder' })).toBeVisible()
    await expect(
      dialog.getByText('You stop receiving changes. Files stay on this device.'),
    ).toBeVisible()
    await expect(dialog.getByRole('button', { name: 'Leave' })).toBeVisible()
  },
}
