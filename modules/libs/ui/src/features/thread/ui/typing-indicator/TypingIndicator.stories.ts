/**
 * The dots take the colour of whatever they stand on, so they are shown on
 * the three grounds they are used on.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import TypingIndicator from './TypingIndicator.vue'

const meta = {
  title: 'Agent/TypingIndicator',
  component: TypingIndicator,
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
} satisfies Meta<typeof TypingIndicator>

export default meta
type Story = StoryObj<typeof meta>

export const Playground: Story = {
  render: (args) => ({
    components: { TypingIndicator },
    setup: () => ({ args }),
    template: `
      <div class="numen flex items-center gap-4 text-ink">
        <TypingIndicator v-bind="args" />
        <span class="flex size-action items-center justify-center rounded-pill bg-bubble text-bubble-ink">
          <TypingIndicator v-bind="args" />
        </span>
        <span class="flex size-action items-center justify-center rounded-pill bg-accent text-accent-ink">
          <TypingIndicator v-bind="args" />
        </span>
      </div>
    `,
  }),
}
