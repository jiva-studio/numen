/**
 * The rows of a setting of connections on the page of a plugin, with invented connections and
 * models, drawn in a browser in both token sets. The checks cover what jsdom cannot: the rows fit a
 * narrow column, a connection that cannot serve the operation is dimmed with its reason, the model
 * field is a combobox the keyboard drives, and a row moves by dragging its grip and by the keys.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { ref, shallowRef } from 'vue'
import type { ConnectionInfo, ConnectionTypeInfo, ModelInfo } from '@/entities/connection'
import type { JsonValue } from '@/shared/api/json'
import type { PluginSettingInfo } from '@/entities/settings'
import { asValue, createInstallation } from '@/pages/settings/testing/installation'
import ConnectionRows from './ConnectionRows.vue'

const type = (id: string, title: string, operations: readonly string[]): ConnectionTypeInfo => ({
  id,
  pluginId: id.split('/')[0] ?? '',
  pluginName: 'Sample connections',
  title,
  description: '',
  operations,
  fields: [],
  isAvailable: true,
})

const SERVICE = type('md.numen.sample/service', 'Sample service', ['complete', 'embed'])
const SERVER = type('md.numen.sample/server', 'Sample server', ['complete'])
const EMBEDDER = type('md.numen.sample/embedder', 'Sample embedder', ['embed'])

const connection = (overrides: Partial<ConnectionInfo>): ConnectionInfo => ({
  id: 'c00000001',
  typeId: SERVICE.id,
  name: 'Work',
  revision: 1,
  fields: {},
  secretKeysSet: [],
  isTypeAvailable: true,
  isComplete: true,
  uses: [],
  ...overrides,
})

const WORK = connection({})
const LOCAL = connection({ id: 'c00000002', typeId: SERVER.id, name: 'Local server' })
const EMBED_ONLY = connection({ id: 'c00000003', typeId: EMBEDDER.id, name: 'Embeddings only' })
const RETIRED = connection({
  id: 'c00000004',
  typeId: 'gone.plugin/x',
  name: 'Retired',
  isTypeAvailable: false,
})
const ALL = [EMBED_ONLY, WORK, RETIRED, LOCAL]

const MODELS: Readonly<Record<string, readonly ModelInfo[]>> = {
  [WORK.id]: [
    { id: 'google/gemini-2.5-flash', title: 'Gemini 2.5 Flash', operations: ['complete'] },
    { id: 'openai/gpt-5-mini', title: 'GPT-5 mini', operations: ['complete'] },
    { id: 'openai/text-embedding-3-small', title: 'Embedding small', operations: ['embed'] },
  ],
  [LOCAL.id]: [{ id: 'qwen2.5:14b', title: 'qwen2.5:14b', operations: [] }],
}

const setting = (overrides: Partial<PluginSettingInfo> = {}): PluginSettingInfo => ({
  key: 'connections',
  type: 'connections',
  title: 'Connections',
  description: 'Services tried in order.',
  default: undefined,
  operation: 'complete',
  max: 5,
  ...overrides,
})

const row = (connectionId: string, model: string, reasoning = '') => ({
  connection: connectionId,
  model,
  reasoning,
})

const SEVERAL = [
  row(WORK.id, 'google/gemini-2.5-flash', 'low'),
  row(LOCAL.id, 'qwen2.5:14b'),
  row(WORK.id, 'openai/gpt-5-mini'),
]

interface Setup {
  readonly rows: readonly ReturnType<typeof row>[]
  readonly max?: number
  readonly connections?: readonly ConnectionInfo[]
}

const render =
  ({ rows, max = 5, connections = ALL }: Setup) =>
  () => ({
    components: { ConnectionRows },
    setup: () => {
      const value = shallowRef<JsonValue | undefined>(rows)
      const installation = createInstallation({
        list: ref(connections),
        types: ref([SERVICE, SERVER, EMBEDDER]),
        listModels: fn((connectionId: string) =>
          Promise.resolve(asValue(MODELS[connectionId] ?? [])),
        ),
        listReasoningLevels: fn((connectionId: string, model: string) =>
          Promise.resolve(
            asValue(
              connectionId === WORK.id && model.startsWith('google/')
                ? [
                    { value: 'low', title: 'Low' },
                    { value: 'high', title: 'High' },
                  ]
                : [],
            ),
          ),
        ),
      })
      const onChange = (next: JsonValue) => {
        value.value = next
      }
      return { value, connections: installation.connections, onChange, plugin: setting({ max }) }
    },
    template: `
      <div class="numen" style="inline-size:30rem;max-inline-size:100%;padding:1rem;background:var(--numen-surface);--settings-choice:16rem">
        <ConnectionRows :setting="plugin" :value="value" :connections="connections" @change="onChange" />
        <output data-testid="written">{{ JSON.stringify(value) }}</output>
      </div>
    `,
  })

const meta: Meta<typeof ConnectionRows> = {
  title: 'Window/Connection rows',
  component: ConnectionRows,
  parameters: { layout: 'fullscreen' },
  render: render({ rows: SEVERAL }),
}

export default meta
type Story = StoryObj<typeof meta>

const readWritten = (canvasElement: HTMLElement): unknown => {
  const text = within(canvasElement).getByTestId('written').textContent
  return JSON.parse(text)
}

const getAnnouncement = (canvasElement: HTMLElement): HTMLElement => {
  const found = canvasElement.querySelector<HTMLElement>('[aria-live]')
  if (!found) throw new Error('no live region is drawn')
  return found
}

const readModels = (canvasElement: HTMLElement): string[] =>
  within(canvasElement)
    .getAllByRole('button', { name: 'Model' })
    .map((one) => one.textContent.trim())

/** Several rows fit the column in order, with Add connection under them and the line about the text of the notes. */
export const SeveralRows: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getAllByRole('button', { name: 'Connection' })).toHaveLength(3)
    await expect(canvas.getByRole('button', { name: 'Add connection' })).toBeEnabled()
    await expect(
      canvas.getByText('The text of your notes is sent to each connection listed, in this order.'),
    ).toBeVisible()
    const column = canvasElement.firstElementChild!.getBoundingClientRect()
    for (const one of canvasElement.querySelectorAll('li')) {
      await expect(one.getBoundingClientRect().right).toBeLessThanOrEqual(column.right + 0.5)
    }
    await waitFor(() => expect(readModels(canvasElement)[0]).toContain('Gemini 2.5 Flash'))
  },
}

