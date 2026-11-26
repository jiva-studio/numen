/**
 * An agent tab as the plugin's own kind opens it: the talk so far, the composer
 * under it, and the window behind it made up for the story.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { defineComponent, h, ref } from 'vue'
import { useConversation, type Conversation } from '../thread'
import type { AgentTabDeps, Turn } from '../types'
import AgentTab from './AgentTab.vue'
import { createAgentKind } from '../model/kind'
import { byWholeText, createAgentConnections, createHost, say } from '../fixtures'
import { useAgentConversation, type AgentTabState } from '../model/useAgentConversation'
import { createAgentList, useAgentChoice } from '../model/agents'
import { WORDS } from '../words'

const createRecordedState =
  (turn: readonly Turn[]) =>
  (deps: AgentTabDeps): AgentTabState => {
    const conversation: Conversation = {
      turns: ref([...turn]),
      isAnswerPending: ref(false),
      ask: async () => undefined,
      getSourcePath: () => null,
      stop: () => undefined,
    }
    return useAgentConversation(conversation, deps)
  }

const createTab = (createState: (deps: AgentTabDeps) => AgentTabState) =>
  defineComponent({
    name: 'AgentKindTab',
    setup() {
      const { kind } = createAgentKind(createHost().host, createState, AgentTab)
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
  render: () => ({ components: { Tab: createTab(createRecordedState([])) }, template: '<Tab />' }),
  play: async ({ canvasElement }) => {
    await waitFor(() =>
      expect(within(canvasElement).getByPlaceholderText('Ask about the vault')).toBeTruthy(),
    )
  },
}

/** A question and its answer, drawn one above the other. */
export const Answered: Story = {
  tags: ['screenshot'],
  parameters: {
    screenshot: [{ name: 'agent', width: 760, height: 360 }],
  },
  render: () => ({
    components: {
      Tab: createTab(
        createRecordedState([
          { id: 'a', voice: 'prompt', text: 'What is entropy?' },
          {
            id: 'b',
            voice: 'response',
            text: 'A measure of how many ways a system can be arranged.',
          },
        ]),
      ),
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
  render: () => ({ components: { Tab: createTab(createRecordedState([])) }, template: '<Tab />' }),
  play: async ({ canvasElement }) => {
    const field = await within(canvasElement).findByPlaceholderText('Ask about the vault')
    await userEvent.type(field, 'hello')
    await waitFor(() => expect((field as HTMLTextAreaElement).value).toBe('hello'))
  },
}

/** A question asked while the plugin has no agent row: the failed turn offers the settings. */
export const NoAgent: Story = {
  render: () => ({
    components: {
      Tab: createTab(
        createRecordedState([
          { id: 'a', voice: 'prompt', text: 'Which pear trees flowered first?' },
          {
            id: 'b',
            voice: 'response',
            text: 'No agent is set up.',
            state: 'failed',
            offersSettings: true,
          },
        ]),
      ),
    },
    template: '<Tab />',
  }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(() => expect(canvas.getByText(byWholeText('No agent is set up.'))).toBeVisible())
    await expect(canvas.getByRole('button', { name: 'Open settings' })).toBeVisible()
  },
}

const ORCHARD_ROWS = [
  {
    connectionId: 'c-orchard',
    connectionName: 'Orchard agent',
    options: { model: 'pear-small' },
    fields: [
      { key: 'model', title: 'Model', optionsFrom: 'llm.models', isRequired: false },
      { key: 'reasoning', title: 'Reasoning', optionsFrom: 'llm.reasoning', isRequired: false },
    ],
  },
  {
    connectionId: 'c-meadow',
    connectionName: 'Meadow agent',
    options: {},
    fields: [{ key: 'model', title: 'Model', optionsFrom: 'llm.models', isRequired: false }],
  },
]

const ORCHARD_OPTIONS: Readonly<Record<string, readonly string[]>> = {
  'c-orchard:model': ['Pear Small', 'Pear Large'],
  'c-meadow:model': ['Clover'],
  'c-orchard:reasoning': ['Low', 'High'],
}

const createAgentsState = (deps: AgentTabDeps): AgentTabState => {
  const { host } = createAgentConnections(
    { frames: [say('Twelve pear trees.')] },
    {
      rows: ORCHARD_ROWS,
      listOptions: (connectionId, field) =>
        (ORCHARD_OPTIONS[`${connectionId}:${field}`] ?? []).map((title) => ({
          value: title.toLowerCase().replace(' ', '-'),
          title,
          group: '',
        })),
    },
  )
  const list = createAgentList(host, WORDS)
  list.reload().catch(() => undefined)
  const choice = useAgentChoice(host, list)
  const conversation = useConversation({
    connections: host,
    words: WORDS,
    conversation: 'conversation:story',
    getPreferredRow: choice.getPreferredRow,
  })
  return useAgentConversation(conversation, deps, choice)
}

/** Select a reasoning level, then switch to a model without reasoning. */
export const Agents: Story = {
  render: () => ({ components: { Tab: createTab(createAgentsState) }, template: '<Tab />' }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const body = within(document.body)
    const reasoning = await canvas.findByRole('button', { name: 'Reasoning' })
    await userEvent.click(reasoning)
    const high = await body.findByRole('menuitemradio', { name: 'High' })
    await userEvent.hover(high)
    await expect(high).toBeVisible()
    await userEvent.click(high)
    await expect(reasoning).toHaveTextContent('High')

    const picker = await canvas.findByRole('button', { name: 'Orchard agent · Pear Small' })

    await userEvent.click(picker)
    await waitFor(() =>
      expect(
        within(body.getByRole('listbox', { name: 'Model' })).getAllByRole('option'),
      ).toHaveLength(4),
    )
    await waitFor(() => expect(canvas.getByRole('button', { name: 'Reasoning' })).toBeVisible())
    await expect(picker).toHaveAccessibleName('Orchard agent · Pear Small')
    const list = body.getByRole('listbox', { name: 'Model' })
    await userEvent.click(within(list).getByRole('option', { name: 'Clover' }))

    await waitFor(() =>
      expect(canvas.getByRole('button', { name: 'Meadow agent · Clover' })).toBeVisible(),
    )
    await expect(canvas.queryByRole('button', { name: 'Reasoning' })).toBeNull()
  },
}
