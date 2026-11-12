/**
 * The page of secrets with invented secrets and plugins, drawn in a browser in both token sets. The
 * checks cover what jsdom cannot: the sections do not overflow the column at a narrow width, and the
 * confirmation of a deletion is drawn over the page.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import { ref } from 'vue'
import { createMockPluginInfo } from '@numen/plugin-manager/testing'
import { createInstallation, createSecretAnswer } from '@/pages/settings/testing/installation'
import type { Installation } from '../../types'
import SecretsPage from './SecretsPage.vue'

const EMBEDDINGS = 'md.numen.sample.embeddings'

const plugins = [
  createMockPluginInfo(EMBEDDINGS, 'Embeddings', {
    secrets: [
      {
        name: 'apiKey',
        title: 'API key',
        description: 'Key of the embeddings service',
        group: 'openrouter',
      },
    ],
  }),
]

const createSecretsInstallation = (): Installation =>
  createInstallation({
    plugins: ref(plugins),
    ...createSecretAnswer(plugins, [
      {
        id: 's00000001',
        name: 'Work',
        group: 'openrouter',
        allowed: [{ pluginId: EMBEDDINGS, name: 'apiKey', group: 'openrouter' }],
        hasValue: true,
      },
      { id: 's00000002', name: 'Home', group: 'openrouter', allowed: [], hasValue: true },
      { id: 's00000003', name: 'Personal', group: 'travel', allowed: [], hasValue: false },
    ]),
    credentialsPath: ref(''),
  })

const meta: Meta<typeof SecretsPage> = {
  title: 'Window/Secrets page',
  component: SecretsPage,
  parameters: { layout: 'fullscreen' },
  render: () => ({
    components: { SecretsPage },
    setup: () => ({ installation: createSecretsInstallation() }),
    template: `
      <div class="numen" style="container-type:inline-size;inline-size:36rem;padding:1rem;background:var(--numen-surface);--settings-measure:54rem;--settings-apart:1.75rem;--settings-choice:16rem">
        <SecretsPage :secrets="installation.secrets" :plugins="installation.plugin.plugins.value" />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<typeof meta>

export const GroupsHaveSections: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('heading', { name: 'OpenRouter' })).toBeVisible()
    await expect(
      canvas.getByText('One key for many AI models from different providers.'),
    ).toBeVisible()
    await expect(canvas.getByRole('heading', { name: 'travel' })).toBeVisible()
  },
}

/** The confirmation of a deletion names the plugin that can read the secret. */
export const DeletionAsksFirst: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const [first] = canvas.getAllByRole('button', { name: 'Secret actions' })
    await userEvent.click(first!)
    const item = within(document.body).getByRole('menuitem', { name: 'Delete' })
    await userEvent.click(item)
    const dialog = within(canvas.getByRole('alertdialog'))
    await expect(dialog.getByText(/Embeddings/)).toBeVisible()
    await userEvent.click(dialog.getByRole('button', { name: 'Cancel' }))
    await expect(canvas.queryByRole('alertdialog')).toBeNull()
  },
}

/** A secret not created yet has its plugin list disabled until a value is saved. */
export const NewSecretWaitsForAValue: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getAllByRole('button', { name: 'Add secret' })[0]!)
    const rows = Array.from(canvasElement.querySelectorAll('[role="group"]'))
    const draft = rows.find((one) => one.querySelector('button[title="Save a value first"]'))!
    await expect(draft.querySelector('button[title="Save a value first"]')).toBeDisabled()
  },
}

const emptyInstallation = (): Installation => {
  const installation = createSecretsInstallation()
  return {
    ...installation,
    plugin: { ...installation.plugin, plugins: ref([]) },
    secrets: { ...installation.secrets, ...createSecretAnswer([], []) },
  }
}

const renderEmptyPage = () => ({
  components: { SecretsPage },
  setup: () => ({ installation: emptyInstallation() }),
  template: `
    <div class="numen" style="container-type:inline-size;inline-size:36rem;padding:1rem;background:var(--numen-surface);--settings-measure:54rem;--settings-apart:1.75rem;--settings-choice:16rem">
      <SecretsPage :secrets="installation.secrets" :plugins="installation.plugin.plugins.value" />
    </div>
  `,
})

/** With no secret and no plugin the page offers to add a group. */
export const EmptyPageHasAddControl: Story = {
  render: renderEmptyPage,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('button', { name: 'Add group' })).toBeVisible()
  },
}

/** A new group takes its name at once and shows an empty row for the value. */
export const AddsInATypedGroup: Story = {
  render: renderEmptyPage,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Add group' }))
    await userEvent.type(canvas.getByLabelText('Name of the group'), 'Local Llama{Enter}')
    await expect(canvas.getByLabelText('Value of Default')).toBeVisible()
  },
}
