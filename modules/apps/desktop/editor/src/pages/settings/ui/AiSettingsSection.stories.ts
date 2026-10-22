/**
 * The AI settings, and the agent setting of a plugin, drawn in a browser.
 *
 * What is asked here is what only a browser can answer: that the two lists of a
 * choice end at the one edge, that the question put to the person fits the
 * page at the width of a narrow window, and that a key typed is never drawn.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import { ref } from 'vue'
import AiSettingsSection from './AiSettingsSection.vue'
import PluginSettingsSection from './PluginSettingsSection.vue'
import type { PluginInfo } from '@/entities/settings'
import { getSettingAt } from '@/entities/settings'
import { createProgramModels } from '@/testing/installation'
import type { Installation } from '../types'

const createPlugin = (id: string, name: string, overrides: Partial<PluginInfo> = {}): PluginInfo => ({
  id,
  name,
  version: '1.0.0',
  description: '',
  isActive: false,
  settings: [],
  capabilities: [],
  secrets: [],
  storedSecrets: [],
  consent: 'none',
  ...overrides,
})

const PLUGINS: readonly PluginInfo[] = [
  createPlugin('sample.agent-a', 'Sample Agent A', {
    secrets: ['apiKey'],
    storedSecrets: ['apiKey'],
  }),
  createPlugin('sample.agent-b', 'Sample Agent B', {
    secrets: ['apiKey'],
  }),
  createPlugin('sample.asking', 'Sample Asking Plugin', {
    consent: 'asked',
    settings: [
      {
        key: 'agent',
        type: 'agent',
        title: 'Agent',
        description: 'The agent and the model that answer this plugin',
        default: undefined,
      },
    ],
  }),
]

const PROGRAMS = [
  {
    name: 'sample-a',
    title: 'Sample Agent A',
    isDefault: true,
    models: [
      { name: 'sample-quick', title: 'Sample quick' },
      { name: 'sample-careful', title: 'Sample careful' },
    ],
  },
  { name: 'sample-b', title: 'Sample Agent B' },
]

const FILE = {
  plugins: { 'sample.asking': { agent: { agent: 'sample-a', model: 'sample-careful' } } },
}

const installation: Installation = {
  getSetting: (path) => getSettingAt(FILE, path),
  getModels: createProgramModels(PROGRAMS),
  write: fn(),
  writeSecret: fn(),
  deleteSecret: fn(),
  retryPlugin: fn(),
  plugins: ref(PLUGINS),
  file: ref('/numen.json'),
  openFile: fn(),
  themes: ref([]),
  applied: ref('preset:numen'),
  mode: ref('system'),
  isPinned: ref(false),
  sizes: ref({ interfaceScale: 1, textScale: 1 }),
  bounds: ref({
    interfaceScale: { least: 0.8, most: 2 },
    textScale: { least: 0.8, most: 1.75 },
  }),
  choose: fn(),
  isSyncing: ref(true),
  isNesting: ref(true),
  headings: ref(6),
  headingsBounds: ref({ least: 1, most: 12 }),
  chooseHeadings: fn(),
}

/** The AI section and a plugin's own settings, at the width a story gives. */
const sections = (width: string) => () => ({
  components: { AiSettingsSection, PluginSettingsSection },
  setup: () => ({ state: { installation }, width }),
  template: `
    <div class="numen" :style="{ inlineSize: width, padding: '1rem', background: 'var(--numen-surface)' }">
      <AiSettingsSection :state="state" />
      <PluginSettingsSection :state="state" />
    </div>
  `,
})

const meta: Meta<typeof AiSettingsSection> = {
  title: 'Window/AI settings',
  component: AiSettingsSection,
  parameters: { layout: 'fullscreen' },
  render: sections('48rem'),
}

export default meta
type Story = StoryObj<typeof meta>

/** The words of the rows a list of choices opens on. */
const getMenuWords = (): readonly string[] =>
  Array.from(document.body.querySelectorAll<HTMLElement>('.menu__item')).map(
    (one) => one.querySelector('.menu__text')?.textContent?.trim() ?? '',
  )

/** The default agent and its model, each announced by its row and ending at one edge. */
export const TheDefaultAgentAndItsModel: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const agent = canvas.getByRole('button', { name: 'Default agent' })
    const model = canvas.getByRole('button', { name: 'Default model' })

    await expect(agent).toHaveTextContent('Agent of the panel (Sample Agent A)')
    const agentRect = agent.getBoundingClientRect()
    const modelRect = model.getBoundingClientRect()
    await expect(Math.round(modelRect.right)).toBe(Math.round(agentRect.right))

    await userEvent.click(model)
    await expect(getMenuWords()).toStrictEqual(['Agent default', 'Sample quick', 'Sample careful'])
    await userEvent.keyboard('{Escape}')
  },
}

/** A plugin that asked is put to the person above the rows, and fits a narrow window. */
export const APluginAskingIsPutToThePerson: Story = {
  render: sections('20rem'),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const activePrompt = canvas.getByRole('alert')
    await expect(activePrompt).toHaveTextContent('Sample Asking Plugin asks to use a model.')

    const frame = canvasElement.querySelector('.numen')
    if (!frame) throw new Error('the page is not drawn')
    const page = frame.getBoundingClientRect()
    const allow = within(activePrompt).getByRole('button', { name: 'Allow' }).getBoundingClientRect()
    const refuse = within(activePrompt).getByRole('button', { name: 'Refuse' }).getBoundingClientRect()
    await expect(allow.right).toBeLessThanOrEqual(page.right)
    await expect(refuse.right).toBeLessThanOrEqual(page.right)
  },
}

/** A key typed is drawn as dots, is announced by its row, and is never filled in. */
export const AKeyIsNeverDrawn: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const field = canvas.getByLabelText('Sample Agent A: apiKey')
    await expect(field).toHaveAttribute('type', 'password')
    await expect(field).toHaveValue('')

    await userEvent.type(field, 'made-up-key')
    await expect(canvas.queryByText('made-up-key')).toBeNull()
  },
}

/** A plugin's agent setting offers the default, said with the agent answering it. */
export const APluginsAgentSetting: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const agent = canvas.getByRole('button', { name: 'Agent' })
    await expect(agent).toHaveTextContent('Sample Agent A')
    await expect(canvas.getByRole('button', { name: 'Model' })).toHaveTextContent('Sample careful')

    await userEvent.click(agent)
    await expect(getMenuWords()).toStrictEqual([
      'Default (Sample Agent A)',
      'Sample Agent A',
      'Sample Agent B',
    ])
    await userEvent.keyboard('{Escape}')
  },
}
