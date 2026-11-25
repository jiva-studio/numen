/**
 * The restore from sheets in a browser: the camera and two slots, no field. The stories open no camera.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import RestoreSheetsDialog from './RestoreSheetsDialog.vue'

interface Knobs {
  slots: (number | undefined)[]
  note: string
  problem: string
  isBusy: boolean
}

const camera = () => new Promise<MediaStream>(() => undefined)

const meta: Meta<Knobs> = {
  title: 'Window/Identity/RestoreSheets',
  component: RestoreSheetsDialog,
  parameters: { layout: 'fullscreen' },
  args: { slots: [undefined, undefined], note: '', problem: '', isBusy: false },
  render: (args) => ({
    components: { RestoreSheetsDialog },
    setup: () => ({ args, camera }),
    template: `
      <div class="numen" style="position:relative;contain:strict;box-sizing:border-box;inline-size:760px;block-size:640px;margin:1.5rem auto;border:1px solid var(--numen-rule);background:var(--numen-surface)">
        <RestoreSheetsDialog v-bind="args" :camera="camera" />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** Two waiting slots and no text field. */
export const Waiting: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body)
    await expect(canvas.getByRole('dialog', { name: 'Recovery kit: scan' })).toBeVisible()
    await expect(canvas.getByText('First sheet')).toBeVisible()
    await expect(canvas.queryByRole('textbox')).toBeNull()
  },
}

/** One sheet is scanned and the camera goes on. */
export const OneScanned: Story = {
  args: { slots: [3, undefined] },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body)
    await expect(canvas.getByText('Sheet 3')).toBeVisible()
    await expect(canvas.getByText('Second sheet')).toBeVisible()
  },
}

/** A refused sheet is named while the scan goes on. */
export const Refused: Story = {
  args: { slots: [3, undefined], note: 'Sheet 3 is scanned already. Scan another sheet.' },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement.ownerDocument.body).getByRole('alert')).toHaveTextContent(
      'scanned already',
    )
  },
}

/** The core restores after the second sheet. */
export const Restoring: Story = {
  args: { slots: [3, 1], isBusy: true },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement.ownerDocument.body).getByText('Restoring…')).toBeVisible()
  },
}

/** The core refuses: the reason, and the next sheet scanned starts over. */
export const Failed: Story = {
  args: { slots: [3, 1], problem: 'The sheets belong to another identity.' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body)
    await expect(canvas.getByRole('alert')).toHaveTextContent('another identity')
    await expect(canvas.queryByRole('button', { name: 'Try again' })).toBeNull()
  },
}
