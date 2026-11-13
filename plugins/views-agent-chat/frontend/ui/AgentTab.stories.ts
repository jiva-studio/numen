/**
 * An agent tab as the plugin's own kind opens it: the talk so far, the composer
 * under it, and the window behind it made up for the story.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { defineComponent, h, ref } from 'vue'
import { useConversation, type Conversation } from '../thread'
import type { Turn } from '../types'
import AgentTab from './AgentTab.vue'
import { createAgentKind } from '../model/kind'
import { byWholeText, createAgentConnections, createHost, say } from '../fixtures'
import { useAgentConversation } from '../model/useAgentConversation'
import { createAgentList, useAgentChoice } from '../model/agents'
import { WORDS } from '../words'

const createTab = (turn: readonly Turn[]) =>
  defineComponent({
    name: 'AgentKindTab',
    setup() {
      const request: string[] = []
      const { kind } = createAgentKind(
        createHost().host,
        (deps) => {
          const conversation: Conversation = {
            turns: ref([...turn]),
            isAnswerPending: ref(false),
            ask: async (text) => {
              request.push(text)
            },
            getSourcePath: () => null,
            stop: () => undefined,
          }
          return useAgentConversation(conversation, deps)
        },
        AgentTab,
      )
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
        { id: 'a', voice: 'prompt', text: 'What is entropy?' },
        {
          id: 'b',
          voice: 'response',
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

/** A question asked while the plugin has no agent row: the failed turn offers the settings. */
export const NoAgent: Story = {
  render: () => ({
    components: {
      Tab: createTab([
        { id: 'a', voice: 'prompt', text: 'Which pear trees flowered first?' },
        {
          id: 'b',
          voice: 'response',
          text: 'No agent is set up.',
          state: 'failed',
          offersSettings: true,
        },
      ]),
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
      { key: 'model', title: 'Model', optionsFrom: 'llm.models' },
      { key: 'reasoning', title: 'Reasoning', optionsFrom: 'llm.reasoning' },
    ],
  },
  {
    connectionId: 'c-meadow',
    connectionName: 'Meadow agent',
    options: {},
    fields: [{ key: 'model', title: 'Model', optionsFrom: 'llm.models' }],
  },
]

const ORCHARD_OPTIONS: Readonly<Record<string, readonly string[]>> = {
  'c-orchard:model': ['Pear Small', 'Pear Large'],
  'c-meadow:model': ['Clover'],
  'c-orchard:reasoning': ['Low', 'High'],
}

const createAgentsTab = () =>
  defineComponent({
    name: 'AgentsTab',
    setup() {
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
      const list = createAgentList(host, WORDS.defaultModel)
      const { kind } = createAgentKind(
        createHost().host,
        (deps) => {
          const choice = useAgentChoice(host, list)
          const conversation = useConversation({
            connections: host,
            words: WORDS,
            conversation: 'conversation:story',
            getPreferred: choice.getPreferred,
          })
          return useAgentConversation(conversation, deps, choice)
        },
        AgentTab,
      )
      const state = kind.open!('', 'agent:story')
      const isListed = ref(false)
      list.reload().then(
        () => {
          isListed.value = true
        },
        () => undefined,
      )
      return () =>
        h('div', { style: { width: '100vw', height: '100vh' } }, [
          isListed.value ? h(AgentTab, { state }) : null,
        ])
    },
  })

/**
 * The agents of the plugin in the picker under the field: the one that answers named with its model, the
 * models of each agent under its name, and the reasoning levels of the model in force.
 */
export const Agents: Story = {
  render: () => ({ components: { Tab: createAgentsTab() }, template: '<Tab />' }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const picker = await canvas.findByRole('button', { name: 'Orchard agent · Pear Small' })
    await waitFor(() => expect(canvas.getByRole('button', { name: 'Reasoning' })).toBeVisible())

    await userEvent.click(picker)
    const body = within(document.body)
    const list = await body.findByRole('listbox', { name: 'Model' })
    await expect(within(list).getAllByRole('option')).toHaveLength(4)
    await userEvent.click(within(list).getByRole('option', { name: 'Clover' }))

    await waitFor(() =>
      expect(canvas.getByRole('button', { name: 'Meadow agent · Clover' })).toBeVisible(),
    )
    await expect(canvas.queryByRole('button', { name: 'Reasoning' })).toBeNull()
  },
}
