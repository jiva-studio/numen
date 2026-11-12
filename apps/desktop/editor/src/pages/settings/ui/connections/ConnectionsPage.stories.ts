/**
 * The page of connections with invented connections and types, drawn in a browser in both token
 * sets. The checks cover what jsdom cannot: the rows and the dialog fit the column at a narrow
 * width, the dialog is drawn over the page, and the form follows the kind of type.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import { ref } from 'vue'
import type { ConnectionFieldInfo, ConnectionInfo, ConnectionTypeInfo } from '@/entities/connection'
import { asValue, createInstallation } from '@/pages/settings/testing/installation'
import ConnectionsPage from './ConnectionsPage.vue'

const field = (
  overrides: Partial<ConnectionFieldInfo> & Pick<ConnectionFieldInfo, 'key'>,
): ConnectionFieldInfo => ({
  type: 'string',
  title: overrides.key,
  description: '',
  isRequired: false,
  isAddress: false,
  options: [],
  ...overrides,
})

const parallel = field({
  key: 'parallelRequests',
  title: 'Parallel requests',
  type: 'number',
  defaultValue: 4,
  min: 1,
  max: 32,
  step: 1,
})

const SERVICE: ConnectionTypeInfo = {
  id: 'md.numen.sample/service',
  pluginId: 'md.numen.sample',
  pluginName: 'Sample connections',
  title: 'Sample service',
  description: 'One key for many models.',
  operations: ['llm.complete', 'llm.embed'],
  fields: [
    field({
      key: 'baseUrl',
      title: 'Address',
      isAddress: true,
      fixedValue: 'https://service.example/v1',
    }),
    field({ key: 'apiKey', title: 'API key', type: 'secret', isRequired: true }),
    parallel,
  ],
  rowFields: [],
  isAvailable: true,
}

const SERVER: ConnectionTypeInfo = {
  ...SERVICE,
  id: 'md.numen.sample/server',
  title: 'Sample server',
  description: 'A server that speaks the same API, on this computer or another.',
  fields: [
    field({ key: 'baseUrl', title: 'Address', isAddress: true, isRequired: true }),
    field({ key: 'apiKey', title: 'API key', type: 'secret' }),
    field({
      key: 'mode',
      title: 'Mode',
      type: 'enum',
      defaultValue: 'fast',
      options: [
        { value: 'fast', title: 'Fast' },
        { value: 'careful', title: 'Careful' },
      ],
    }),
    field({ key: 'stream', title: 'Stream the answer', type: 'boolean' }),
    parallel,
  ],
}

const AGENT: ConnectionTypeInfo = {
  ...SERVICE,
  id: 'md.numen.sample.agents/agent',
  pluginId: 'md.numen.sample.agents',
  pluginName: 'Sample agent',
  title: 'Sample agent',
  description: 'A program that runs on this computer.',
  operations: ['llm.complete'],
  fields: [],
}

const connection = (overrides: Partial<ConnectionInfo>): ConnectionInfo => ({
  id: 'c00000001',
  typeId: SERVICE.id,
  name: 'Work',
  revision: 1,
  fields: { parallelRequests: 4 },
  secretKeysSet: ['apiKey'],
  isTypeAvailable: true,
  isComplete: true,
  uses: [],
  ...overrides,
})

const LIST: readonly ConnectionInfo[] = [
  connection({}),
  connection({
    id: 'c00000002',
    typeId: SERVER.id,
    name: 'Local server with a long name that has to wrap inside the column of the page',
    fields: { baseUrl: 'http://localhost:11434/v1/with/a/long/path/that/also/has/to/wrap/inside' },
    secretKeysSet: [],
  }),
  connection({
    id: 'c00000003',
    typeId: AGENT.id,
    name: 'Agent',
    fields: {},
    secretKeysSet: [],
  }),
  connection({
    id: 'c00000004',
    typeId: 'gone.plugin/x',
    name: 'Retired',
    fields: {},
    secretKeysSet: [],
    isTypeAvailable: false,
  }),
  connection({ id: 'c00000005', name: 'No key yet', secretKeysSet: [], isComplete: false }),
]

const createPage = (connections: readonly ConnectionInfo[], types: readonly ConnectionTypeInfo[]) =>
  createInstallation({
    list: ref(connections),
    types: ref(types),
    create: fn(() => Promise.resolve(asValue(connection({})))),
  }).connections

const render =
  (connections: readonly ConnectionInfo[], types: readonly ConnectionTypeInfo[]) => () => ({
    components: { ConnectionsPage },
    setup: () => ({ connections: createPage(connections, types) }),
    template: `
      <div class="numen" style="container-type:inline-size;inline-size:26rem;padding:1rem;background:var(--numen-surface);--settings-measure:54rem;--settings-apart:1.75rem;--settings-choice:16rem">
        <ConnectionsPage :connections="connections" />
      </div>
    `,
  })

const meta: Meta<typeof ConnectionsPage> = {
  title: 'Window/Connections page',
  component: ConnectionsPage,
  parameters: { layout: 'fullscreen' },
  render: render(LIST, [SERVICE, SERVER, AGENT]),
}

export default meta
type Story = StoryObj<typeof meta>

/** Every row fits inside the column, a connection of a missing type says so, and an incomplete one is marked. */
export const ListFitsTheColumn: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByText('Type unavailable')).toBeVisible()
    await expect(canvas.getByText('Incomplete')).toBeVisible()
    await expect(canvas.getByText(/Plugins skip such a connection/)).toBeVisible()
    const column = canvasElement.firstElementChild!.getBoundingClientRect()
    for (const row of canvasElement.querySelectorAll('[role="list"] [role="button"]')) {
      await expect(row.getBoundingClientRect().right).toBeLessThanOrEqual(column.right + 0.5)
    }
  },
}