/** A setting that holds one row at most shows no Add button once it has its row. */
export const OneRowHasNoAddButton: Story = {
  render: render({ rows: [row(WORK.id, 'google/gemini-2.5-flash')], max: 1 }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getAllByRole('button', { name: 'Connection' })).toHaveLength(1)
    await expect(canvas.queryByRole('button', { name: 'Add connection' })).toBeNull()
    await expect(
      canvas.getByText('The text of your notes is sent to the connection chosen here.'),
    ).toBeVisible()
  },
}

/** With no row the page offers Add connection and says nothing about the text of the notes. */
export const EmptyPage: Story = {
  render: render({ rows: [] }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('button', { name: 'Add connection' })).toBeEnabled()
    await expect(canvas.queryByRole('list')).toBeNull()
    await expect(canvas.queryByText(/text of your notes/)).toBeNull()
  },
}

/** With no connection to choose from, Add connection is off and the page says where to make one. */
export const NoConnectionToChoose: Story = {
  render: render({ rows: [], connections: [] }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('button', { name: 'Add connection' })).toBeDisabled()
    await expect(canvas.getByText(/Add one on the Connections page/)).toBeVisible()
  },
}

/** With no row the page offers a way to the Connections page, and asks to open it. */
export const EmptyPageLinksToConnections: Story = {
  render: render({ rows: [] }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('button', { name: 'Open Connections' })).toBeEnabled()
  },
}

/** A connection that cannot do the operation is dimmed with its reason, after the usable ones. */
export const DimmedConnection: Story = {
  render: render({ rows: [row(WORK.id, 'google/gemini-2.5-flash')] }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Connection' }))
    const menu = within(document.body)
    const items = menu.getAllByRole('menuitemradio')
    await expect(items.map((one) => one.textContent)).toStrictEqual([
      expect.stringContaining('Work'),
      expect.stringContaining('Local server'),
      expect.stringContaining('Embeddings only'),
      expect.stringContaining('Retired'),
    ])
    const embeddings = menu.getByRole('menuitemradio', { name: /Embeddings only/ })
    await expect(embeddings).toBeDisabled()
    await expect(embeddings).toHaveTextContent('This type cannot complete')
    await expect(menu.getByRole('menuitemradio', { name: /Retired/ })).toHaveTextContent(
      'Type unavailable',
    )
    await userEvent.keyboard('{Escape}')
  },
}

/** A row that names a connection that cannot serve says why under the row. */
export const RowOfAnUnusableConnection: Story = {
  render: render({ rows: [row(EMBED_ONLY.id, 'any')] }),
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByText('This type cannot complete')).toBeVisible()
  },
}

/** Adding a row and choosing a connection fills the model with the first model of its list and writes the row. */
export const ChoosingAConnectionFillsTheModel: Story = {
  render: render({ rows: [] }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Add connection' }))
    await expect(readWritten(canvasElement)).toStrictEqual([])

    await userEvent.click(canvas.getByRole('button', { name: 'Connection' }))
    await userEvent.click(within(document.body).getByRole('menuitemradio', { name: /^Work/ }))

    await waitFor(() =>
      expect(readWritten(canvasElement)).toStrictEqual([
        { connection: WORK.id, model: 'google/gemini-2.5-flash', reasoning: '' },
      ]),
    )
    await expect(canvas.getByRole('button', { name: 'Model' })).toHaveTextContent(
      'Gemini 2.5 Flash',
    )
  },
}

