/**
 * The rows of a setting of connections on the page of a plugin, with invented connections and
 * models, drawn in a browser in both token sets. The checks cover what jsdom cannot: the rows fit a
 * narrow column, a connection that cannot serve the operation is dimmed with its reason, the model
 * field is a combobox the keyboard drives, and a row moves by dragging its grip and by the keys.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { ref, shallowRef } from 'vue'
import {
  writeRows,
  type ConnectionFieldInfo,
  type ConnectionInfo,
  type ConnectionTypeInfo,
  type RowOption,
  type StoredRow,
} from '@/entities/connection'
import type { JsonValue } from '@/shared/api/json'
import type { PluginSettingInfo } from '@/entities/settings'
import { asValue, createInstallation } from '@/pages/settings/testing/installation'
import ConnectionRows from './ConnectionRows.vue'

const rowField = (key: string, title: string, overrides: Partial<ConnectionFieldInfo> = {}) => ({
  key,
  type: 'string' as const,
  title,
  description: '',
  isRequired: true,
  isAddress: false,
  options: [],
  optionsFrom: `sample.${key}s`,
  ...overrides,
})

const ROW_FIELDS: readonly ConnectionFieldInfo[] = [
  rowField('model', 'Model'),
  rowField('reasoning', 'Reasoning', { type: 'enum', isRequired: false }),
]

const type = (id: string, title: string, operations: readonly string[]): ConnectionTypeInfo => ({
  id,
  pluginId: id.split('/')[0] ?? '',
  pluginName: 'Sample connections',
  title,
  description: '',
  operations: [...operations, 'sample.models', 'sample.reasonings'],
  fields: [],
  rowFields: ROW_FIELDS,
  isAvailable: true,
})

const SERVICE = type('md.numen.sample/service', 'Sample service', ['llm.complete', 'llm.embed'])
const SERVER = type('md.numen.sample/server', 'Sample server', ['llm.complete'])
const EMBEDDER = type('md.numen.sample/embedder', 'Sample embedder', ['llm.embed'])

const connection = (overrides: Partial<ConnectionInfo>): ConnectionInfo => ({
  id: 'c00000001',
  typeId: SERVICE.id,
  name: 'Work',
  revision: 1,
  fields: {},
  secretKeysSet: [],
  isTypeAvailable: true,
  isComplete: true,
  isDetected: false,
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

/** The models of each connection, each with the operations the sample service lists it for. */
const MODELS: Readonly<Record<string, readonly (RowOption & { readonly operation: string })[]>> = {
  [WORK.id]: [
    {
      value: 'google/gemini-2.5-flash',
      title: 'Gemini 2.5 Flash',
      group: '',
      operation: 'llm.complete',
    },
    { value: 'openai/gpt-5-mini', title: 'GPT-5 mini', group: '', operation: 'llm.complete' },
    {
      value: 'openai/text-embedding-3-small',
      title: 'Embedding small',
      group: '',
      operation: 'llm.embed',
    },
  ],
  [LOCAL.id]: [{ value: 'qwen2.5:14b', title: '', group: '', operation: '' }],
}

const listModels = (connectionId: string, operations: readonly string[]): readonly RowOption[] =>
  (MODELS[connectionId] ?? MODELS[WORK.id] ?? [])
    .filter((one) => one.operation === '' || operations.includes(one.operation))
    .map(({ value, title, group }) => ({ value, title, group }))

const setting = (overrides: Partial<PluginSettingInfo> = {}): PluginSettingInfo => ({
  key: 'connections',
  type: 'connections',
  title: 'Connections',
  description: 'Services tried in order.',
  default: undefined,
  operations: ['llm.complete'],
  max: 5,
  ...overrides,
})

