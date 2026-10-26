/**
 * The conversation another plugin mounts into a panel of its own, here a panel the size of the
 * review's, with a made-up agent that answers every question the same way.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import type { AgentPort } from '@numen/plugin-sdk'
import ConversationView from './ConversationView.vue'
import { frameStory } from '../fixtures/frame'

const agent: AgentPort = {
  async *ask(query) {
    yield {
      kind: 'said',
      text: `Leaf mould is compost made of fallen leaves alone. You asked: ${query.question}`,
    }
    yield { kind: 'stopped', error: '' }
  },
  async finish() {},
}

const meta = {
  title: 'Agent/Conversation view',
  component: ConversationView,
  decorators: [frameStory],
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof ConversationView>

export default meta
type Story = StoryObj<typeof meta>

/** Mounted for a card, before anything is asked. */
export const Empty: Story = {
  args: { context: { agent, focus: 'decks/Words.md', placeholder: 'Ask about this card' } },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByText('Nothing said yet')).toBeVisible()
  },
}

/** A question sent, and the answer the agent gave. */
export const Answered: Story = {
  args: { context: { agent, focus: 'decks/Words.md', placeholder: 'Ask about this card' } },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(canvas.getByRole('textbox', { name: 'Ask about this card' }), 'why')
    await userEvent.click(canvas.getByRole('button', { name: 'Send' }))
    // An answer arriving is drawn a word to an element, so it is read off the thread whole.
    await waitFor(() =>
      expect(canvasElement.querySelector('.thread')?.textContent).toMatch(
        /compost\s+made\s+of\s+fallen\s+leaves/,
      ),
    )
  },
}

/** Mounted where the agent cannot be reached: the reason stands where the answers would. */
export const Unreachable: Story = {
  args: {
    context: {
      agent,
      focus: 'decks/Words.md',
      placeholder: 'Ask about this card',
      getUnreachable: () => 'The agent could not be reached.',
    },
  },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByText('The agent could not be reached.')).toBeVisible()
  },
}
