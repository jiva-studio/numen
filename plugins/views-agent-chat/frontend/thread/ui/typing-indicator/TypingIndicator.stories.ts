/**
 * The dots take the colour of whatever they stand on, so they are shown on
 * the three grounds they are used on.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { STORY_BOXES } from '../../../fixtures/frame'
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
    setup: () => ({ args, STORY_BOXES }),
    template: `
      <div :class="STORY_BOXES.indicators">
        <TypingIndicator v-bind="args" />
        <span :class="STORY_BOXES.bubble">
          <TypingIndicator v-bind="args" />
        </span>
        <span :class="STORY_BOXES.accent">
          <TypingIndicator v-bind="args" />
        </span>
      </div>
    `,
  }),
}
