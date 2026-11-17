import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import { Tooltip } from '.'
import { DARK, expectDark } from '@/shared/fixtures/theme'

interface Knobs {
  text: string
}

const meta: Meta<Knobs> = {
  title: 'Controls/Tooltip',
  parameters: { layout: 'fullscreen' },
  argTypes: { text: { control: 'text' } },
  args: { text: 'Port, Sea, Quay' },
  render: (args) => ({
    components: { Tooltip },
    setup: () => ({ args }),
    template: `<Tooltip :at="{ x: 120, y: 96, width: 16, height: 16 }">{{ args.text }}</Tooltip>`,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** The label of one item. */
export const Label: Story = {
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByRole('tooltip')).toHaveTextContent('Port, Sea, Quay')
  },
}

export const LabelInTheDarkTheme: Story = {
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    await expect(within(canvasElement).getByRole('tooltip')).toHaveTextContent('Port, Sea, Quay')
  },
}

/** A label of far too many words wraps inside the viewport. */
export const FarTooManyWords: Story = {
  args: { text: Array.from({ length: 60 }, (_, index) => `Tag ${index + 1}`).join(', ') },
  play: async ({ canvasElement }) => {
    const tooltip = within(canvasElement).getByRole('tooltip')
    await expect(tooltip).toBeVisible()
    const box = tooltip.getBoundingClientRect()
    await expect(box.left).toBeGreaterThanOrEqual(0)
    await expect(box.right).toBeLessThanOrEqual(window.innerWidth)
  },
}
