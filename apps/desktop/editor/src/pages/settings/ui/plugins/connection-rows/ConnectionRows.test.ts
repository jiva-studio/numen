/**
 * The rows of a setting of connections on the page of a plugin, in jsdom. The connections, the
 * models and every name here are invented.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref, type Ref } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { Select } from '@numen/ui'
import type {
  ConnectionFieldInfo,
  ConnectionInfo,
  ConnectionTypeInfo,
  RowOption,
} from '@/entities/connection'
import type { PluginSettingInfo } from '@/entities/settings'
import type { JsonValue } from '@/shared/api/json'
import type { InstallationConnections } from '../../../types'
import { asFailure, asValue, createInstallation } from '@/pages/settings/testing/installation'
import { click, dispose, findButton, findButtons, root } from '@/testing/dom'
import ConnectionRows from './ConnectionRows.vue'

const rowField = (
  key: string,
  title: string,
  overrides: Partial<ConnectionFieldInfo> = {},
): ConnectionFieldInfo => ({
  key,
  type: 'string',
  title,
  description: '',
  isRequired: true,
  isAddress: false,
  options: [],
  optionsFrom: `llm.${key}s`,
  ...overrides,
})

const ROW_FIELDS = [
  rowField('model', 'Model'),
  rowField('reasoning', 'Reasoning', { type: 'enum', isRequired: false }),
]

const type = (id: string, operations: readonly string[]): ConnectionTypeInfo => ({
  id,
  pluginId: 'a.b',
  pluginName: 'A',
  title: id,
  description: '',
  operations: [...operations, 'llm.models', 'llm.reasonings'],
  fields: [],
  rowFields: ROW_FIELDS,
  isAvailable: true,
})

const SERVICE = type('a.b/service', ['llm.complete', 'llm.embed'])
const EMBEDDER = type('a.b/embedder', ['llm.embed'])

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
const HOME = connection({ id: 'c00000002', name: 'Home' })
const EMBED_ONLY = connection({ id: 'c00000003', typeId: EMBEDDER.id, name: 'Embed only' })
const RETIRED = connection({
  id: 'c00000004',
  typeId: 'gone/x',
  name: 'Retired',
  isTypeAvailable: false,
})

const MODELS: readonly RowOption[] = [
  { value: 'alpha', title: 'Alpha', group: '' },
  { value: 'beta', title: 'Beta', group: '' },
]

const listModelsOnly =
  (models: readonly RowOption[]): InstallationConnections['listRowOptions'] =>
  (_connectionId, field) =>
    Promise.resolve(asValue(field === 'model' ? models : []))

const setting = (overrides: Partial<PluginSettingInfo> = {}): PluginSettingInfo => ({
  key: 'connections',
  type: 'connections',
  title: 'Connections',
  description: '',
  default: undefined,
  operations: ['llm.complete'],
  max: 3,
  ...overrides,
})

const row = (connectionId: string, model: string) => ({ connection: connectionId, model })

const stored = (connectionId: string, model: string) => ({
  connection: connectionId,
  options: { model },
})

interface Options {
  readonly value?: JsonValue
  readonly max?: number
  readonly connections?: readonly ConnectionInfo[]
  readonly list?: Ref<readonly ConnectionInfo[]>
  readonly listRowOptions?: InstallationConnections['listRowOptions']
  readonly operation?: string
  readonly countNotes?: InstallationConnections['countNotes']
  readonly create?: InstallationConnections['edits']['create']
}

const mountRows = (options: Options = {}) => {
  const listRowOptions = vi.fn(options.listRowOptions ?? listModelsOnly(MODELS))
  const installation = createInstallation({
    list: options.list ?? ref(options.connections ?? [EMBED_ONLY, WORK, RETIRED, HOME]),
    types: ref([SERVICE, EMBEDDER]),
    operationTitles: ref({ 'llm.complete': 'complete text' }),
    listRowOptions,
    ...(options.countNotes ? { countNotes: options.countNotes } : {}),
    ...(options.create ? { create: options.create } : {}),
  })
  const wrapper = mount(ConnectionRows, {
    attachTo: document.body,
    props: {
      setting: setting({
        max: options.max ?? 3,
        operations: [options.operation ?? 'llm.complete'],
      }),
      value: options.value ?? [],
      connections: installation.connections,
    },
  })
  return { wrapper, installation, listRowOptions }
}

const changes = (wrapper: VueWrapper): unknown[] =>
  (wrapper.emitted('change') ?? []).map((one) => one[0])

const problemOf = (label: string): string =>
  root().querySelector(`button[aria-label="${label}"]`)?.getAttribute('aria-description') ?? ''

const grips = (): HTMLElement[] => Array.from(root().querySelectorAll<HTMLElement>('[data-grip]'))

const press = (element: HTMLElement, key: string): void => {
  element.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('the rows of a setting of connections', () => {
  it('draws a row for each stored row, in order', async () => {
    const { wrapper } = mountRows({ value: [row(WORK.id, 'alpha'), row(HOME.id, 'beta')] })
    await flushPromises()

    const items = root().querySelectorAll('li')
    expect(items).toHaveLength(2)
    expect(items[0]?.textContent).toContain('Work')
    expect(items[1]?.textContent).toContain('Home')
    dispose(wrapper)
  })

  it('shows the one row of a setting of one row, which cannot be moved or removed', async () => {
    const { wrapper } = mountRows({ max: 1, value: [row(WORK.id, 'alpha')] })
    await flushPromises()

    expect(root().querySelectorAll('li')).toHaveLength(1)
    expect(grips()).toHaveLength(0)
    expect(findButtons(/remove row/i)).toHaveLength(0)
    dispose(wrapper)
  })

  it('draws only the Add button when there is no row', async () => {
    const { wrapper } = mountRows()
    await flushPromises()

    expect(root().querySelectorAll('li')).toHaveLength(0)
    expect(findButton(/add connection/i)).toBeTruthy()
    dispose(wrapper)
  })

  it('offers no Add button while the rows number max', async () => {
    const { wrapper } = mountRows({ max: 1, value: [row(WORK.id, 'alpha')] })
    await flushPromises()

    expect(root().textContent).not.toContain('Add connection')
    dispose(wrapper)
  })

  it('offers no Add button, and says to add a connection first, while there is none', async () => {
    const { wrapper } = mountRows({ connections: [] })
    await flushPromises()

    expect(findButtons(/add connection/i)).toHaveLength(0)
    expect(root().textContent).toContain('No connections yet. Add a connection first.')
    dispose(wrapper)
  })

  it('offers a way to the Connections page while there is no connection, and asks to open it', async () => {
    const { wrapper } = mountRows({ connections: [] })
    await flushPromises()

    await click(findButton(/open connections/i))

    expect(wrapper.emitted('open-connections')).toHaveLength(1)
    dispose(wrapper)
  })

  it('offers no way to the Connections page once a row is set', async () => {
    const { wrapper } = mountRows({ value: [row(WORK.id, 'alpha')] })
    await flushPromises()

    expect(root().textContent).not.toContain('Open Connections')
    dispose(wrapper)
  })

  it('reads the connections when it opens', async () => {
    const { wrapper, installation } = mountRows()
    await flushPromises()

    expect(installation.connections.load).toHaveBeenCalledOnce()
    dispose(wrapper)
  })

  it('writes nothing for a row that names no connection yet', async () => {
    const { wrapper } = mountRows()
    await click(findButton(/add connection/i))

    expect(root().querySelectorAll('li')).toHaveLength(1)
    expect(changes(wrapper)).toStrictEqual([])
    dispose(wrapper)
  })

  it('asks for a model once a connection is chosen, and writes the row only when the model is chosen', async () => {
    const { wrapper, listRowOptions } = mountRows()
    await click(findButton(/add connection/i))

    wrapper.findAllComponents(Select)[0]?.vm.$emit('update:modelValue', HOME.id)
    await flushPromises()

    expect(listRowOptions).toHaveBeenCalledWith(HOME.id, 'model', {}, ['llm.complete'])
    expect(wrapper.findAllComponents(Select)[1]?.props('modelValue')).toBe('')
    expect(problemOf('Model')).toBe('Choose a model')
    expect(changes(wrapper)).toStrictEqual([])

    wrapper.findAllComponents(Select)[1]?.vm.$emit('update:modelValue', 'beta')
    await flushPromises()

    expect(problemOf('Model')).toBe('')
    expect(changes(wrapper)).toStrictEqual([[stored(HOME.id, 'beta')]])
    dispose(wrapper)
  })

  it('writes nothing when a stored row takes another connection, until its model is chosen', async () => {
    const { wrapper } = mountRows({ value: [row(WORK.id, 'alpha'), row(HOME.id, 'beta')] })
    await flushPromises()

    wrapper.findAllComponents(Select)[0]?.vm.$emit('update:modelValue', HOME.id)
    await flushPromises()

    expect(changes(wrapper)).toStrictEqual([])
    expect(root().querySelectorAll('li')).toHaveLength(2)

    wrapper.findAllComponents(Select)[1]?.vm.$emit('update:modelValue', 'alpha')
    await flushPromises()

    expect(changes(wrapper)).toStrictEqual([[stored(HOME.id, 'alpha'), stored(HOME.id, 'beta')]])
    dispose(wrapper)
  })

  it('keeps the model of a stored row when it lists the models', async () => {
    const { wrapper } = mountRows({ value: [row(WORK.id, 'typed-by-hand')] })
    await flushPromises()

    expect(changes(wrapper)).toStrictEqual([])
    dispose(wrapper)
  })

  it('writes the rows without the one removed', async () => {
    const { wrapper } = mountRows({ value: [row(WORK.id, 'alpha'), row(HOME.id, 'beta')] })
    await flushPromises()

    await click(findButton(/remove row 1/i))

    expect(changes(wrapper)).toStrictEqual([[stored(HOME.id, 'beta')]])
    dispose(wrapper)
  })

  it('lists the connections that can serve first, and dims the others with the reason', async () => {
    const { wrapper } = mountRows({ value: [row(WORK.id, 'alpha')] })
    await flushPromises()

    const choices = wrapper.findAllComponents(Select)[0]?.props('choices') ?? []

    expect(choices.map((one) => one.text)).toStrictEqual([
      'Work',
      'Home',
      'Embed only',
      'Retired',
      'New connection…',
    ])
    expect(choices.map((one) => one.isDisabled === true)).toStrictEqual([
      false,
      false,
      true,
      true,
      false,
    ])
    expect(choices[2]?.detail).toBe('This type cannot complete text')
    expect(choices[3]?.detail).toBe('Type unavailable')
    dispose(wrapper)
  })

  it('opens the dialog of a new connection from the last entry, and leaves the row alone when it is closed', async () => {
    const { wrapper } = mountRows({ value: [row(WORK.id, 'alpha')] })
    await flushPromises()

    wrapper.findAllComponents(Select)[0]?.vm.$emit('update:modelValue', 'new-connection')
    await flushPromises()
    expect(root().querySelector('[role="dialog"]')).not.toBeNull()
    await click(findButton(/^cancel$/i, root().querySelector('[role="dialog"]')!))

    expect(root().querySelector('[role="dialog"]')).toBeNull()
    expect(wrapper.findAllComponents(Select)[0]?.props('modelValue')).toBe(WORK.id)
    expect(changes(wrapper)).toStrictEqual([])
    dispose(wrapper)
  })

  it('chooses the saved connection in the row and asks for its models, without a model chosen', async () => {
    const fresh = connection({ id: 'c00000009', name: 'Fresh' })
    const list = ref<readonly ConnectionInfo[]>([WORK, HOME])
    const create = vi.fn(() => {
      list.value = [...list.value, fresh]
      return Promise.resolve(asValue(fresh))
    })
    const { wrapper, listRowOptions } = mountRows({
      list,
      operation: 'llm.embed',
      value: [row(WORK.id, 'alpha')],
      create,
    })
    await flushPromises()

    wrapper.findAllComponents(Select)[0]?.vm.$emit('update:modelValue', 'new-connection')
    await flushPromises()
    await click(findButton(/^save$/i, root().querySelector('[role="dialog"]')!))
    await flushPromises()

    expect(create).toHaveBeenCalledOnce()
    expect(root().querySelector('[role="dialog"]')).toBeNull()
    expect(wrapper.findAllComponents(Select)[0]?.props('modelValue')).toBe(fresh.id)
    expect(wrapper.findAllComponents(Select)[1]?.props('modelValue')).toBe('')
    expect(problemOf('Model')).toBe('Choose a model')
    expect(listRowOptions).toHaveBeenCalledWith(fresh.id, 'model', {}, ['llm.embed'])
    expect(changes(wrapper)).toStrictEqual([])
    dispose(wrapper)
  })

  it('asks for a connection in the one row of a setting that names none yet', async () => {
    const { wrapper } = mountRows({ max: 1, operation: 'llm.embed' })
    await flushPromises()

    expect(wrapper.findAllComponents(Select)[0]?.props('problem')).toBe('Choose a connection')
    expect(changes(wrapper)).toStrictEqual([])
    dispose(wrapper)
  })

  it('asks for nothing once the one row names a connection', async () => {
    const { wrapper } = mountRows({ max: 1, operation: 'llm.embed', value: [row(WORK.id, 'alpha')] })
    await flushPromises()

    expect(wrapper.findAllComponents(Select)[0]?.props('problem')).toBe('')
    dispose(wrapper)
  })

  it('says in the row why the connection it names cannot serve', async () => {
    const { wrapper } = mountRows({ value: [row(EMBED_ONLY.id, 'any')] })
    await flushPromises()

    expect(problemOf('Connection')).toBe('This type cannot complete text')
    dispose(wrapper)
  })

  it('shows why the models could not be listed and keeps the typed model', async () => {
    const listRowOptions = () => Promise.resolve(asFailure('the key was rejected'))
    const { wrapper } = mountRows({ value: [row(WORK.id, 'typed-by-hand')], listRowOptions })
    await flushPromises()

    expect(problemOf('Model')).toBe('The values of Model could not be listed: the key was rejected')
    expect(changes(wrapper)).toStrictEqual([])
    dispose(wrapper)
  })

  it('asks the reasoning levels of the chosen model, and clears the level when the model changes', async () => {
    const listRowOptions = vi.fn<InstallationConnections['listRowOptions']>(
      (_connectionId, field, options) =>
        Promise.resolve(
          asValue(
            field === 'model'
              ? MODELS
              : [{ value: 'deep', title: `Deep for ${options['model'] ?? ''}`, group: '' }],
          ),
        ),
    )
    const { wrapper } = mountRows({
      value: [{ connection: WORK.id, model: 'alpha', reasoning: 'deep' }],
      listRowOptions,
    })
    await flushPromises()

    expect(listRowOptions).toHaveBeenCalledWith(
      WORK.id,
      'reasoning',
      { model: 'alpha', reasoning: 'deep' },
      ['llm.complete'],
    )
    expect(
      wrapper
        .findAllComponents(Select)[2]
        ?.props('choices')
        .map((one) => one.text),
    ).toStrictEqual(['Default', 'Deep for alpha'])
    wrapper.findAllComponents(Select)[1]?.vm.$emit('update:modelValue', 'beta')
    await flushPromises()

    expect(changes(wrapper)).toStrictEqual([[stored(WORK.id, 'beta')]])
    dispose(wrapper)
  })

  it('moves a row with Space, an arrow and Space, and writes the new order once it is dropped', async () => {
    const { wrapper } = mountRows({
      value: [row(WORK.id, 'alpha'), row(HOME.id, 'beta'), row(WORK.id, 'beta')],
    })
    await flushPromises()
    const [first] = grips()

    press(first!, ' ')
    await flushPromises()
    expect(first?.getAttribute('aria-pressed')).toBe('true')
    press(first!, 'ArrowDown')
    await flushPromises()
    expect(changes(wrapper)).toStrictEqual([])
    press(grips()[1]!, ' ')
    await flushPromises()

    expect(changes(wrapper)).toStrictEqual([
      [stored(HOME.id, 'beta'), stored(WORK.id, 'alpha'), stored(WORK.id, 'beta')],
    ])
    expect(root().querySelector('[role="status"]')?.textContent).toBe('Dropped at position 2 of 3.')
    dispose(wrapper)
  })

  it('puts a picked-up row back on Escape and writes nothing', async () => {
    const { wrapper } = mountRows({ value: [row(WORK.id, 'alpha'), row(HOME.id, 'beta')] })
    await flushPromises()

    press(grips()[0]!, ' ')
    press(grips()[0]!, 'ArrowDown')
    await flushPromises()
    press(grips()[1]!, 'Escape')
    await flushPromises()

    const items = root().querySelectorAll('li')
    expect(items[0]?.textContent).toContain('Work')
    expect(changes(wrapper)).toStrictEqual([])
    dispose(wrapper)
  })

  it('does not move a row that was not picked up', async () => {
    const { wrapper } = mountRows({ value: [row(WORK.id, 'alpha'), row(HOME.id, 'beta')] })
    await flushPromises()

    press(grips()[0]!, 'ArrowDown')
    await flushPromises()

    expect(root().querySelectorAll('li')[0]?.textContent).toContain('Work')
    dispose(wrapper)
  })

  it('does not move a row past the ends of the list', async () => {
    const { wrapper } = mountRows({ value: [row(WORK.id, 'alpha'), row(HOME.id, 'beta')] })
    await flushPromises()

    press(grips()[0]!, ' ')
    press(grips()[0]!, 'ArrowUp')
    await flushPromises()

    expect(root().querySelectorAll('li')[0]?.textContent).toContain('Work')
    dispose(wrapper)
  })

  it('draws again the rows the core stored when they differ from the page', async () => {
    const { wrapper } = mountRows({ value: [row(WORK.id, 'alpha'), row(HOME.id, 'beta')] })
    await flushPromises()

    await wrapper.setProps({ value: [row(HOME.id, 'beta')] })

    expect(root().querySelectorAll('li')).toHaveLength(1)
    dispose(wrapper)
  })
})

describe('the rows of embeddings, whose change indexes the vault again', () => {
  const EMBED_MODELS: readonly RowOption[] = [
    { value: 'alpha', title: 'Alpha', group: '' },
    { value: 'beta', title: 'Beta', group: '' },
  ]

  const mountEmbedRows = (options: Options = {}) =>
    mountRows({
      operation: 'llm.embed',
      max: 1,
      listRowOptions: listModelsOnly(EMBED_MODELS),
      countNotes: vi.fn(() => Promise.resolve(120)),
      ...options,
    })

  const choose = async (wrapper: VueWrapper, index: number, id: string): Promise<void> => {
    wrapper.findAllComponents(Select)[index]?.vm.$emit('update:modelValue', id)
    await flushPromises()
  }

  it('waits for a model after a connection is chosen, and asks nothing yet', async () => {
    const { wrapper } = mountEmbedRows()
    await flushPromises()

    await choose(wrapper, 0, HOME.id)

    expect(root().querySelector('[role="alertdialog"]')).toBeNull()
    expect(changes(wrapper)).toStrictEqual([])
    expect(wrapper.findAllComponents(Select)[1]?.props('modelValue')).toBe('')
    dispose(wrapper)
  })

  it('asks before it writes a connection, naming the notes and the connection', async () => {
    const { wrapper } = mountEmbedRows()
    await flushPromises()

    await choose(wrapper, 0, HOME.id)
    await choose(wrapper, 1, 'alpha')

    expect(root().querySelector('[role="alertdialog"]')?.textContent).toContain(
      'The vault will be indexed again: 120 notes through Home. This can take a while and may cost money.',
    )
    expect(changes(wrapper)).toStrictEqual([])
    dispose(wrapper)
  })

  it('writes the row once the question is confirmed', async () => {
    const { wrapper } = mountEmbedRows()
    await flushPromises()
    await choose(wrapper, 0, HOME.id)
    await choose(wrapper, 1, 'alpha')

    await click(findButton(/^index$/i))

    expect(changes(wrapper)).toStrictEqual([[stored(HOME.id, 'alpha')]])
    expect(root().querySelector('[role="alertdialog"]')).toBeNull()
    dispose(wrapper)
  })

  it('writes nothing and keeps the stored rows when the question is cancelled', async () => {
    const { wrapper } = mountEmbedRows({ value: [row(WORK.id, 'alpha')] })
    await flushPromises()
    await choose(wrapper, 0, HOME.id)
    await choose(wrapper, 1, 'beta')

    await click(findButton(/^cancel$/i))

    expect(changes(wrapper)).toStrictEqual([])
    expect(root().querySelector('li')?.textContent).toContain('Work')
    expect(root().querySelector('[role="alertdialog"]')).toBeNull()
    dispose(wrapper)
  })

  it('keeps the row with its connection when the question about a first row is cancelled', async () => {
    const { wrapper } = mountEmbedRows()
    await flushPromises()
    await choose(wrapper, 0, HOME.id)
    await choose(wrapper, 1, 'alpha')

    await click(findButton(/^cancel$/i))

    expect(changes(wrapper)).toStrictEqual([])
    expect(root().querySelectorAll('li')).toHaveLength(1)
    expect(root().querySelector('li')?.textContent).toContain('Home')
    dispose(wrapper)
  })

  it('asks again when only the model of the stored row changes', async () => {
    const { wrapper } = mountEmbedRows({ value: [row(WORK.id, 'alpha')] })
    await flushPromises()

    wrapper.findAllComponents(Select)[1]?.vm.$emit('update:modelValue', 'beta')
    await flushPromises()

    expect(root().querySelector('[role="alertdialog"]')?.textContent).toContain(
      'The vault will be indexed again: 120 notes through Work.',
    )
    expect(changes(wrapper)).toStrictEqual([])
    dispose(wrapper)
  })

  it('does not ask when the row is removed', async () => {
    const { wrapper } = mountEmbedRows({ max: 3, value: [row(WORK.id, 'alpha')] })
    await flushPromises()

    await click(findButton(/remove row 1/i))

    expect(root().querySelector('[role="alertdialog"]')).toBeNull()
    expect(changes(wrapper)).toStrictEqual([[]])
    dispose(wrapper)
  })

  it('does not ask for the rows of another operation', async () => {
    const { wrapper } = mountRows({
      operation: 'llm.complete',
      countNotes: vi.fn(() => Promise.resolve(5)),
    })
    await click(findButton(/add connection/i))

    await choose(wrapper, 0, HOME.id)
    await choose(wrapper, 1, 'alpha')

    expect(root().querySelector('[role="alertdialog"]')).toBeNull()
    expect(changes(wrapper)).toStrictEqual([[stored(HOME.id, 'alpha')]])
    dispose(wrapper)
  })

  it('says the notes without a number when they cannot be counted', async () => {
    const { wrapper } = mountEmbedRows({
      countNotes: vi.fn(() => Promise.reject(new Error('no list'))),
    })
    await flushPromises()

    await choose(wrapper, 0, HOME.id)
    await choose(wrapper, 1, 'alpha')

    expect(root().querySelector('[role="alertdialog"]')?.textContent).toContain(
      'The vault will be indexed again through Home. This can take a while and may cost money.',
    )
    dispose(wrapper)
  })
})