/** A page with no connection says so and offers to add one. */
export const EmptyPage: Story = {
  render: render([], [SERVICE]),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByText('No connections yet.')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Add connection' })).toBeEnabled()
  },
}

/** With no plugin that provides a type, the page cannot add a connection and says why. */
export const EmptyPageWithoutTypes: Story = {
  render: render([], []),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(
      canvas.getByText('No installed plugin provides a kind of connection.'),
    ).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Add connection' })).toBeDisabled()
  },
}

const openDialog = async (canvasElement: HTMLElement) => {
  await userEvent.click(within(canvasElement).getByRole('button', { name: 'Add connection' }))
  const dialog = within(document.body).getByRole('dialog', { name: 'Add connection' })
  const box = dialog.getBoundingClientRect()
  await expect(box.left).toBeGreaterThanOrEqual(0)
  await expect(box.right).toBeLessThanOrEqual(document.documentElement.clientWidth + 0.5)
  return within(dialog)
}

/** A service with a fixed address asks for a name and a key, and shows no address. */
export const AddDialogOfAService: Story = {
  render: render([], [SERVICE]),
  play: async ({ canvasElement }) => {
    const dialog = await openDialog(canvasElement)
    await expect(dialog.getByLabelText('Name')).toBeVisible()
    await expect(dialog.getByLabelText('API key')).toBeVisible()
    await expect(dialog.getByText('Parallel requests')).toBeVisible()
    await expect(dialog.queryByLabelText('Address')).toBeNull()
    await expect(dialog.queryByRole('link')).toBeNull()
  },
}

/** A server asks for the address, and a key. */
export const AddDialogOfAServer: Story = {
  render: render([], [SERVER]),
  play: async ({ canvasElement }) => {
    const dialog = await openDialog(canvasElement)
    await expect(dialog.getByLabelText('Address')).toBeVisible()
    await expect(dialog.getByText('Mode')).toBeVisible()
    await expect(dialog.getByText('Stream the answer')).toBeVisible()
  },
}

/** An agent type has nothing to enter beyond the name. */
export const AddDialogOfAnAgent: Story = {
  render: render([], [AGENT]),
  play: async ({ canvasElement }) => {
    const dialog = await openDialog(canvasElement)
    await expect(dialog.getByLabelText('Name')).toBeVisible()
    await expect(dialog.getByText(/No other field is needed/)).toBeVisible()
    await expect(dialog.queryByLabelText('API key')).toBeNull()
  },
}

/** Saving without a required value is refused in the dialog. */
export const RequiredValueIsAsked: Story = {
  render: render([], [SERVICE]),
  play: async ({ canvasElement }) => {
    const dialog = await openDialog(canvasElement)
    await userEvent.click(dialog.getByRole('button', { name: 'Save' }))
    await expect(dialog.getByRole('alert')).toHaveTextContent('API key')
  },
}

/** Deleting asks first, over the page. */
export const DeletionAsksFirst: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Work' }))
    const edit = within(document.body).getByRole('dialog', { name: 'Edit connection' })
    await userEvent.click(within(edit).getByRole('button', { name: 'Delete' }))
    const dialog = within(canvas.getByRole('alertdialog'))
    await expect(dialog.getByText(/Delete "Work"/)).toBeVisible()
    await userEvent.click(dialog.getByRole('button', { name: 'Cancel' }))
    await expect(canvas.queryByRole('alertdialog')).toBeNull()
  },
}
