/**
 * The pill that names the model an agent works with, and the searchable list it opens: the
 * models of each agent under the agent's name, narrowed by the words typed.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import AgentModelSelector from './AgentModelSelector.vue'
import type { AgentModelOption } from '../../model/options'

const option = (agentId: string, agentTitle: string, modelId: string, modelTitle: string) => ({
  agentId,
  agentTitle,
  modelId,
  modelTitle,
  isAvailable: true,
})

const OPTIONS: readonly AgentModelOption[] = [
  option('antigravity', 'Antigravity', 'argon', 'Gemini 4 Argon'),
  option('antigravity', 'Antigravity', 'flash', 'Gemini 3.8 Flash'),
  option('antigravity', 'Antigravity', 'opus', 'Claude Opus 5.5'),
  option('claude', 'Claude Code', 'default', 'Default (recommended)'),
  option('claude', 'Claude Code', 'opus', 'Opus 5.5'),
  option('claude', 'Claude Code', 'sonnet', 'Sonnet 5.5'),
  option('claude', 'Claude Code', 'haiku', 'Haiku 5.5'),
]

const meta = {
  title: 'Chat/AgentModelSelector',
  component: AgentModelSelector,
  parameters: { layout: 'centered' },
  args: {
    options: OPTIONS,
    selectedAgentId: 'claude',
    selectedModelId: 'sonnet',
    onSelect: fn(),
  },
  decorators: [
    (story) => ({
      components: { story },
      template: '<div class="numen" style="padding:12rem 8rem 2rem"><story /></div>',
    }),
  ],
} satisfies Meta<typeof AgentModelSelector>

export default meta
type Story = StoryObj<typeof meta>

const open: NonNullable<Story['play']> = async ({ canvasElement }) => {
  await userEvent.click(within(canvasElement).getByRole('button'))
}

/** The models under the names of their agents, the one in force marked. */
export const Opened: Story = {
  play: async (context) => {
    await open(context)
    const list = await within(document.body).findByRole('listbox', { name: 'Model' })
    await expect(within(list).getAllByRole('option')).toHaveLength(OPTIONS.length)
    await expect(within(list).getByRole('option', { name: 'Sonnet 5.5' })).toHaveAttribute(
      'aria-current',
      'true',
    )
  },
}

/** Words typed narrow the models, and Enter chooses the one the keyboard is on. */
export const Searched: Story = {
  play: async (context) => {
    await open(context)
    const body = within(document.body)
    await userEvent.type(await body.findByRole('combobox', { name: 'Search models' }), 'haiku')
    await waitFor(() => expect(body.getAllByRole('option')).toHaveLength(1))
    await userEvent.keyboard('{Enter}')
    await expect(context.args.onSelect).toHaveBeenCalledWith(
      expect.objectContaining({ modelId: 'haiku' }),
    )
  },
}
