/**
 * Every situation a confirmation dialog has to survive. Also the test corpus: each story is run
 * in a browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import ConfirmDialog from './ConfirmDialog.vue'

interface Knobs {
  text: string
  tone: 'danger' | 'default'
  onConfirm: () => void
  onCancel: () => void
}

const meta: Meta<Knobs> = {
  title: 'Controls/Confirm dialog',
  component: ConfirmDialog,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    text: { control: 'text' },
    tone: { control: 'inline-radio', options: ['danger', 'default'] },
  },
  args: { text: 'Delete the page? It is removed from this machine.', tone: 'danger', onConfirm: fn(), onCancel: fn() },
  render: (args) => ({
    components: { ConfirmDialog },
    setup: () => ({ args }),
    template: `<ConfirmDialog :text="args.text" :tone="args.tone" label="Delete page" confirm-text="Delete" cancel-text="Cancel" @confirm="args.onConfirm" @cancel="args.onCancel" />`,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** A dangerous question: the confirm button is drawn as a danger button. */
export const Danger: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('button', { name: 'Cancel' })).toHaveFocus()
    await userEvent.tab()
    await expect(canvas.getByRole('button', { name: 'Delete' })).toHaveFocus()
  },
}

/** A question without harm in it. */
export const Default: Story = { args: { tone: 'default', text: 'Open the page in the browser?' } }

/** A question far longer than anything asked. */
export const FarTooLong: Story = {
  args: {
    text: 'Delete everything that was written in this place and all that depends on it? '.repeat(6),
  },
}

/** A question in another script. */
export const OtherScripts: Story = { args: { text: 'Удалить заметку? Её не вернуть.' } }

/** The person confirms: the dialog says so once and does not say cancel. */
export const Confirms: Story = {
  play: async ({ args, canvasElement }) => {
    await userEvent.click(within(canvasElement).getByRole('button', { name: 'Delete' }))
    await expect(args.onConfirm).toHaveBeenCalledTimes(1)
    await expect(args.onCancel).not.toHaveBeenCalled()
  },
}

/** The person cancels with the button, and again with the Escape key. */
export const Cancels: Story = {
  play: async ({ args, canvasElement }) => {
    await userEvent.click(within(canvasElement).getByRole('button', { name: 'Cancel' }))
    await expect(args.onCancel).toHaveBeenCalledTimes(1)
    await userEvent.keyboard('{Escape}')
    await expect(args.onCancel).toHaveBeenCalledTimes(2)
    await expect(args.onConfirm).not.toHaveBeenCalled()
  },
}