/** The model field is a combobox: it names its list and its active option, the arrows move, Enter chooses. */
export const ModelComboboxFollowsTheKeyboard: Story = {
  render: render({ rows: [row(WORK.id, 'google/gemini-2.5-flash')] }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(() =>
      expect(canvas.getByRole('button', { name: 'Model' })).toHaveTextContent('Gemini'),
    )
    canvas.getByRole('button', { name: 'Model' }).focus()
    await userEvent.keyboard('{ArrowDown}')

    const combobox = await within(document.body).findByRole('combobox')
    const list = within(document.body).getByRole('listbox')
    await expect(combobox).toHaveAttribute('aria-controls', list.id)
    const first = combobox.getAttribute('aria-activedescendant')
    await expect(first).not.toBeNull()
    await expect(document.getElementById(first!)).toHaveAttribute('role', 'option')

    await userEvent.keyboard('{ArrowDown}')
    const second = combobox.getAttribute('aria-activedescendant')
    await expect(second).not.toBe(first)
    await userEvent.keyboard('{Enter}')

    await waitFor(() =>
      expect(readWritten(canvasElement)).toStrictEqual([
        { connection: WORK.id, model: 'openai/gpt-5-mini', reasoning: '' },
      ]),
    )
  },
}

/** A model the service does not list is accepted when typed. */
export const ModelNameCanBeTyped: Story = {
  render: render({ rows: [row(WORK.id, 'google/gemini-2.5-flash')] }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Model' }))
    const combobox = await within(document.body).findByRole('combobox')

    await userEvent.type(combobox, 'my-own-model')
    await expect(await within(document.body).findByText('Use custom value')).toBeVisible()
    await userEvent.keyboard('{Enter}')

    await waitFor(() =>
      expect(readWritten(canvasElement)).toStrictEqual([
        { connection: WORK.id, model: 'my-own-model', reasoning: '' },
      ]),
    )
  },
}

/** The keyboard moves a row: Space picks it up, the arrows move it, Space drops it. */
export const RowMovedByTheKeyboard: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(() => expect(readModels(canvasElement)[0]).toContain('Gemini'))
    const [first] = canvas.getAllByRole('button', { name: /^Move row/ })
    first!.focus()

    await userEvent.keyboard(' ')
    await expect(first).toHaveAttribute('aria-pressed', 'true')
    await userEvent.keyboard('{ArrowDown}')
    await expect(getAnnouncement(canvasElement)).toHaveTextContent('Moved to position 2 of 3')
    await userEvent.keyboard(' ')

    await expect(getAnnouncement(canvasElement)).toHaveTextContent('Dropped at position 2 of 3')
    await waitFor(() =>
      expect(readWritten(canvasElement)).toStrictEqual([
        row(LOCAL.id, 'qwen2.5:14b'),
        row(WORK.id, 'google/gemini-2.5-flash', 'low'),
        row(WORK.id, 'openai/gpt-5-mini'),
      ]),
    )
  },
}

/** Escape puts a picked-up row back and writes nothing. */
export const MoveCancelledByEscape: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const [first] = canvas.getAllByRole('button', { name: /^Move row/ })
    first!.focus()

    await userEvent.keyboard(' ')
    await userEvent.keyboard('{ArrowDown}')
    await userEvent.keyboard('{Escape}')

    await expect(getAnnouncement(canvasElement)).toHaveTextContent('Move cancelled.')
    await expect(readWritten(canvasElement)).toStrictEqual(SEVERAL)
  },
}

/** Dragging the grip of a row down past the next one moves it. */
export const RowMovedByDragging: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(() => expect(readModels(canvasElement)[0]).toContain('Gemini'))
    const [first] = canvas.getAllByRole('button', { name: /^Move row/ })
    const rows = canvasElement.querySelectorAll('li')
    const last = rows[rows.length - 1]!.getBoundingClientRect()
    const grip = first!.getBoundingClientRect()

    await userEvent.pointer([
      {
        keys: '[MouseLeft>]',
        target: first!,
        coords: { clientX: grip.x + 4, clientY: grip.y + 4 },
      },
      { coords: { clientX: grip.x + 4, clientY: grip.y + 30 } },
      { coords: { clientX: grip.x + 4, clientY: last.bottom - 4 } },
      { keys: '[/MouseLeft]' },
    ])

    await waitFor(() =>
      expect(readWritten(canvasElement)).toStrictEqual([
        row(LOCAL.id, 'qwen2.5:14b'),
        row(WORK.id, 'openai/gpt-5-mini'),
        row(WORK.id, 'google/gemini-2.5-flash', 'low'),
      ]),
    )
  },
}

/** Removing a row writes the others. */
export const RowRemoved: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Remove row 2' }))

    await waitFor(() =>
      expect(readWritten(canvasElement)).toStrictEqual([
        row(WORK.id, 'google/gemini-2.5-flash', 'low'),
        row(WORK.id, 'openai/gpt-5-mini'),
      ]),
    )
  },
}
