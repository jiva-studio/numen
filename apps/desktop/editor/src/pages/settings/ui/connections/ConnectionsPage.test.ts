/**
 * The page of connections in the settings tab, opened by the id `connections`. The connections,
 * the types and every key here are invented.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import type { ConnectionFieldInfo, ConnectionInfo, ConnectionTypeInfo } from '@/entities/connection'
import SettingsTab from '../SettingsTab.vue'
import {
  asFailure,
  asValue,
  createInstallation,
  createSettingsState,
} from '@/pages/settings/testing/installation'
import {
  click,
  dispose,
  enterText,
  findButton,
  findText,
  maskedInputs,
  nameOf,
  root,
  textInputs,
} from '@/testing/dom'

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

const type = (
  id: string,
  title: string,
  fields: readonly ConnectionFieldInfo[],
  overrides: Partial<ConnectionTypeInfo> = {},
): ConnectionTypeInfo => ({
  id,
  pluginId: id.split('/')[0] ?? '',
  pluginName: 'Sample connections',
  title,
  description: '',
  operations: ['llm.complete', 'llm.embed'],
  fields,
  rowFields: [],
  isAvailable: true,
  ...overrides,
})

const SERVICE = type('md.numen.sample/service', 'Sample service', [
  field({
    key: 'baseUrl',
    title: 'Address',
    isAddress: true,
    fixedValue: 'https://service.example/v1',
  }),
  field({ key: 'apiKey', title: 'API key', type: 'secret', isRequired: true }),
  field({
    key: 'parallelRequests',
    title: 'Parallel requests',
    type: 'number',
    defaultValue: 4,
    min: 1,
    max: 32,
    step: 1,
  }),
])
const SERVER = type('md.numen.sample/server', 'Sample server', [
  field({ key: 'baseUrl', title: 'Address', isAddress: true, isRequired: true }),
  field({ key: 'apiKey', title: 'API key', type: 'secret' }),
])
const AGENT = type('md.numen.sample/agent', 'Sample agent', [], { operations: ['llm.complete'] })

const connection = (overrides: Partial<ConnectionInfo> = {}): ConnectionInfo => ({
  id: 'c00000001',
  typeId: SERVICE.id,
  name: 'Work',
  revision: 2,
  fields: { parallelRequests: 4 },
  secretKeysSet: ['apiKey'],
  isTypeAvailable: true,
  isComplete: true,
  isDetected: false,
  uses: [],
  ...overrides,
})

const mountPage = (
  connections: readonly ConnectionInfo[],
  types: readonly ConnectionTypeInfo[],
  overrides: Parameters<typeof createInstallation>[0] = {},
) => {
  const installation = createInstallation({
    list: ref(connections),
    types: ref(types),
    ...overrides,
  })
  const wrapper = mount(SettingsTab, {
    attachTo: document.body,
    props: { state: createSettingsState(installation, { initialPluginId: ref('connections') }) },
  })
  return { installation, wrapper }
}

const rows = (): HTMLElement[] =>
  Array.from(root().querySelectorAll<HTMLElement>('[role="list"] button[aria-label]'))

const rowOf = (name: string): HTMLElement => {
  const found = rows().find((one) => one.getAttribute('aria-label') === name)
  if (!found) throw new Error(`no row ${name}`)
  return found
}

/** An entry of the list by its name, a button or not. */
const entryOf = (name: string): HTMLElement => {
  const found = root().querySelector<HTMLElement>(`[role="list"] [aria-label="${name}"]`)
  if (!found) throw new Error(`no entry ${name}`)
  return found
}

