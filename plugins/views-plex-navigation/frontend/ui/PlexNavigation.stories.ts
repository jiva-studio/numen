/** The Back and Forward buttons of a board. Each story is run in a browser by `@storybook/addon-vitest`. */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import PlexNavigation from './PlexNavigation.vue'

const meta = {
  title: 'Plex/Navigation',
  component: PlexNavigation,
  args: { canBack: true, canForward: true, onBack: fn(), onForward: fn() },
  render: (args) => ({
    components: { PlexNavigation },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="padding:1rem;background:var(--numen-surface)">
        <PlexNavigation v-bind="args" />
      </div>
    `,
  }),
} satisfies Meta<typeof PlexNavigation>

export default meta
type Story = StoryObj<typeof meta>

/** Both directions are open; pressing a button reports it. */
export const GoesBackAndForward: Story = {
  tags: ['screenshot'],
  parameters: {
    screenshot: [
      { name: 'navigation', width: 240, height: 96, drawn: 500 },
    ],
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: 'Back' }))
    await userEvent.click(await canvas.findByRole('button', { name: 'Forward' }))
    await expect(args.onBack).toHaveBeenCalledTimes(1)
    await expect(args.onForward).toHaveBeenCalledTimes(1)
  },
}

/** A first visit has nothing behind it or ahead of it. */
export const NoHistory: Story = {
  args: { canBack: false, canForward: false },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByRole('button', { name: 'Back' })).toBeDisabled()
    await expect(await canvas.findByRole('button', { name: 'Forward' })).toBeDisabled()
  },
}
