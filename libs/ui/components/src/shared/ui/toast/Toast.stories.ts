import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { Toast } from '.'
import { DARK, expectDark } from '@/shared/fixtures/theme'

interface Knobs {
  text: string
  durationMs: number
  onClose: () => void
}

const meta: Meta<Knobs> = {
  title: 'Controls/Toast',
  parameters: { layout: 'fullscreen' },
  argTypes: { text: { control: 'text' }, durationMs: { control: 'number' } },
  args: { text: 'Recovery kit saved', durationMs: 600_000, onClose: fn() },
  render: (args) => ({
    components: { Toast },
    setup: () => ({ args }),
    template: `<Toast :text="args.text" :duration-ms="args.durationMs" @close="args.onClose" />`,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** A message that something the person did is done. */
export const Saved: Story = {
  play: async () => {
    await expect(within(document.body).getByRole('status')).toHaveTextContent('Recovery kit saved')
  },
}

export const SavedInTheDarkTheme: Story = {
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    await expect(within(document.body).getByRole('status')).toHaveTextContent('Recovery kit saved')
  },
}

/** A long message wraps inside the toast. */
export const LongText: Story = {
  args: { text: Array.from({ length: 30 }, (_, index) => `word ${index + 1}`).join(' ') },
  play: async () => {
    const status = within(document.body).getByRole('status')
    await waitFor(() => expect(status).toHaveTextContent('word 1 word 2 word 3'))
  },
}

/** An empty text shows nothing. */
export const Empty: Story = {
  args: { text: '' },
  play: async () => {
    await expect(within(document.body).getByRole('status')).toHaveTextContent('')
  },
}

/** The toast asks to be closed once its duration has passed, also while the person goes on working. */
export const AsksToBeClosed: Story = {
  args: { durationMs: 300 },
  play: async ({ args }) => {
    await userEvent.tab()
    await waitFor(() => expect(args.onClose).toHaveBeenCalledTimes(1))
  },
}
