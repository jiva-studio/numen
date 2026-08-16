/**
 * The dots take the colour of whatever they stand on, so they are shown on
 * the three grounds they are used on.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import Dots from './Dots.vue'

const meta = {
  title: 'Generic/Dots',
  component: Dots,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Three dots rising in turn: something is being written. They take ' +
          'the colour of whatever they stand on, and where motion is turned ' +
          'down they say it by weight.',
      },
    },
  },
} satisfies Meta<typeof Dots>

export default meta
type Story = StoryObj<typeof meta>

export const Playground: Story = {
  render: (args) => ({
    components: { Dots },
    setup: () => ({ args }),
    template: `
      <div class="numen flex items-center gap-4 text-ink">
        <Dots v-bind="args" />
        <span class="flex size-action items-center justify-center rounded-pill bg-bubble text-bubble-ink">
          <Dots v-bind="args" />
        </span>
        <span class="flex size-action items-center justify-center rounded-pill bg-accent text-accent-ink">
          <Dots v-bind="args" />
        </span>
      </div>
    `,
  }),
}
