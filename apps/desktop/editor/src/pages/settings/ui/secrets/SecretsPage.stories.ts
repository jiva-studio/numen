/**
 * The page of secrets with invented secrets and plugins, drawn in a browser in both token sets. The
 * checks cover what jsdom cannot: the sections do not overflow the column at a narrow width, and the
 * confirmation of a deletion is drawn over the page.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import { ref } from 'vue'
import { createMockPluginInfo } from '@numen/plugin-manager/testing'
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

const createInstallation = (): Installation => ({
  getSetting: () => undefined,
  getModels: () => [],
  write: () => undefined,
  createSecret: async () => 's00000009',
  renameSecret: () => undefined,
  writeSecret: () => undefined,
  deleteSecret: () => undefined,
  moveSecret: () => undefined,
  renameSecretGroup: () => undefined,
  allowPluginSecret: () => undefined,
  denyPluginSecret: () => undefined,
  retryPlugin: () => undefined,
  choosePath: async () => '',
  file: ref('/numen.json'),
  plugins: ref(plugins),
  secrets: ref([
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
    setup: () => ({ state: { installation: createInstallation() } }),
    template: `
      <div class="numen" style="container-type:inline-size;inline-size:36rem;padding:1rem;background:var(--numen-surface);--settings-measure:54rem;--settings-apart:1.75rem;--settings-choice:16rem">
        <SecretsPage :state="state" />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<typeof meta>

/** Each group has a section, and every row fits inside the column. */
export const SectionsFitTheColumn: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('heading', { name: 'OpenRouter' })).toBeVisible()
    await expect(
      canvas.getByText('One key for many AI models from different providers.'),
    ).toBeVisible()
    await expect(canvas.getByRole('heading', { name: 'travel' })).toBeVisible()
    const column = canvasElement.firstElementChild!.getBoundingClientRect()
    for (const row of canvasElement.querySelectorAll('li')) {
      await expect(row.getBoundingClientRect().right).toBeLessThanOrEqual(column.right + 0.5)
    }
  },
}

/** The confirmation of a deletion names the plugin that can read the secret. */
export const DeletionAsksFirst: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const [first] = canvas.getAllByRole('button', { name: 'Secret actions' })
    await userEvent.click(first!)
    const item = within(document.body).getByRole('menuitem', { name: 'Delete' })
    await expect(item.closest('[role="menu"]')!.getBoundingClientRect().right).toBeLessThanOrEqual(
      first!.getBoundingClientRect().right + 0.5,
    )
    await userEvent.click(item)
    const dialog = within(canvas.getByRole('alertdialog'))
    await expect(dialog.getByText(/Embeddings/)).toBeVisible()
    await userEvent.click(dialog.getByRole('button', { name: 'Cancel' }))
    await expect(canvas.queryByRole('alertdialog')).toBeNull()
  },
}

/**
 * A secret not created yet has the columns, the field text and the controls of a saved one, and
 * its plugin list is disabled.
 */
export const NewSecretLooksLikeSaved: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getAllByRole('button', { name: 'Add key' })[0]!)
    const rows = Array.from(canvasElement.querySelectorAll('li'))
    const draft = rows.find((one) => one.querySelector('button[title="Save a value first"]'))!
    const saved = rows.find((one) => one !== draft && one.querySelector('input[type="text"]'))!
    await expect(getComputedStyle(draft).gridTemplateColumns).toBe(
      getComputedStyle(saved).gridTemplateColumns,
    )
    await expect(draft.querySelector('button[title="Save a value first"]')).toBeDisabled()
    const sizes = [...draft.querySelectorAll('input, button')].map(
      (one) => getComputedStyle(one).fontSize,
    )
    await expect(new Set(sizes.slice(0, 3)).size).toBe(1)
  },
}

const emptyInstallation = (): Installation => ({
  ...createInstallation(),
  plugins: ref([]),
  secrets: ref([]),
})

const renderEmpty = () => ({
  components: { SecretsPage },
  setup: () => ({ state: { installation: emptyInstallation() } }),
  template: `
    <div class="numen" style="container-type:inline-size;inline-size:36rem;padding:1rem;background:var(--numen-surface);--settings-measure:54rem;--settings-apart:1.75rem;--settings-choice:16rem">
      <SecretsPage :state="state" />
    </div>
  `,
})

/** With no secret and no plugin the page offers to add a group. */
export const EmptyPageHasAddControl: Story = {
  render: renderEmpty,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('button', { name: 'Add group' })).toBeVisible()
  },
}

/** A new group takes its name at once and shows an empty row for the value. */
export const AddsInATypedGroup: Story = {
  render: renderEmpty,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Add group' }))
    await userEvent.type(canvas.getByLabelText('Name of the group'), 'Local Llama{Enter}')
    await expect(canvas.getByLabelText('Value of Default')).toBeVisible()
    const column = canvasElement.firstElementChild!.getBoundingClientRect()
    for (const row of canvasElement.querySelectorAll('li')) {
      await expect(row.getBoundingClientRect().right).toBeLessThanOrEqual(column.right + 0.5)
    }
  },
}
