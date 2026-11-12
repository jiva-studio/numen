/**
 * The indicator a pane displays while its content is loading.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import LoadingIndicator from './LoadingIndicator.vue'

const meta = {
  title: 'LoadingIndicator',
  component: LoadingIndicator,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'A tab drawing nothing yet and a tab holding nothing look the ' +
          'same, and only this tells them apart. Every pane that waits says ' +
          'so the same way: one ring, one word, one size, in the middle of ' +
          'whatever room it was given.',
      },
    },
  },
} satisfies Meta<typeof LoadingIndicator>

export default meta
type Story = StoryObj<typeof meta>

/** In the space of a pane. */
export const Playground: Story = {
  render: () => ({
    components: { LoadingIndicator },
    template: `
      <div class="numen" style="block-size:320px;background:var(--numen-surface)">
        <LoadingIndicator :delay="0" />
      </div>
    `,
  }),
}

/**
 * A pane whose content arrives within the delay never displays the ring: the pane stays empty until
 * the delay has passed.
 */
export const EmptyWithinItsDelay: Story = {
  render: () => ({
    components: { LoadingIndicator },
    template: `
      <div class="numen" style="block-size:120px;background:var(--numen-surface)">
        <LoadingIndicator :delay="300" />
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const pane = within(canvasElement).getByRole('status')

    await expect(pane.querySelector('.spinner')).toBeNull()
    await expect(pane).toHaveTextContent('')
    await new Promise((resolve) => setTimeout(resolve, 400))
    await expect(pane.querySelector('.spinner')).not.toBeNull()
    await expect(pane).toHaveTextContent('Loading…')
  },
}