const row = (connectionId: string, model: string, reasoning = '') => ({
  connection: connectionId,
  model,
  ...(reasoning === '' ? {} : { reasoning }),
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
      const list = ref(connections)
      const installation = createInstallation({
        list,
        create: fn((draft: { name: string; typeId: string }) => {
          const made = connection({
            id: 'c00000099',
            name: draft.name || 'Fresh',
            typeId: draft.typeId,
          })
          list.value = [...list.value, made]
          return Promise.resolve(asValue(made))
        }),
        types: ref([SERVICE, SERVER, EMBEDDER]),
        operationTitles: ref({ 'llm.complete': 'complete text' }),
        listRowOptions: fn(
          (
            connectionId: string,
            field: string,
            options: Readonly<Record<string, string>>,
            operations: readonly string[],
          ) => {
            if (field === 'model')
              return Promise.resolve(asValue(listModels(connectionId, operations)))
            const isGoogle =
              connectionId === WORK.id && (options['model'] ?? '').startsWith('google/')
            return Promise.resolve(
              asValue(
                isGoogle
                  ? [
                      { value: 'low', title: 'Low', group: '' },
                      { value: 'high', title: 'High', group: '' },
                    ]
                  : [],
              ),
            )
          },
        ),
      })
      const onChange = (next: readonly StoredRow[]) => {
        value.value = writeRows(next)
      }
      return { value, connections: installation.connections, onChange, plugin: setting({ max }) }
    },
    template: `
      <div class="numen" style="inline-size:30rem;max-inline-size:100%;padding:1rem;background:var(--numen-surface)">
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

/** Several rows in order, with Add connection above them. */
export const SeveralRows: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getAllByRole('button', { name: 'Connection' })).toHaveLength(3)
    await expect(canvas.getByRole('button', { name: 'Add connection' })).toBeEnabled()
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

/** With no connection to choose from, there is no Add button and the page says to add a connection first. */
export const NoConnectionToChoose: Story = {
  render: render({ rows: [], connections: [] }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.queryByRole('button', { name: 'Add connection' })).toBeNull()
    await expect(canvas.getByText('No connections yet. Add a connection first.')).toBeVisible()
  },
}

/** With no connection the page offers a way to the Connections page. */
export const EmptyPageLinksToConnections: Story = {
  render: render({ rows: [], connections: [] }),
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
    const items = await menu.findAllByRole('option')
    await expect(items.map((one) => one.textContent)).toStrictEqual([
      expect.stringContaining('Work'),
      expect.stringContaining('Local server'),
      expect.stringContaining('Embeddings only'),
      expect.stringContaining('Retired'),
      expect.stringContaining('New connection'),
    ])
    const embeddings = menu.getByRole('option', { name: /Embeddings only/ })
    await expect(embeddings).toHaveAttribute('aria-disabled', 'true')
    await expect(embeddings).toHaveTextContent('This type cannot complete text')
    await expect(menu.getByRole('option', { name: /Retired/ })).toHaveTextContent(
      'Type unavailable',
    )
    await userEvent.keyboard('{Escape}')
  },
}

/** A row that names a connection that cannot serve says why in its connection control. */
export const RowOfAnUnusableConnection: Story = {
  render: render({ rows: [row(EMBED_ONLY.id, 'any')] }),
  play: async ({ canvasElement }) => {
    await expect(
      within(canvasElement).getByRole('button', { name: 'Connection' }),
    ).toHaveAccessibleDescription('This type cannot complete text')
  },
}

/** Adding a row and choosing a connection writes nothing until the model is chosen, and the model control asks for it. */
export const ChoosingAConnectionAsksForTheModel: Story = {
  render: render({ rows: [] }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Add connection' }))
    await expect(readWritten(canvasElement)).toStrictEqual([])

    await userEvent.click(canvas.getByRole('button', { name: 'Connection' }))
    await userEvent.click(await within(document.body).findByRole('option', { name: /^Work/ }))

    await expect(canvas.getByRole('button', { name: 'Model' })).toHaveAccessibleDescription(
      'Choose a model',
    )
    await expect(readWritten(canvasElement)).toStrictEqual([])

    await userEvent.click(canvas.getByRole('button', { name: 'Model' }))
    await userEvent.click(await within(document.body).findByRole('option', { name: /GPT-5 mini/ }))

    await waitFor(() =>
      expect(readWritten(canvasElement)).toStrictEqual([
        { connection: WORK.id, model: 'openai/gpt-5-mini' },
      ]),
    )
  },
}

/** The last entry of the connection list adds a connection; the saved one is chosen, the models open, and the row is written once a model is chosen. */
export const NewConnectionEntryChoosesTheModelNext: Story = {
  render: render({ rows: [row(WORK.id, 'google/gemini-2.5-flash')] }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Connection' }))
    await userEvent.click(
      await within(document.body).findByRole('option', { name: /New connection/ }),
    )

    const dialog = await within(document.body).findByRole('dialog', { name: 'Add connection' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(within(document.body).queryByRole('dialog')).toBeNull())
    await expect(canvas.getByRole('button', { name: 'Connection' })).toHaveTextContent('Fresh')
    await expect(await within(document.body).findByRole('combobox')).toBeVisible()
    await expect(readWritten(canvasElement)).toStrictEqual([
      row(WORK.id, 'google/gemini-2.5-flash'),
    ])

    await userEvent.click(await within(document.body).findByRole('option', { name: /GPT-5 mini/ }))

    await waitFor(() =>
      expect(readWritten(canvasElement)).toStrictEqual([
        { connection: 'c00000099', model: 'openai/gpt-5-mini' },
      ]),
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
        { connection: WORK.id, model: 'openai/gpt-5-mini' },
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
        { connection: WORK.id, model: 'my-own-model' },
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

/** Dragging the grip of a row down past the end of the list moves it to the end. */
export const RowMovedByDragging: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(() => expect(readModels(canvasElement)[0]).toContain('Gemini'))
    const [first] = canvas.getAllByRole('button', { name: /^Move row/ })

    await userEvent.pointer([
      { keys: '[MouseLeft>]', target: first!, coords: { clientX: 0, clientY: 0 } },
      { coords: { clientX: 0, clientY: 30 } },
      { coords: { clientX: 0, clientY: 100_000 } },
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
