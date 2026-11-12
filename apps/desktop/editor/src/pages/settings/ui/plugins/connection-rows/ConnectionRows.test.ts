/**
 * The rows of a setting of connections on the page of a plugin, in jsdom. The connections, the
 * models and every name here are invented.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { Select } from '@numen/ui'
import type {
  ConnectionInfo,
  ConnectionTypeInfo,
  ModelInfo,
  ModelListFailure,
} from '@/entities/connection'
import type { PluginSettingInfo } from '@/entities/settings'
import type { JsonValue } from '@/shared/api/json'
import type { InstallationConnections } from '../../../types'
import { asFailure, asValue, createInstallation } from '@/pages/settings/testing/installation'
import { click, dispose, findButton, root } from '@/testing/dom'
import ConnectionRows from './ConnectionRows.vue'

const type = (id: string, operations: readonly string[]): ConnectionTypeInfo => ({
  id,
  pluginId: 'a.b',
  pluginName: 'A',
  title: id,
  description: '',
  operations,
  fields: [],
  isAvailable: true,
})

const SERVICE = type('a.b/service', ['complete', 'embed'])
const EMBEDDER = type('a.b/embedder', ['embed'])

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

const MODELS: readonly ModelInfo[] = [
  { id: 'alpha', title: 'Alpha', operations: ['complete'] },
  { id: 'beta', title: 'Beta', operations: ['complete'] },
]

const setting = (overrides: Partial<PluginSettingInfo> = {}): PluginSettingInfo => ({
  key: 'connections',
  type: 'connections',
  title: 'Connections',
  description: '',
  default: undefined,
  operation: 'complete',
  max: 3,
  ...overrides,
})

const row = (connectionId: string, model: string, reasoning = '') => ({
  connection: connectionId,
  model,
  reasoning,
})

interface Options {
  readonly value?: JsonValue
  readonly max?: number
  readonly connections?: readonly ConnectionInfo[]
  readonly listModels?: InstallationConnections['models']['listModels']
  readonly operation?: string
  readonly countNotes?: InstallationConnections['countNotes']
  readonly create?: InstallationConnections['create']
}

const mountRows = (options: Options = {}) => {
  const listModels = options.listModels ?? vi.fn(() => Promise.resolve(asValue(MODELS)))
  const installation = createInstallation({
    list: ref(options.connections ?? [EMBED_ONLY, WORK, RETIRED, HOME]),
    types: ref([SERVICE, EMBEDDER]),
    listModels,
    listReasoningLevels: vi.fn(() => Promise.resolve(asValue([]))),
    ...(options.countNotes ? { countNotes: options.countNotes } : {}),
    ...(options.create ? { create: options.create } : {}),
  })
  const wrapper = mount(ConnectionRows, {
    attachTo: document.body,
    props: {
      setting: setting({ max: options.max ?? 3, operation: options.operation ?? 'complete' }),
      value: options.value ?? [],
      connections: installation.connections,
    },
  })
  return { wrapper, installation, listModels }
}

const changes = (wrapper: VueWrapper): unknown[] =>
  (wrapper.emitted('change') ?? []).map((one) => one[0])

const grips = (): HTMLElement[] => Array.from(root().querySelectorAll<HTMLElement>('[data-grip]'))

const press = (element: HTMLElement, key: string): void => {
  element.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('the rows of a setting of connections', () => {
  it('draws a row for each stored row, in order, with the line about the text of the notes', async () => {
    const { wrapper } = mountRows({ value: [row(WORK.id, 'alpha'), row(HOME.id, 'beta')] })
    await flushPromises()

    const items = root().querySelectorAll('li')
    expect(items).toHaveLength(2)
    expect(items[0]?.textContent).toContain('Work')
    expect(items[1]?.textContent).toContain('Home')
    expect(root().textContent).toContain(
      'The text of your notes is sent to each connection listed, in this order.',
    )
    dispose(wrapper)
  })

  it('names the one connection, without an order, where the setting holds one row', async () => {
    const { wrapper } = mountRows({ max: 1, value: [row(WORK.id, 'alpha')] })
    await flushPromises()

    expect(root().textContent).toContain(
      'The text of your notes is sent to the connection chosen here.',
    )
    expect(root().textContent).not.toContain('in this order')
    dispose(wrapper)
  })

  it('draws only the button when there is no row, and says nothing about the notes', async () => {
    const { wrapper } = mountRows()
    await flushPromises()

    expect(root().querySelectorAll('li')).toHaveLength(0)
    expect(root().textContent).not.toContain('The text of your notes')
    expect(findButton(/add connection/i)).toBeTruthy()
    dispose(wrapper)
  })

  it('offers no Add button while the rows number max', async () => {
    const { wrapper } = mountRows({ max: 1, value: [row(WORK.id, 'alpha')] })
    await flushPromises()

    expect(root().textContent).not.toContain('Add connection')
    dispose(wrapper)
  })

  it('turns Add off, and says where to make a connection, while there is none', async () => {
    const { wrapper } = mountRows({ connections: [] })
    await flushPromises()

    expect(findButton(/add connection/i).hasAttribute('disabled')).toBe(true)
    expect(root().textContent).toContain('Add one on the Connections page.')
    dispose(wrapper)
  })

  it('offers a way to the Connections page while no row is set, and asks to open it', async () => {
    const { wrapper } = mountRows()
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

  it('fills the model with the first model of the list once a connection is chosen, and writes the row', async () => {
    const { wrapper, listModels } = mountRows()
    await click(findButton(/add connection/i))

    wrapper.findAllComponents(Select)[0]?.vm.$emit('update:modelValue', HOME.id)
    await flushPromises()

    expect(listModels).toHaveBeenCalledWith(HOME.id, 'complete')
    expect(changes(wrapper)).toStrictEqual([[row(HOME.id, 'alpha')]])
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

    expect(changes(wrapper)).toStrictEqual([[row(HOME.id, 'beta')]])
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
    expect(choices[2]?.detail).toBe('This type cannot complete')
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
    const create = vi.fn(() => Promise.resolve(asValue(fresh)))
    const { wrapper, listModels } = mountRows({
      operation: 'embed',
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
    expect(listModels).toHaveBeenCalledWith(fresh.id, 'embed')
    expect(changes(wrapper)).toStrictEqual([])
    dispose(wrapper)
  })

  it('says under a row why the connection it names cannot serve', async () => {
    const { wrapper } = mountRows({ value: [row(EMBED_ONLY.id, 'any')] })
    await flushPromises()

    expect(root().textContent).toContain('This type cannot complete')
    dispose(wrapper)
  })

  it('shows why the models could not be listed and keeps the typed model', async () => {
    const listModels = vi.fn(() =>
      Promise.resolve(
        asFailure<ModelListFailure>({
          errorCode: 'provider_unavailable',
          detail: 'the key was rejected',
        }),
      ),
    )
    const { wrapper } = mountRows({ value: [row(WORK.id, 'typed-by-hand')], listModels })
    await flushPromises()

    expect(root().textContent).toContain('The models could not be listed: the key was rejected')
    expect(changes(wrapper)).toStrictEqual([])
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
      [row(HOME.id, 'beta'), row(WORK.id, 'alpha'), row(WORK.id, 'beta')],
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
  const EMBED_MODELS: readonly ModelInfo[] = [
    { id: 'alpha', title: 'Alpha', operations: ['embed'] },
    { id: 'beta', title: 'Beta', operations: ['embed'] },
  ]

  const mountEmbedRows = (options: Options = {}) =>
    mountRows({
      operation: 'embed',
      max: 1,
      listModels: vi.fn(() => Promise.resolve(asValue(EMBED_MODELS))),
      countNotes: vi.fn(() => Promise.resolve(120)),
      ...options,
    })

  const choose = async (wrapper: VueWrapper, index: number, id: string): Promise<void> => {
    wrapper.findAllComponents(Select)[index]?.vm.$emit('update:modelValue', id)
    await flushPromises()
  }

  it('waits for a model after a connection is chosen, and asks nothing yet', async () => {
    const { wrapper } = mountEmbedRows()
    await click(findButton(/add connection/i))

    await choose(wrapper, 0, HOME.id)

    expect(root().querySelector('[role="alertdialog"]')).toBeNull()
    expect(changes(wrapper)).toStrictEqual([])
    expect(wrapper.findAllComponents(Select)[1]?.props('modelValue')).toBe('')
    dispose(wrapper)
  })

  it('asks before it writes a connection, naming the notes and the connection', async () => {
    const { wrapper } = mountEmbedRows()
    await click(findButton(/add connection/i))

    await choose(wrapper, 0, HOME.id)
    await choose(wrapper, 1, 'alpha')

    expect(root().querySelector('[role="alertdialog"]')?.textContent).toContain(
      'This indexes 120 notes through Home. It may cost money.',
    )
    expect(changes(wrapper)).toStrictEqual([])
    dispose(wrapper)
  })

  it('writes the row once the question is confirmed', async () => {
    const { wrapper } = mountEmbedRows()
    await click(findButton(/add connection/i))
    await choose(wrapper, 0, HOME.id)
    await choose(wrapper, 1, 'alpha')

    await click(findButton(/^index$/i))

    expect(changes(wrapper)).toStrictEqual([[row(HOME.id, 'alpha')]])
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
    await click(findButton(/add connection/i))
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
      'This indexes 120 notes through Work.',
    )
    expect(changes(wrapper)).toStrictEqual([])
    dispose(wrapper)
  })

  it('does not ask when the row is removed', async () => {
    const { wrapper } = mountEmbedRows({ value: [row(WORK.id, 'alpha')] })
    await flushPromises()

    await click(findButton(/remove row 1/i))

    expect(root().querySelector('[role="alertdialog"]')).toBeNull()
    expect(changes(wrapper)).toStrictEqual([[]])
    dispose(wrapper)
  })

  it('does not ask for the rows of another operation', async () => {
    const { wrapper } = mountRows({
      operation: 'complete',
      countNotes: vi.fn(() => Promise.resolve(5)),
    })
    await click(findButton(/add connection/i))

    await choose(wrapper, 0, HOME.id)

    expect(root().querySelector('[role="alertdialog"]')).toBeNull()
    expect(changes(wrapper)).toStrictEqual([[row(HOME.id, 'alpha')]])
    dispose(wrapper)
  })

  it('says the notes without a number when they cannot be counted', async () => {
    const { wrapper } = mountEmbedRows({
      countNotes: vi.fn(() => Promise.reject(new Error('no list'))),
    })
    await click(findButton(/add connection/i))

    await choose(wrapper, 0, HOME.id)
    await choose(wrapper, 1, 'alpha')

    expect(root().querySelector('[role="alertdialog"]')?.textContent).toContain(
      'This indexes your notes through Home. It may cost money.',
    )
    dispose(wrapper)
  })
})
