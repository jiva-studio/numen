import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import DeleteCloudCopyDialog from './DeleteCloudCopyDialog.vue'
import { decorateWithFrame } from './storyFrame'
import '../styles.css'

const meta = {
  title: 'NumenSync/DeleteCloudCopyDialog',
  component: DeleteCloudCopyDialog,
  decorators: [decorateWithFrame],
  render: (args) => ({
    components: { DeleteCloudCopyDialog },
    setup: () => ({ args }),
    template: '<DeleteCloudCopyDialog v-bind="args" />',
  }),
} satisfies Meta<typeof DeleteCloudCopyDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Question: Story = {
  args: { name: 'Recipes' },
  play: async ({ canvasElement }) => {
    const dialog = within(canvasElement.ownerDocument.body)
    await expect(
      await dialog.findByRole('alertdialog', {
        name: 'Delete the cloud copy of Recipes? It cannot be restored.',
      }),
    ).toBeVisible()
    await expect(dialog.getByRole('button', { name: 'Delete' })).toBeVisible()
    await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeVisible()
  },
}
