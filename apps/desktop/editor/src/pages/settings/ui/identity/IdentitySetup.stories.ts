/**
 * The two steps that make a new identity, in the area of a tab: the twelve words that are written down, and the check
 * of three of them. Both are drawn in the shared dialog, so the screenshots show its width, rhythm and buttons.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import IdentitySetup from './IdentitySetup.vue'
import type { IdentitySetupProps } from './IdentitySetup.types'

const onCheck = fn()
const onBack = fn()
const onCancel = fn()
const onKeep = fn()

const WORDS =
  'amber basil cedar delta ember fjord grove harbor ivory jungle kernel lantern'.split(' ')

const meta: Meta<IdentitySetupProps> = {
  title: 'Window/Identity/Create words',
  component: IdentitySetup,
  parameters: { layout: 'fullscreen' },
  args: {
    stage: 'words',
    recoveryWords: WORDS,
    positions: [2, 5, 9],
    entered: {},
    wrong: [],
    isComplete: false,
    problem: '',
    isBusy: false,
  },
  render: (args) => ({
    components: { IdentitySetup },
    setup: () => ({ args, onCheck, onBack, onCancel, onKeep }),
    template: `
      <div
        class="numen"
        style="position:relative;contain:strict;box-sizing:border-box;inline-size:760px;block-size:560px;margin:1.5rem auto;border:1px solid var(--numen-rule);background:var(--numen-surface)"
      >
        <IdentitySetup v-bind="args" @check="onCheck" @back="onBack" @cancel="onCancel" @keep="onKeep" />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<IdentitySetupProps>

/** The twelve words, shown once, and the warning that nobody can recover them. */
export const Words: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('dialog', { name: 'Recovery words' })).toBeVisible()
    await expect(canvas.getByText('amber')).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'I wrote the words down' }))
    await expect(onCheck).toHaveBeenCalled()
  },
}

/** Cancel closes the dialog. */
export const Cancelled: Story = {
  play: async ({ canvasElement }) => {
    await userEvent.click(within(canvasElement).getByRole('button', { name: 'Cancel' }))
    await expect(onCancel).toHaveBeenCalled()
  },
}
