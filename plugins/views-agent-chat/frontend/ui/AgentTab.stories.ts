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
import type { HostOptions } from '../fixtures'
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

/** The notes of the vault the answers in the window link to. */
const VAULT: HostOptions = {
  focus: { path: 'Physics/Entropy.md', title: 'Entropy' },
  notes: {
    'name://Entropy': 'Physics/Entropy.md',
    "name://Landauer's principle": "Computation/Landauer's principle.md",
    'name://Boltzmann 1877': 'Reading/Boltzmann 1877.pdf',
  },
}

const createPaneTab = (createState: (deps: AgentTabDeps) => AgentTabState) =>
  defineComponent({
    name: 'AgentPaneTab',
    setup() {
      const { kind } = createAgentKind(createHost(VAULT).host, createState, AgentTab)
      const state = kind.open!('', 'agent:story')
      return () => h('div', { style: { width: '100%', height: '100%' } }, [h(AgentTab, { state })])
    },
  })

const DEMON: readonly Turn[] = [
  { id: 'a', voice: 'prompt', text: "Why doesn't Maxwell's demon break the second law?" },
  {
    id: 'b',
    voice: 'response',
    text:
      'Because the demon has to remember which molecule is which, and the memory is finite. ' +
      "[[Landauer's principle]] puts the price on clearing it: at least kT ln 2 of heat for " +
      'every bit erased, which is exactly what the sorting saved.\n\n' +
      'Your note [[Entropy]] makes the same point under *Where the two meet*, and ' +
      '[[Boltzmann 1877]] counts the arrangements on p. 373.',
  },
]

/** The agent in a pane of the window beside the note it was asked about, its answer linking to the vault. */
export const InTheWindow: Story = {
  tags: ['screenshot'],
  parameters: {
    frame: {
      place: 'tab',
      tab: { title: 'Agent', icon: 'agent' },
      note: 'before',
      share: 0.45,
      selected: 'Physics/Entropy.md',
    },
    screenshot: [{ name: 'agent' }],
  },
  render: () => ({
    components: { Tab: createPaneTab(createRecordedState(DEMON)) },
    template: '<Tab />',
  }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(() => expect(canvas.getByText(byWholeText(DEMON[0]!.text))).toBeVisible())
  },
}

const AGENT_ROWS = [
  {
    connectionId: 'c-claude',
    connectionName: 'Claude Code',
    options: { model: 'sonnet-5.5' },
    fields: [
      { key: 'model', title: 'Model', optionsFrom: 'llm.models', isRequired: false },
      { key: 'reasoning', title: 'Reasoning', optionsFrom: 'llm.reasoning', isRequired: false },
    ],
  },
  {
    connectionId: 'c-antigravity',
    connectionName: 'Antigravity',
    options: {},
    fields: [{ key: 'model', title: 'Model', optionsFrom: 'llm.models', isRequired: false }],
  },
]

const AGENT_OPTIONS: Readonly<Record<string, readonly string[]>> = {
  'c-claude:model': ['Opus 5.5', 'Sonnet 5.5', 'Haiku 5.5'],
  'c-claude:reasoning': ['Low', 'Medium', 'High'],
  'c-antigravity:model': ['Gemini 4 Argon', 'Gemini 3.8 Flash'],
}

const createChoosingState =
  (rows: typeof AGENT_ROWS) =>
  (deps: AgentTabDeps): AgentTabState => {
  const { host } = createAgentConnections(
    { frames: [] },
    {
      rows,
      listOptions: (connectionId, field) =>
        (AGENT_OPTIONS[`${connectionId}:${field}`] ?? []).map((title) => ({
          value: title.toLowerCase().replace(/ /g, '-'),
          title,
          group: '',
        })),
    },
  )
  const list = createAgentList(host, WORDS)
  list.reload().catch(() => undefined)
  const choice = useAgentChoice(host, list)
  const conversation: Conversation = {
    turns: ref([...DEMON]),
    isAnswerPending: ref(false),
    ask: async () => undefined,
    getSourcePath: () => null,
    stop: () => undefined,
  }
  return useAgentConversation(conversation, deps, choice)
}

/** The rows of the agents with Antigravity first, on a model of its own. */
const ANTIGRAVITY_FIRST: typeof AGENT_ROWS = [
  { ...AGENT_ROWS[1]!, options: { model: 'gemini-4-argon' } },
  AGENT_ROWS[0]!,
]

const renderChoosing = (rows: typeof AGENT_ROWS) => () => ({
  components: { Tab: createPaneTab(createChoosingState(rows)) },
  template: '<Tab />',
})

const openModels =
  (picker: string): NonNullable<Story['play']> =>
  async ({ canvasElement }) => {
    await userEvent.click(await within(canvasElement).findByRole('button', { name: picker }))
    const list = await within(document.body).findByRole('listbox', { name: 'Model' })
    await waitFor(() => expect(within(list).getByRole('option', { name: 'Opus 5.5' })).toBeVisible())
  }

const AGENT_FRAME = {
  place: 'tab',
  tab: { title: 'Agent', icon: 'agent' },
  note: 'before',
  share: 0.45,
}

/** The models of the agents set up, opened from the composer of the agent in the window on Claude Code. */
export const ChoosingAModel: Story = {
  tags: ['screenshot'],
  parameters: {
    frame: AGENT_FRAME,
    screenshot: [{ name: 'models', plugin: 'agents-claude' }],
  },
  render: renderChoosing(AGENT_ROWS),
  play: openModels('Claude Code · Sonnet 5.5'),
}

/** The models of the agents set up, opened from the composer of the agent in the window on Antigravity. */
export const ChoosingAnAntigravityModel: Story = {
  tags: ['screenshot'],
  parameters: {
    frame: AGENT_FRAME,
    screenshot: [{ name: 'models', plugin: 'agents-antigravity' }],
  },
  render: renderChoosing(ANTIGRAVITY_FIRST),
  play: openModels('Antigravity · Gemini 4 Argon'),
}
