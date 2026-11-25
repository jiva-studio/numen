/**
 * The ring that shows how far something has come.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import ProgressRing from './ProgressRing.vue'

const meta = {
  title: 'ProgressRing',
  component: ProgressRing,
  parameters: {
    docs: {
      description: {
        component:
          'A ring filled to the progress where it is known and a short turning arc where it is ' +
          'not. The large ring writes the percent inside.',
      },
    },
  },
  render: (args) => ({
    components: { ProgressRing },
    setup: () => ({ args }),
    template: `<div class="numen" style="padding:16px"><ProgressRing v-bind="args" /></div>`,
  }),
} satisfies Meta<typeof ProgressRing>

export default meta
type Story = StoryObj<typeof meta>

/** The progress is known, so the large ring is filled and writes the percent. */
export const Known: Story = {
  args: { value: 0.29, size: 'large', label: 'Downloading' },
  play: async ({ canvasElement }) => {
    const ring = within(canvasElement).getByRole('progressbar', { name: 'Downloading' })
    await expect(ring).toHaveAttribute('aria-valuenow', '29')
    await expect(ring).toHaveTextContent('29%')
  },
}

/** The progress is not known, so a short arc turns and no percent is written. */
export const Unknown: Story = {
  args: { size: 'large', label: 'Downloading' },
  play: async ({ canvasElement }) => {
    const ring = within(canvasElement).getByRole('progressbar', { name: 'Downloading' })
    await expect(ring).not.toHaveAttribute('aria-valuenow')
  },
}

/** The small ring beside a row has no label, so it is decorative. */
export const Small: Story = {
  args: { value: 0.5 },
}
