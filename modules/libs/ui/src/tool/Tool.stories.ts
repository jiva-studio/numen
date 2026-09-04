/**
 * A tool in hand, and a tool put down. What the thread shows while the agent
 * works.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import Tool from './Tool.vue'

const meta = {
  title: 'Agent/Tool',
  component: Tool,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'One line about work: what the agent reached for, what it reached ' +
          'for it about, and whether it still has it in hand. Quieter than ' +
          'what is said, so a glance down the thread reads the conversation ' +
          'and finds the doing.',
      },
    },
  },
  args: { tool: 'note_search', about: '', working: false },
} satisfies Meta<typeof Tool>

export default meta
type Story = StoryObj<typeof meta>

export const Playground: Story = {}

/** Still in hand: the dots say the answer has not come back yet. */
export const Working: Story = {
  args: { tool: 'note_neighbourhood', about: 'Harmonic oscillator', working: true },
}

/** Put down, with what it was about. */
export const Done: Story = {
  args: { tool: 'note_rewrite', about: 'Simple pendulum' },
}

/** One after another, as a thread collects them. */
export const InAThread: Story = {
  render: () => ({
    components: { Tool },
    template: `
      <div class="numen flex w-[320px] flex-col gap-1 text-ink">
        <Tool tool="note_search" about="entropy" />
        <Tool tool="note_read" about="Entropy" />
        <Tool tool="link_add" about="Entropy → Thermodynamics" />
        <Tool tool="note_rewrite" about="Simple pendulum" working />
      </div>
    `,
  }),
}

/** A name with nowhere to break, and one that runs long. */
export const AwkwardText: Story = {
  args: {
    tool: 'note_neighbourhood',
    about: 'a note whose title runs on well past the width of the panel it is drawn in',
  },
}

/**
 * A call carrying the text of a note is written for minutes, and how much has
 * arrived is the only thing that moves while it is.
 */
export const BeingWritten: Story = {
  args: {
    tool: 'Create a note',
    about: "Bram Doyle's warning",
    aside: '12 015 characters',
    working: true,
  },
  play: async ({ canvasElement }) => {
    const line = canvasElement.querySelector('.tool')
    await expect(line).toHaveTextContent("Bram Doyle's warning")
    await expect(line).toHaveTextContent('12 015 characters')
  },
}
