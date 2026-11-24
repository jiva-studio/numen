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
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('dialog', { name: 'Restore from sheets' })).toBeVisible()
    await expect(canvas.getByText('Sheet 1 waiting')).toBeVisible()
    await expect(canvas.queryByRole('textbox')).toBeNull()
  },
}

/** One sheet is scanned and the camera goes on. */
export const OneScanned: Story = {
  args: { slots: [3, undefined] },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByText('Sheet 3 scanned')).toBeVisible()
    await expect(canvas.getByText('Sheet 2 waiting')).toBeVisible()
  },
}

/** A refused sheet is named while the scan goes on. */
export const Refused: Story = {
  args: { slots: [3, undefined], note: 'Sheet 3 is scanned already. Scan another sheet.' },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByRole('alert')).toHaveTextContent('scanned already')
  },
}

/** The core restores after the second sheet. */
export const Restoring: Story = {
  args: { slots: [3, 1], isBusy: true },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByText('Restoring…')).toBeVisible()
  },
}

/** The core refuses: the reason and Try again. */
export const Failed: Story = {
  args: { slots: [3, 1], problem: 'The sheets belong to another account.' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('alert')).toHaveTextContent('another account')
    await expect(canvas.getByRole('button', { name: 'Try again' })).toBeVisible()
  },
}