const dialog = (): HTMLElement => {
  const found = root().querySelector<HTMLElement>('[role="dialog"]')
  if (!found) throw new Error('no dialog is open')
  return found
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('the page of connections', () => {
  it('reads the connections when it opens', async () => {
    const { installation, wrapper } = mountPage([], [SERVICE])
    await flushPromises()

    expect(installation.connections.load).toHaveBeenCalledOnce()
    dispose(wrapper)
  })

  it('lists each connection by name, with its type and its address', async () => {
    const { wrapper } = mountPage(
      [
        connection(),
        connection({
          id: 'c00000002',
          typeId: SERVER.id,
          name: 'Local',
          fields: { baseUrl: 'http://localhost:11434/v1' },
          secretKeysSet: [],
        }),
      ],
      [SERVICE, SERVER],
    )
    await flushPromises()

    const listed = rows()
    expect(listed.map((one) => one.getAttribute('aria-label'))).toStrictEqual(['Work', 'Local'])
    expect(listed[0]?.textContent).toContain('Sample service')
    expect(listed[0]?.querySelector('button')).toBeNull()
    expect(listed[1]?.textContent).toContain('Sample server')
    expect(listed[1]?.textContent).toContain('http://localhost:11434/v1')
    dispose(wrapper)
  })

  it('marks a connection of a type no plugin provides as Type unavailable and explains it', async () => {
    const { wrapper } = mountPage(
      [connection({ typeId: 'gone.plugin/x', name: 'Old', isTypeAvailable: false })],
      [SERVICE],
    )
    await flushPromises()

    expect(rowOf('Old').textContent).toContain('Type unavailable')
    expect(findText(/Plugins skip such a connection/)).not.toBeNull()
    dispose(wrapper)
  })

  it('marks a connection that lacks a required value as incomplete', async () => {
    const { wrapper } = mountPage([connection({ isComplete: false, secretKeysSet: [] })], [SERVICE])
    await flushPromises()

    expect(rowOf('Work').textContent).toContain('Incomplete')
    dispose(wrapper)
  })

  it('marks a detected connection, which is no button and opens no dialog', async () => {
    const detected = connection({
      id: `${SERVICE.id}/Sample agent`,
      typeId: SERVICE.id,
      name: 'Sample agent',
      secretKeysSet: [],
      isDetected: true,
    })
    const { wrapper } = mountPage([connection(), detected], [SERVICE])
    await flushPromises()
    const sample = entryOf('Sample agent')

    expect(sample.textContent).toContain('Discovered')
    expect(sample.tagName).not.toBe('BUTTON')
    expect(rows().map((one) => one.getAttribute('aria-label'))).toEqual(['Work'])
    expect(rowOf('Work').textContent).not.toContain('Discovered')
    await click(sample)

    expect(root().querySelector('[role="dialog"]')).toBeNull()
    dispose(wrapper)
  })

  it('shows the empty state, and no way to add when no plugin provides a type', async () => {
    const { wrapper } = mountPage([], [])
    await flushPromises()

    expect(findText('No connections yet.')).not.toBeNull()
    expect(findText('No installed plugin provides a kind of connection.')).not.toBeNull()
    expect(findButton(/add connection/i).hasAttribute('disabled')).toBe(true)
    dispose(wrapper)
  })

  it('invites to add a connection when a type exists', async () => {
    const { wrapper } = mountPage([], [SERVICE])
    await flushPromises()

    expect(findText(/You enter the address and the key once/)).not.toBeNull()
    expect(findButton(/add connection/i).hasAttribute('disabled')).toBe(false)
    dispose(wrapper)
  })
})

describe('the Add dialog', () => {
  it('builds the form from the type: the name, the key and the number, and no address', async () => {
    const { wrapper } = mountPage([], [SERVICE])
    await flushPromises()

    await click(findButton(/add connection/i))

    const labels = Array.from(dialog().querySelectorAll('label')).map(nameOf)
    expect(labels).toStrictEqual(['Name', 'API key', 'Parallel requests'])
    expect(dialog().textContent).not.toContain('Address')
    expect(dialog().textContent).not.toContain('Choose the model on the page')
    expect(dialog().querySelector('a')).toBeNull()
    dispose(wrapper)
  })

  it('shows an address and a key for a server', async () => {
    const { wrapper } = mountPage([], [SERVER])
    await flushPromises()

    await click(findButton(/add connection/i))

    const labels = Array.from(dialog().querySelectorAll('label')).map(nameOf)
    expect(labels).toStrictEqual(['Name', 'Address', 'API key'])
    dispose(wrapper)
  })

  it('shows only the name for an agent type', async () => {
    const { wrapper } = mountPage([], [AGENT])
    await flushPromises()

    await click(findButton(/add connection/i))

    expect(Array.from(dialog().querySelectorAll('label')).map(nameOf)).toStrictEqual(['Name'])
    expect(dialog().textContent).toContain('No other field is needed.')
    dispose(wrapper)
  })

  it('does not save while a required value is missing', async () => {
    const { installation, wrapper } = mountPage([], [SERVICE])
    await flushPromises()
    await click(findButton(/add connection/i))

    await click(findButton(/^save$/i))

    expect(installation.connections.edits.create).not.toHaveBeenCalled()
    expect(dialog().querySelector('[role="alert"]')?.textContent).toContain('API key')
    dispose(wrapper)
  })

  it('saves the typed key and the default of the number, and closes', async () => {
    const created = connection({ id: 'c00000009', name: 'Service' })
    const createConnection = vi.fn(() => Promise.resolve(asValue(created)))
    const { wrapper } = mountPage([], [SERVICE], { create: createConnection })
    await flushPromises()
    await click(findButton(/add connection/i))

    const [key] = maskedInputs(dialog())
    await enterText(key!, 'sk-secret-1')
    await click(findButton(/^save$/i))

    expect(createConnection).toHaveBeenCalledWith({
      typeId: SERVICE.id,
      name: '',
      fields: { parallelRequests: 4 },
      secrets: { apiKey: 'sk-secret-1' },
    })
    expect(root().querySelector('[role="dialog"]')).toBeNull()
    dispose(wrapper)
  })

  it('keeps the dialog open and shows the reason when the core refuses', async () => {
    const createConnection = vi.fn(() =>
      Promise.resolve(asFailure('Another connection has this name.')),
    )
    const { wrapper } = mountPage([], [SERVICE], { create: createConnection })
    await flushPromises()
    await click(findButton(/add connection/i))
    await enterText(maskedInputs(dialog())[0]!, 'sk-secret-1')

    await click(findButton(/^save$/i))

    expect(dialog().querySelector('[role="alert"]')?.textContent).toBe(
      'Another connection has this name.',
    )
    dispose(wrapper)
  })

  it('closes on Escape and on Cancel without saving', async () => {
    const { installation, wrapper } = mountPage([], [SERVICE])
    await flushPromises()
    await click(findButton(/add connection/i))

    await click(findButton(/^cancel$/i, dialog()))

    expect(root().querySelector('[role="dialog"]')).toBeNull()
    expect(installation.connections.edits.create).not.toHaveBeenCalled()
    dispose(wrapper)
  })
})

describe('editing and deleting', () => {
  it('opens the form of the connection with its values, an empty key that shows one is stored, and sends the revision it read', async () => {
    const updateConnection = vi.fn(() => Promise.resolve(asValue(connection({ revision: 3 }))))
    const { wrapper } = mountPage([connection()], [SERVICE], { update: updateConnection })
    await flushPromises()
    await click(rowOf('Work'))

    expect(maskedInputs(dialog())[0]?.value).toBe('')
    expect(maskedInputs(dialog())[0]?.placeholder).not.toBe('')
    expect(textInputs(dialog())[0]?.value).toBe('Work')
    await click(findButton(/^save$/i))

    expect(updateConnection).toHaveBeenCalledWith({
      id: 'c00000001',
      revision: 2,
      name: 'Work',
      fields: { parallelRequests: 4 },
      secrets: {},
    })
    dispose(wrapper)
  })

  it('sends the revision of the connection as the list last read it', async () => {
    const updateConnection = vi.fn(() => Promise.resolve(asValue(connection({ revision: 4 }))))
    const list = ref<readonly ConnectionInfo[]>([connection()])
    const { wrapper } = mountPage([], [SERVICE], { list, update: updateConnection })
    await flushPromises()
    await click(rowOf('Work'))

    list.value = [connection({ revision: 3 })]
    await flushPromises()
    await click(findButton(/^save$/i))

    expect(updateConnection).toHaveBeenCalledWith(expect.objectContaining({ revision: 3 }))
    dispose(wrapper)
  })

  it('opens a row from its button, which the keyboard reaches', async () => {
    const { wrapper } = mountPage([connection()], [SERVICE])
    await flushPromises()
    const row = rowOf('Work') as HTMLButtonElement
    expect(row.type).toBe('button')

    await click(row)

    expect(textInputs(dialog())[0]?.value).toBe('Work')
    dispose(wrapper)
  })

  it('says that a changed address removes the stored key', async () => {
    const { wrapper } = mountPage(
      [
        connection({
          typeId: SERVER.id,
          fields: { baseUrl: 'https://a.example/v1' },
          secretKeysSet: ['apiKey'],
        }),
      ],
      [SERVER],
    )
    await flushPromises()
    await click(rowOf('Work'))
    expect(dialog().textContent).not.toContain('Changing the address removes the stored key')

    const address = textInputs(dialog()).find((one) => one.value === 'https://a.example/v1')!
    await enterText(address, 'https://b.example/v1')

    expect(dialog().textContent).toContain('Changing the address removes the stored key')
    dispose(wrapper)
  })

  it('asks before it deletes, and says the key is removed', async () => {
    const deleteConnection = vi.fn(() => Promise.resolve(asValue(undefined)))
    const { wrapper } = mountPage([connection()], [SERVICE], { delete: deleteConnection })
    await flushPromises()
    await click(rowOf('Work'))
    await click(findButton(/^delete$/i, dialog()))

    const confirmation = root().querySelector('[role="alertdialog"]')
    expect(confirmation?.textContent).toContain(
      'Delete "Work"? Its key is removed from this machine.',
    )
    expect(deleteConnection).not.toHaveBeenCalled()
    await click(findButton(/^delete$/i, confirmation!))

    expect(deleteConnection).toHaveBeenCalledWith('c00000001')
    expect(root().querySelector('[role="alertdialog"]')).toBeNull()
    dispose(wrapper)
  })

  it('lists the plugins whose rows use a connection, and says their rows are removed with it', async () => {
    const uses = [
      {
        pluginId: 'md.numen.tools.proofreading',
        pluginName: 'Proofreading',
        operations: ['llm.complete'],
      },
      {
        pluginId: 'md.numen.engines.embeddings',
        pluginName: 'Embeddings',
        operations: ['llm.embed'],
      },
    ]
    const { wrapper } = mountPage(
      [connection({ uses }), connection({ id: 'c00000002', name: 'Spare' })],
      [SERVICE],
    )
    await flushPromises()

    expect(rowOf('Work').textContent).toContain('Used by Proofreading and Embeddings.')
    expect(rowOf('Spare').textContent).not.toContain('Used by')
    await click(rowOf('Work'))
    await click(findButton(/^delete$/i, dialog()))
    expect(root().querySelector('[role="alertdialog"]')?.textContent).toContain(
      'Used by Proofreading and Embeddings. Their rows for it are removed.',
    )
    dispose(wrapper)
  })

  it('moves the focus to the next row after a deletion, and to Add after the last one', async () => {
    const list = ref<readonly ConnectionInfo[]>([
      connection(),
      connection({ id: 'c00000002', name: 'Spare' }),
    ])
    const deleteConnection = vi.fn((id: string) => {
      list.value = list.value.filter((one) => one.id !== id)
      return Promise.resolve(asValue(undefined))
    })
    const { wrapper } = mountPage([], [SERVICE], { list, delete: deleteConnection })
    await flushPromises()

    await click(rowOf('Work'))
    await click(findButton(/^delete$/i, dialog()))
    await click(findButton(/^delete$/i, root().querySelector('[role="alertdialog"]')!))
    expect(document.activeElement).toBe(rowOf('Spare'))

    await click(rowOf('Spare'))
    await click(findButton(/^delete$/i, dialog()))
    await click(findButton(/^delete$/i, root().querySelector('[role="alertdialog"]')!))
    expect(document.activeElement).toBe(findButton(/add connection/i))
    dispose(wrapper)
  })

  it('keeps the connection when the person cancels the deletion', async () => {
    const { installation, wrapper } = mountPage([connection()], [SERVICE])
    await flushPromises()
    await click(rowOf('Work'))
    await click(findButton(/^delete$/i, dialog()))

    await click(findButton(/^cancel$/i))

    expect(installation.connections.edits.delete).not.toHaveBeenCalled()
    expect(root().querySelector('[role="alertdialog"]')).toBeNull()
    dispose(wrapper)
  })
})
