/**
 * Every situation a confirmation dialog has to survive. Also the test corpus: each story is run
 * in a browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import ConfirmDialog from './ConfirmDialog.vue'

interface Knobs {
  text: string
  tone: 'danger' | 'default'
}

const meta: Meta<Knobs> = {
  title: 'Window/Confirm dialog',
  component: ConfirmDialog,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    text: { control: 'text' },
    tone: { control: 'inline-radio', options: ['danger', 'default'] },
  },
  args: { text: 'Delete the page? It is removed from this machine.', tone: 'danger' },
  render: (args) => ({
    components: { ConfirmDialog },
    setup: () => ({ args }),
    template: `<ConfirmDialog :text="args.text" :tone="args.tone" label="Delete page" confirm-text="Delete" cancel-text="Cancel" />`,
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
