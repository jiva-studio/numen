/** The questions about where the keys are kept on a computer without a system keychain, in a tab-sized frame. */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import type { Component } from 'vue'
import KeyFileDialog from './KeyFileDialog.vue'
import KeyFileRow from './KeyFileRow.vue'
import KeyFileUnlockDialog from './KeyFileUnlockDialog.vue'

const onChoose = fn()
const onUnlock = fn()
const onCancel = fn()

const FRAME =
  'position:relative;contain:strict;inline-size:48rem;block-size:34rem;background:var(--numen-surface)'

const meta: Meta = {
  title: 'Window/Identity/Keychain',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

const frame = (body: string, components: Record<string, Component>) => ({
  components,
  setup: () => ({ onChoose, onUnlock, onCancel }),
  template: `<div class="numen" style="${FRAME}">${body}</div>`,
})

const CHOOSE = '<KeyFileDialog :problem="\'\'" :is-busy="false" @choose="onChoose" @cancel="onCancel" />'

/** The passphrase is chosen first and Continue waits for a repeated passphrase. */
export const ChoosePassphrase: Story = {
  render: () => frame(CHOOSE, { KeyFileDialog }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('dialog', { name: 'Key storage' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Continue' })).toBeDisabled()
    await userEvent.type(canvas.getByLabelText('Passphrase'), 'moon river')
    await userEvent.type(canvas.getByLabelText('Repeat the passphrase'), 'moon river')
    await userEvent.click(canvas.getByRole('button', { name: 'Continue' }))
    await expect(onChoose).toHaveBeenCalledWith('passphrase', 'moon river')
  },
}

/** A repeat that differs is named. */
export const ChooseMismatch: Story = {
  render: () => frame(CHOOSE, { KeyFileDialog }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(canvas.getByLabelText('Passphrase'), 'moon river')
    await userEvent.type(canvas.getByLabelText('Repeat the passphrase'), 'moon')
    await expect(canvas.getByRole('alert')).toHaveTextContent('not the same')
    await expect(canvas.getByRole('button', { name: 'Continue' })).toBeDisabled()
  },
}

/** The plain file shows its warning and needs no passphrase. */
export const ChoosePlainFile: Story = {
  render: () => frame(CHOOSE, { KeyFileDialog }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('radio', { name: /Save in a file/ }))
    await expect(canvas.getByText(/Anyone with access to your files/)).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Continue' }))
    await expect(onChoose).toHaveBeenCalledWith('plaintext', '')
  },
}

/** The refusal of the core stays in the dialog. */
export const ChooseRefused: Story = {
  render: () =>
    frame(
      '<KeyFileDialog problem="The choice could not be saved. Try again." :is-busy="false" @choose="onChoose" @cancel="onCancel" />',
      { KeyFileDialog },
    ),
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByText(/could not be saved/)).toBeVisible()
  },
}

/** At the start the passphrase opens the keys. */
export const UnlockAtStart: Story = {
  render: () =>
    frame(
      '<KeyFileUnlockDialog problem="" :is-busy="false" @unlock="onUnlock" @cancel="onCancel" />',
      { KeyFileUnlockDialog },
    ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(canvas.getByLabelText('Passphrase'), 'moon river')
    await userEvent.click(canvas.getByRole('button', { name: 'Unlock' }))
    await expect(onUnlock).toHaveBeenCalledWith('moon river')
  },
}

/** A wrong passphrase is named. */
export const UnlockWrongPassphrase: Story = {
  render: () =>
    frame(
      '<KeyFileUnlockDialog problem="This passphrase does not open your keys. Try again." :is-busy="false" @unlock="onUnlock" @cancel="onCancel" />',
      { KeyFileUnlockDialog },
    ),
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByRole('alert')).toHaveTextContent('does not open')
  },
}

/** The row in Identity and licenses, with its action in the header. */
export const RowPassphrase: Story = {
  render: () =>
    frame(
      '<div style="padding:1rem"><KeyFileRow :state="{ isInUse: true, mode: \'passphrase\', isChoiceNeeded: false, isLocked: false, warning: \'\' }" /></div>',
      { KeyFileRow },
    ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByText('Where keys are kept')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Change' })).toBeVisible()
  },
}
