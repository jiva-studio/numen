/**
 * The restore dialog of the identity with its one field, in a browser. The checks cover what jsdom
 * cannot: the button of the restore is the library's solid button and waits for a whole entry, the
 * cancel is the ghost button, and the line under the field says what was recognized.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import { ref } from 'vue'
import { useRecoveryProof } from '../../model/useRecoveryProof'
import IdentityRestoreDialog from './IdentityRestoreDialog.vue'

interface Knobs {
  problem: string
  isBusy: boolean
}

const onRestore = fn()
const onCancel = fn()

const PHRASE = Array.from({ length: 12 }, (_, index) => `word${index + 1}`).join(' ')
const createSheet = (prefix: string): string =>
  Array.from({ length: 33 }, (_, index) => `${prefix}${index + 1}`).join(' ')

const render = (args: Knobs) => ({
  components: { IdentityRestoreDialog },
  setup: () => {
    const proof = useRecoveryProof(ref(''))
    return { args, proof, onRestore, onCancel }
  },
  template: `
    <div class="numen" style="position:relative;contain:strict;box-sizing:border-box;inline-size:760px;block-size:560px;margin:1.5rem auto;border:1px solid var(--numen-rule);background:var(--numen-surface)">
      <IdentityRestoreDialog
        :text="proof.text.value"
        :reading="proof.reading.value"
        :problem="args.problem"
        :is-busy="args.isBusy"
        @change="proof.setText"
        @restore="onRestore"
        @cancel="onCancel"
      />
    </div>
  `,
})

const meta: Meta<Knobs> = {
  title: 'Window/Identity/Restore',
  component: IdentityRestoreDialog,
  parameters: { layout: 'fullscreen' },
  argTypes: { problem: { control: 'text' }, isBusy: { control: 'boolean' } },
  args: { problem: '', isBusy: false },
  render,
}

export default meta
type Story = StoryObj<Knobs>

const getField = (canvas: ReturnType<typeof within>) =>
  canvas.getByRole('textbox', { name: 'Recovery words or sheets' })

const enter = async (canvas: ReturnType<typeof within>, text: string) => {
  await userEvent.click(getField(canvas))
  await userEvent.paste(text)
}

/** Nothing entered: the help line is shown and the restore waits. */
export const Empty: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('dialog', { name: 'Restore identity' })).toBeVisible()
    await expect(canvas.getByText(/any two sheets of your recovery kit/)).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Restore the account' })).toBeDisabled()
    await expect(canvas.getByRole('button', { name: 'Cancel' })).toHaveAttribute(
      'data-slot',
      'button',
    )
  },
}

/** The 12 words are recognized and the restore is asked for. */
export const Words: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await enter(canvas, PHRASE)
    await expect(canvas.getByText('12 words recognized')).toBeVisible()
    const button = canvas.getByRole('button', { name: 'Restore the account' })
    await expect(button).toBeEnabled()
    await userEvent.click(button)
    await expect(onRestore).toHaveBeenCalled()
  },
}

/** A 13th word is refused with its count and the restore waits. */
export const ExtraWord: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await enter(canvas, `${PHRASE} extra`)
    await expect(canvas.getByRole('alert')).toHaveTextContent('13 words')
    await expect(canvas.getByRole('button', { name: 'Restore the account' })).toBeDisabled()
  },
}

/** One sheet needs another, and two sheets are recognized on separate lines. */
export const Sheets: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await enter(canvas, createSheet('a'))
    await expect(canvas.getByText('Sheet 1 recognized, one more sheet needed')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Restore the account' })).toBeDisabled()
    await userEvent.paste(`\n${createSheet('b')}`)
    await expect(canvas.getByText('2 of 3 sheets recognized')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Restore the account' })).toBeEnabled()
  },
}

/** A repeated sheet is named and refused. */
export const RepeatedSheet: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await enter(canvas, `${createSheet('a')}\n${createSheet('a')}`)
    await expect(canvas.getByRole('alert')).toHaveTextContent('Sheet 2 is the same as sheet 1')
    await expect(canvas.getByRole('button', { name: 'Restore the account' })).toBeDisabled()
  },
}

/** The refusal of the core replaces the line under the field, and Cancel closes the dialog. */
export const Refused: Story = {
  args: { problem: 'The sheets belong to another account.' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('alert')).toHaveTextContent('belong to another account')
    await userEvent.click(canvas.getByRole('button', { name: 'Cancel' }))
    await expect(onCancel).toHaveBeenCalled()
  },
}

/** While the core works, the field and both buttons are disabled. */
export const Busy: Story = {
  args: { isBusy: true },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(getField(canvas)).toBeDisabled()
    await expect(canvas.getByRole('button', { name: 'Restoring…' })).toBeDisabled()
    await expect(canvas.getByRole('button', { name: 'Cancel' })).toBeDisabled()
  },
}
