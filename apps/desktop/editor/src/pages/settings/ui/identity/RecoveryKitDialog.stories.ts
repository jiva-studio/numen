/** The dialog that asks for the words before a recovery kit is saved, in the area of a tab. */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import { ref } from 'vue'
import { useRecoveryProof } from '../../model/useRecoveryProof'
import RecoveryKitDialog from './RecoveryKitDialog.vue'

interface Knobs {
  problem: string
  isBusy: boolean
}

const onDownload = fn()
const onCancel = fn()
const PHRASE = Array.from({ length: 12 }, (_, index) => `word${index + 1}`).join(' ')

const meta: Meta<Knobs> = {
  title: 'Window/Identity/Recovery kit',
  component: RecoveryKitDialog,
  parameters: { layout: 'fullscreen' },
  args: { problem: '', isBusy: false },
  render: (args) => ({
    components: { RecoveryKitDialog },
    setup: () => ({ args, proof: useRecoveryProof(ref('')), onDownload, onCancel }),
    template: `
      <div
        class="numen"
        style="position:relative;contain:strict;box-sizing:border-box;inline-size:760px;block-size:560px;margin:1.5rem auto;border:1px solid var(--numen-rule);background:var(--numen-surface)"
      >
        <RecoveryKitDialog
          :text="proof.text.value"
          :reading="proof.reading.value"
          :problem="args.problem"
          :is-busy="args.isBusy"
          @change="proof.setText"
          @download="onDownload"
          @cancel="onCancel"
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** Nothing entered: the save waits. */
export const Empty: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('button', { name: 'Download PDF' })).toBeDisabled()
  },
}

/** The 12 words are entered and the kit is saved. */
export const Filled: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('textbox'))
    await userEvent.paste(PHRASE)
    const button = canvas.getByRole('button', { name: 'Download PDF' })
    await expect(button).toBeEnabled()
    await userEvent.click(button)
    await expect(onDownload).toHaveBeenCalled()
  },
}

/** The core refused the words. */
export const Refused: Story = {
  args: { problem: 'These words do not belong to this identity.' },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByRole('alert')).toHaveTextContent('do not belong')
  },
}

/** While the PDF is made, the field and the buttons wait. */
export const Busy: Story = {
  args: { isBusy: true },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByRole('button', { name: 'Saving…' })).toBeDisabled()
  },
}
