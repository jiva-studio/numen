import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import SupporterBadge from './SupporterBadge.vue'
import { WORDS } from '../words'
import '../styles.css'

const meta = {
  title: 'Supporter/Badge',
  component: SupporterBadge,
  parameters: { layout: 'fullscreen' },
  args: { isActive: true },
  render: (args) => ({
    components: { SupporterBadge },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="display:flex;align-items:flex-start;justify-content:flex-end;block-size:100vh;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
        <header style="display:flex;align-items:center;justify-content:flex-end;position:relative;inline-size:100%;block-size:2.25rem;background:var(--numen-surface);border-bottom:1px solid var(--numen-rule);font-size:var(--numen-text-1)">
          <SupporterBadge v-bind="args" />
        </header>
      </div>
    `,
  }),
} satisfies Meta<typeof SupporterBadge>

export default meta
type Story = StoryObj<typeof meta>

export const Active: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const badge = await canvas.findByRole('status', { name: WORDS.thankYou })
    await expect(badge).toBeVisible()
    await expect(badge).toHaveTextContent(WORDS.badgeLabel)
  },
}

export const Inactive: Story = {
  args: { isActive: false },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.queryByRole('status')).toBeNull()
  },
}
