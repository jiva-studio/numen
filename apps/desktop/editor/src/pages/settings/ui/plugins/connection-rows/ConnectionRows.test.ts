/**
 * The rows of a setting of connections on the page of a plugin, in jsdom. The connections, the
 * models and every name here are invented.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { Select } from '@numen/ui'
import {
  asFailure,
  asValue,
  type ConnectionInfo,
  type ModelListFailure,
  type ConnectionTypeInfo,
  type ModelInfo,
} from '@numen/core-client'
import type { PluginSettingInfo } from '@/entities/settings'
import type { Installation } from '../../../types'
import { createInstallation } from '@/testing/installation'
import { click, dispose, findButton, root } from '@/testing/secretsDom'
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
  isPrivateNetworkAllowed: false,
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
  readonly value?: unknown
  readonly max?: number
  readonly connections?: readonly ConnectionInfo[]
  readonly listModels?: NonNullable<Installation['listModels']>
}

const mountRows = (options: Options = {}) => {
  const listModels = options.listModels ?? vi.fn(() => Promise.resolve(asValue(MODELS)))
  const installation = createInstallation({
    connections: ref(options.connections ?? [WORK, HOME, EMBED_ONLY, RETIRED]),
    connectionTypes: ref([SERVICE, EMBEDDER]),
    listModels,
    listReasoningLevels: vi.fn(() => Promise.resolve(asValue([]))),
  })
  const wrapper = mount(ConnectionRows, {
    attachTo: document.body,
    props: {
      setting: setting({ max: options.max ?? 3 }),
      value: options.value ?? [],
      installation,
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

  it('reads the connections when it opens', async () => {
    const { wrapper, installation } = mountRows()
    await flushPromises()

    expect(installation.loadConnections).toHaveBeenCalledOnce()
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

    expect(choices.map((one) => one.text)).toStrictEqual(['Work', 'Home', 'Embed only', 'Retired'])
    expect(choices.map((one) => one.isDisabled === true)).toStrictEqual([false, false, true, true])
    expect(choices[2]?.detail).toBe('This type cannot complete')
    expect(choices[3]?.detail).toBe('Type unavailable')
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
