/**
 * The conversation another plugin mounts into a panel of its own, here a panel the size of the
 * review's, with made-up connections that answer every question the same way.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import ConversationView from './ConversationView.vue'
import { frameStory } from '../fixtures/frame'
import { createAgentConnections, failWith, say } from '../fixtures'

const { host: connections } = createAgentConnections((request) => ({
  frames: [say(`Leaf mould is compost made of fallen leaves alone. You asked: ${request.prompt}`)],
}))

const { host: noAgent } = createAgentConnections({
  end: failWith('no_connection', 'no connection serves acp.task'),
})

const meta = {
  title: 'Agent/Conversation view',
  component: ConversationView,
  decorators: [frameStory],
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof ConversationView>

export default meta
type Story = StoryObj<typeof meta>

/** Mounted for a card, before any message is sent. */
export const Empty: Story = {
  args: { context: { connections, placeholder: 'Ask about this card' } },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByText('No messages yet')).toBeVisible()
  },
}

/** A question sent, and the answer the agent gave. */
export const Answered: Story = {
  args: { context: { connections, placeholder: 'Ask about this card' } },
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

/** Mounted by a plugin with no agent row: the question fails and says no agent is set up. */
export const NoAgent: Story = {
  args: { context: { connections: noAgent, placeholder: 'Ask about this card' } },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(canvas.getByRole('textbox', { name: 'Ask about this card' }), 'why')
    await userEvent.click(canvas.getByRole('button', { name: 'Send' }))
    await waitFor(() => expect(canvas.getByText('No agent is set up.')).toBeVisible())
  },
}
