/** A fingerprint at the widths it is shown in. */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import Fingerprint from './Fingerprint.vue'

const VALUE = 'F9D2 40ED 59AF 4B0D 55BB 09D1 D750 C8AD'

const meta = {
  title: 'Application/Fingerprint',
  component: Fingerprint,
  args: { value: VALUE },
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Fingerprint>

export default meta
type Story = StoryObj<typeof meta>

const inWidth = (width: string) => ({
  components: { Fingerprint },
  setup: () => ({ value: VALUE, width }),
  template: `<div :style="{ inlineSize: width, padding: '1rem' }"><Fingerprint :value="value" /></div>`,
})

/** Room for all of it: it reads in full. */
export const Wide: Story = {
  render: () => inWidth('24rem'),
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByLabelText(VALUE)).toBeVisible()
  },
}

/** Less room: the middle is an ellipsis and the last two groups remain. */
export const Narrow: Story = { render: () => inWidth('11rem') }

/** Hardly any room: the tail alone remains. */
export const Tight: Story = { render: () => inWidth('6rem') }
