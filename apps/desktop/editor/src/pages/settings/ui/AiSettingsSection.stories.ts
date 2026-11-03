/**
 * The AI settings, and the settings of a plugin, drawn in a browser.
 *
 * Only a browser can verify these: that the two lists of a choice end at the
 * same edge, that the prompt shown to the user fits the page at the width of a
 * narrow window, and that a typed key is never displayed.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import { ref } from 'vue'
import AiSettingsSection from './AiSettingsSection.vue'
import PluginSettingsSection from './PluginSettingsSection.vue'
import type { PluginInfo } from '@/entities/settings'
import { getSettingAt } from '@/entities/settings'
import { createInstallation, createProgramModels } from '@/testing/installation'

const createPlugin = (
  id: string,
  name: string,
  overrides: Partial<PluginInfo> = {},
): PluginInfo => ({
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
  hasFrontend: false,
  ...overrides,
})

const PLUGINS: readonly PluginInfo[] = [
  createPlugin('sample.agent-a', 'Sample Agent A', {
    secrets: [{ name: 'apiKey', title: 'apiKey', description: '' }],
    storedSecrets: ['apiKey'],
  }),
  createPlugin('sample.agent-b', 'Sample Agent B', {
    secrets: [{ name: 'apiKey', title: 'apiKey', description: '' }],
  }),
  createPlugin('sample.asking', 'Sample Asking Plugin', {
    consent: 'requested',
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

const FILE = {}

const installation = createInstallation({
  getSetting: (path) => getSettingAt(FILE, path),
  getModels: createProgramModels(PROGRAMS),
  write: fn(),
  plugins: ref(PLUGINS),
})

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

    await expect(agent).toHaveTextContent('Panel agent (Sample Agent A)')
    const agentRect = agent.getBoundingClientRect()
    const modelRect = model.getBoundingClientRect()
    await expect(Math.round(modelRect.right)).toBe(Math.round(agentRect.right))

    await userEvent.click(model)
    await expect(getMenuWords()).toStrictEqual(['Agent default', 'Sample quick', 'Sample careful'])
    await userEvent.keyboard('{Escape}')
  },
}

/** A plugin's request is shown to the user above the rows, and fits a narrow window. */
export const APluginAskingIsPutToThePerson: Story = {
  render: sections('20rem'),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const activePrompt = canvas.getByRole('alert')
    await expect(activePrompt).toHaveTextContent('Sample Asking Plugin requests model use.')

    const frame = canvasElement.querySelector('.numen')
    if (!frame) throw new Error('the page is not drawn')
    const page = frame.getBoundingClientRect()
    const allow = within(activePrompt)
      .getByRole('button', { name: 'Allow' })
      .getBoundingClientRect()
    const refuse = within(activePrompt)
      .getByRole('button', { name: 'Deny' })
      .getBoundingClientRect()
    await expect(allow.right).toBeLessThanOrEqual(page.right)
    await expect(refuse.right).toBeLessThanOrEqual(page.right)
  },
}

/** A key typed is drawn as dots, is announced by its row, and is cleared by the icon inside its field. */
export const AKeyIsNeverDrawn: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const field = canvas.getByLabelText('Sample Agent A: apiKey')
    const emptyField = canvas.getByLabelText('Sample Agent B: apiKey')
    await expect(field).toHaveAttribute('type', 'password')
    await expect(field).toHaveValue('')

    const fieldBox = field.getBoundingClientRect()
    const emptyFieldBox = emptyField.getBoundingClientRect()
    await expect(Math.round(fieldBox.width)).toBe(Math.round(emptyFieldBox.width))
    await expect(Math.round(fieldBox.right)).toBe(Math.round(emptyFieldBox.right))

    const deleteButtons = canvas.getAllByRole('button', { name: 'Delete the key' })
    await expect(deleteButtons).toHaveLength(1)
    const deleteButton = deleteButtons[0]!
    await expect(deleteButton).toHaveAttribute('title', 'Delete the key')
    const deleteBox = deleteButton.getBoundingClientRect()
    await expect(deleteBox.left).toBeGreaterThanOrEqual(fieldBox.left)
    await expect(deleteBox.right).toBeLessThanOrEqual(fieldBox.right)
    await expect(deleteBox.top).toBeGreaterThanOrEqual(fieldBox.top)
    await expect(deleteBox.bottom).toBeLessThanOrEqual(fieldBox.bottom)

    await userEvent.click(deleteButton)
    await expect(installation.deleteSecret).toHaveBeenCalledWith('sample.agent-a', 'apiKey')

    await userEvent.type(field, 'made-up-key')
    await expect(canvas.queryByText('made-up-key')).toBeNull()
  },
}
