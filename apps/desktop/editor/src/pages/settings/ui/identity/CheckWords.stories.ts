/** The check of three recovery words, in the area of a tab: empty, filled, wrong, busy. */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import IdentitySetup from './IdentitySetup.vue'
import type { IdentitySetupProps } from './IdentitySetup.types'

const onBack = fn()

const meta: Meta<IdentitySetupProps> = {
  title: 'Window/Identity/Check words',
  component: IdentitySetup,
  parameters: { layout: 'fullscreen' },
  args: {
    stage: 'check',
    recoveryWords: [],
    positions: [2, 5, 9],
    entered: {},
    wrong: [],
    isComplete: false,
    problem: '',
    isBusy: false,
  },
  render: (args) => ({
    components: { IdentitySetup },
    setup: () => ({ args, onBack }),
    template: `
      <div
        class="numen"
        style="position:relative;contain:strict;box-sizing:border-box;inline-size:760px;block-size:560px;margin:1.5rem auto;border:1px solid var(--numen-rule);background:var(--numen-surface)"
      >
        <IdentitySetup v-bind="args" @back="onBack" />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<IdentitySetupProps>

/** Three empty fields and a Continue that waits. */
export const Empty: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body)
    await expect(canvas.getByRole('dialog', { name: 'Check the words' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Continue' })).toBeDisabled()
  },
}

/** All three words are entered and Continue is asked for. */
export const Filled: Story = {
  args: { entered: { 2: 'basil', 5: 'fjord', 9: 'ivory' }, isComplete: true },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement.ownerDocument.body).getByRole('button', { name: 'Continue' })).toBeEnabled()
  },
}

/** A refusal of the core stays in the dialog. */
export const Mistaken: Story = {
  args: { entered: { 2: 'basil', 5: 'wrong', 9: 'ivory' }, wrong: [5], problem: 'Check the words.' },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement.ownerDocument.body).getByRole('alert')).toHaveTextContent('Check the words.')
  },
}

/** While the identity is made, Back and Continue wait. */
export const Busy: Story = {
  args: { isBusy: true, isComplete: true },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body)
    await expect(canvas.getByRole('button', { name: 'Back' })).toBeDisabled()
    await userEvent.tab()
  },
}
