/**
 * How many cards are due, as one figure in a pill. Also the test corpus: each
 * story is run in a browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import DueCount from './DueCount.vue'
import { Button } from '@numen/ui'
import { DUE_WORDS } from './due'

interface Knobs {
  due: number | null
  isBare: boolean
  isOnButton: boolean
}

const renderPills = (args: Knobs) => ({
  components: { DueCount, Button },
  setup: () => ({ args }),
  template: `
    <div class="numen" style="display:flex;align-items:center;gap:24px;padding:32px;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
      <span style="display:inline-flex;align-items:center;gap:8px">
        Sanskrit
        <DueCount :due="args.due" :is-bare="args.isBare" />
      </span>
      <Button variant="solid">
        Review
        <DueCount :due="args.due" :is-bare="args.isBare" is-on-button />
      </Button>
    </div>
  `,
})

const meta: Meta<Knobs> = {
  title: 'Flash Cards/Due count',
  component: DueCount,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'One figure and the word for what it counts. A figure not worked ' +
          'out yet is drawn as the shape it will be, in the same box, so the ' +
          'row it is in does not move when it lands.',
      },
    },
  },
  argTypes: {
    due: { control: 'number' },
    isBare: { control: 'boolean' },
    isOnButton: { control: 'boolean' },
  },
  args: { due: 12, isBare: false, isOnButton: false },
  render: renderPills,
}

export default meta
type Story = StoryObj<Knobs>

/** A figure that has landed, on the surface and on a filled button. */
export const Playground: Story = {}

/** The figure alone, which is what a long list of rows draws. */
export const Bare: Story = {
  args: { isBare: true },
  play: async ({ canvasElement }) => {
    const [pill] = within(canvasElement).getAllByRole('status')
    await expect(pill?.textContent?.trim()).toBe('12')

    // Drawn as the figure alone, and still read out as what it counts.
    await expect(pill?.getAttribute('aria-label')).toBe(DUE_WORDS.formatDue(12))
  },
}

/**
 * Still being counted: the pill keeps the shape the figure will take, and
 * announces it to screen readers.
 */
export const StillCounting: Story = {
  args: { due: null },
  play: async ({ canvasElement }) => {
    const [pill] = within(canvasElement).getAllByRole('status')
    await expect(pill?.getAttribute('aria-label')).toBe(DUE_WORDS.counting)
    await expect(pill?.textContent?.trim()).toBe('')
  },
}
