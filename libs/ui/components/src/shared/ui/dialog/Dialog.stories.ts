/**
 * The one dialog of the interface in its three widths. Each story checks what every dialog shares: the role and the
 * title, the buttons at the right with the primary last, Escape and the backdrop that cancel, and the focus on the
 * first control.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import { Button } from '../button'
import Dialog from './Dialog.vue'

interface Knobs {
  size: 'sm' | 'md' | 'lg'
  onCancel: () => void
}

const meta: Meta<Knobs> = {
  title: 'Controls/Dialog/Sizes',
  component: Dialog,
  parameters: { layout: 'fullscreen' },
  args: { size: 'md', onCancel: fn() },
  argTypes: { size: { control: 'inline-radio', options: ['sm', 'md', 'lg'] } },
  render: (args) => ({
    components: { Dialog, Button },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="position:relative;contain:strict;inline-size:60rem;block-size:30rem;background:var(--numen-surface)">
        <Dialog label='Rename the vault' hint="The name is shown in the list of vaults." :size="args.size" @cancel="args.onCancel">
          <input aria-label='Name' value="Notes" />
          <template #actions>
            <Button variant="ghost" @click="args.onCancel">Cancel</Button>
            <Button variant="solid">Rename</Button>
          </template>
        </Dialog>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const getDialog = () => within(document.body).getByRole('dialog', { name: 'Rename the vault' })

/** Small: a short question or one field. */
export const Small: Story = {
  args: { size: 'sm' },
  play: async () => {
    await expect(getDialog()).toBeVisible()
  },
}

/** Medium: the width of nearly every dialog. It is as high as its content. */
export const Medium: Story = {
  play: async ({ args }) => {
    await expect(getDialog()).toBeVisible()
    await expect(within(getDialog()).getByRole('textbox', { name: 'Name' })).toHaveFocus()
    await userEvent.keyboard('{Escape}')
    await expect(args.onCancel).toHaveBeenCalled()
  },
}

/** Large: a body with a list or a picture. */
export const Large: Story = {
  args: { size: 'lg' },
  play: async () => {
    await expect(getDialog()).toBeVisible()
  },
}
