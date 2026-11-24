/**
 * The restore from words in a browser: one field, Restore waits for the 12 words.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import { ref } from 'vue'
import { readWordsOnly } from '../../model/readProof'
import { useRecoveryProof } from '../../model/useRecoveryProof'
import RestoreWordsDialog from './RestoreWordsDialog.vue'

interface Knobs {
  problem: string
  isBusy: boolean
}

const PHRASE = Array.from({ length: 12 }, (_, index) => `word${index + 1}`).join(' ')

const meta: Meta<Knobs> = {
  title: 'Window/Identity/RestoreWords',
  component: RestoreWordsDialog,
  parameters: { layout: 'fullscreen' },
  args: { problem: '', isBusy: false },
  render: (args) => ({
    components: { RestoreWordsDialog },
    setup: () => ({ args, proof: useRecoveryProof(ref(''), readWordsOnly) }),
    template: `
      <div class="numen" style="position:relative;contain:strict;box-sizing:border-box;inline-size:760px;block-size:560px;margin:1.5rem auto;border:1px solid var(--numen-rule);background:var(--numen-surface)">
        <RestoreWordsDialog
          :text="proof.text.value"
          :reading="proof.reading.value"
          :problem="args.problem"
          :is-busy="args.isBusy"
          @change="proof.setText"
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** Nothing entered: no message line and Restore waits. */
export const Empty: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('dialog', { name: 'Recovery words' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Restore' })).toBeDisabled()
  },
}

/** The 12 words are recognized. */
export const Words: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('textbox'))
    await userEvent.paste(PHRASE)
    await expect(canvas.getByText('12 words recognized')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Restore' })).toBeEnabled()
  },
}

/** While the core works, everything is disabled. */
export const Busy: Story = {
  args: { isBusy: true },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByRole('textbox')).toBeDisabled()
  },
}
