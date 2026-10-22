/**
 * An agent tab as the plugin's own kind opens it: the talk so far, the composer
 * under it, and the window and agent behind it made up for the story.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { defineComponent, h, ref } from 'vue'
import type { Conversation, Turn } from '@numen/ui'
import AgentTab from './AgentTab.vue'
import { createAgentKind } from '../kind'
import { createHost } from '../fixtures'
import { useAgentConversation } from '../model/useAgentConversation'

/** An agent tab opened by the kind over a talk that answers with the transcript so far. */
const createTab = (turn: readonly Turn[]) =>
  defineComponent({
    name: 'AgentKindTab',
    setup() {
      const request: string[] = []
      const { kind } = createAgentKind(createHost().host, (deps) => {
        const conversation: Conversation = {
          turns: ref([...turn]),
          isWorking: ref(false),
          ask: async (text) => void request.push(text),
          getSourceLocation: () => null,
          stop: () => {},
          finish: () => {},
        }
        return useAgentConversation(conversation, deps)
      })
      const state = kind.open!('', 'agent:story')
      return () =>
        h('div', { style: { width: '100vw', height: '100vh' } }, [h(AgentTab, { state })])
    },
  })

const meta = {
  title: 'Agent/Tab',
  parameters: { layout: 'fullscreen' },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = {
  render: () => ({ components: { Tab: createTab([]) }, template: '<Tab />' }),
  play: async ({ canvasElement }) => {
    await waitFor(() =>
      expect(within(canvasElement).getByPlaceholderText('Ask about the vault')).toBeTruthy(),
    )
  },
}

/** A question and its answer, drawn one above the other. */
export const Answered: Story = {
  render: () => ({
    components: {
      Tab: createTab([
        { id: 'a', voice: 'asked', text: 'What is entropy?' },
        {
          id: 'b',
          voice: 'answered',
          text: 'A measure of how many ways a system can be arranged.',
        },
      ]),
    },
    template: '<Tab />',
  }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(() => expect(canvas.getByText('What is entropy?')).toBeTruthy())
  },
}

/** Typing into the composer fills it with what was typed. */
export const Typing: Story = {
  render: () => ({ components: { Tab: createTab([]) }, template: '<Tab />' }),
  play: async ({ canvasElement }) => {
    const field = await within(canvasElement).findByPlaceholderText('Ask about the vault')
    await userEvent.type(field, 'hello')
    await waitFor(() => expect((field as HTMLTextAreaElement).value).toBe('hello'))
  },
}
