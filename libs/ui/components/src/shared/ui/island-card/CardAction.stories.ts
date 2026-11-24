/** One action of a card header: an icon, its label and the states the label and the button take. */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { Plus } from '@lucide/vue'
import { expect, within } from 'storybook/test'
import CardAction from './CardAction.vue'

const meta: Meta<typeof CardAction> = {
  title: 'Controls/IslandCard/Action',
  component: CardAction,
  args: { label: 'Create' },
  render: (args) => ({
    components: { CardAction, Plus },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="padding:1rem;background:var(--numen-surface)">
        <CardAction v-bind="args"><Plus :size="14" aria-hidden="true" /></CardAction>
      </div>
    `,
  }),
}
export default meta
type Story = StoryObj<typeof CardAction>

/** The label stands beside the icon. */
export const Default: Story = {
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByRole('button', { name: 'Create' })).toBeVisible()
    await expect(within(canvasElement).getByText('Create')).toBeVisible()
  },
}

/** The primary action is drawn stronger than the others. */
export const Primary: Story = {
  args: { isPrimary: true },
}

/** An action that opens a menu has a chevron after its label. */
export const WithMenu: Story = {
  args: { hasMenu: true },
}

/** Only the icon shows; the label stays the name of the button. */
export const IconOnly: Story = {
  args: { isIconOnly: true },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByRole('button', { name: 'Create' })).toBeVisible()
    await expect(within(canvasElement).queryByText('Create')).toBeNull()
  },
}

/** A disabled action takes no press. */
export const Disabled: Story = {
  args: { isDisabled: true },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByRole('button', { name: 'Create' })).toBeDisabled()
  },
}
