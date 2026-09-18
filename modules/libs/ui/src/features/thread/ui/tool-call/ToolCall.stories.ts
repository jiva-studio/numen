/**
 * A tool in hand, and a tool put down. What the thread shows while the agent
 * works.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import ToolCall from './ToolCall.vue'

const meta = {
  title: 'Agent/Tool call',
  component: ToolCall,
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
  args: { tool: 'note_search', subject: '', isWorking: false },
} satisfies Meta<typeof ToolCall>

export default meta
type Story = StoryObj<typeof meta>

export const Playground: Story = {}

/** Still in hand: the dots say the answer has not come back yet. */
export const Working: Story = {
  args: { tool: 'note_neighbourhood', subject: 'Harmonic oscillator', isWorking: true },
}

/** Put down, with what it was about. */
export const Done: Story = {
  args: { tool: 'note_rewrite', subject: 'Simple pendulum' },
}

/** One after another, as a thread collects them. */
export const InAThread: Story = {
  render: () => ({
    components: { ToolCall },
    template: `
      <div class="numen flex w-[320px] flex-col gap-1 text-ink">
        <ToolCall tool="note_search" about="entropy" />
        <ToolCall tool="note_read" about="Entropy" />
        <ToolCall tool="link_add" about="Entropy → Thermodynamics" />
        <ToolCall tool="note_rewrite" about="Simple pendulum" working />
      </div>
    `,
  }),
}

/** A name with nowhere to break, and one that runs long. */
export const AwkwardText: Story = {
  args: {
    tool: 'note_neighbourhood',
    subject: 'a note whose title runs on well past the width of the panel it is drawn in',
  },
}

/**
 * A call carrying the text of a note is written for minutes, and how much has
 * arrived is the only thing that moves while it is.
 */
export const BeingWritten: Story = {
  args: {
    tool: 'Create a note',
    subject: "Bram Doyle's warning",
    aside: '12 015 characters',
    isWorking: true,
  },
  play: async ({ canvasElement }) => {
    const line = canvasElement.querySelector('.tool-call')
    await expect(line).toHaveTextContent("Bram Doyle's warning")
    await expect(line).toHaveTextContent('12 015 characters')
  },
}
