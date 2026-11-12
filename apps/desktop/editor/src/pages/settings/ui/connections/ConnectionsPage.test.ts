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
  operations: ['complete', 'embed'],
  fields,
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
const AGENT = type('md.numen.sample/agent', 'Sample agent', [], { operations: ['complete'] })

const connection = (overrides: Partial<ConnectionInfo> = {}): ConnectionInfo => ({
  id: 'c00000001',
  typeId: SERVICE.id,
  name: 'Work',
  revision: 2,
  fields: { parallelRequests: 4 },
  secretKeysSet: ['apiKey'],
  isTypeAvailable: true,
  isComplete: true,
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

  it('lists each connection with its type, its address and what the type can do', async () => {
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

    const rows = Array.from(root().querySelectorAll('li[aria-label]'))
    expect(rows.map((one) => one.getAttribute('aria-label'))).toStrictEqual(['Work', 'Local'])
    expect(rows[0]?.textContent).toContain('Sample service')
    expect(
      Array.from(rows[0]?.querySelectorAll('.connection-row__chip') ?? []).map((one) =>
        one.textContent?.trim(),
      ),
    ).toStrictEqual(['Completion', 'Embeddings'])
    expect(rows[0]?.querySelector('button')).toBeNull()
    expect(rows[1]?.textContent).toContain('http://localhost:11434/v1')
    dispose(wrapper)
  })

  it('marks a connection of a type no plugin provides as Type unavailable and explains it', async () => {
    const { wrapper } = mountPage(
      [connection({ typeId: 'gone.plugin/x', name: 'Old', isTypeAvailable: false })],
      [SERVICE],
    )
    await flushPromises()

    const row = root().querySelector('li[aria-label="Old"]')
    expect(row?.textContent).toContain('Type unavailable')
    expect(row?.textContent).not.toContain('Completion')
    expect(findText(/Plugins skip such a connection/)).not.toBeNull()
    dispose(wrapper)
  })

  it('marks a connection that lacks a required value as incomplete', async () => {
    const { wrapper } = mountPage([connection({ isComplete: false, secretKeysSet: [] })], [SERVICE])
    await flushPromises()

    expect(root().querySelector('li[aria-label="Work"]')?.textContent).toContain('Incomplete')
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

  it('shows an address and an optional key for a server', async () => {
    const { wrapper } = mountPage([], [SERVER])
    await flushPromises()

    await click(findButton(/add connection/i))

    const labels = Array.from(dialog().querySelectorAll('label')).map(nameOf)
    expect(labels).toStrictEqual(['Name', 'Address', 'API key'])
    expect(dialog().textContent).toContain('May stay empty for a server on this computer.')
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

    expect(installation.connections.create).not.toHaveBeenCalled()
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
    expect(installation.connections.create).not.toHaveBeenCalled()
    dispose(wrapper)
  })
})

describe('editing and deleting', () => {
  it('opens the form of the connection with its values and no key, and sends the revision it read', async () => {
    const updateConnection = vi.fn(() => Promise.resolve(asValue(connection({ revision: 3 }))))
    const { wrapper } = mountPage([connection()], [SERVICE], { update: updateConnection })
    await flushPromises()
    await click(root().querySelector('li[aria-label="Work"]')!)

    expect(maskedInputs(dialog())[0]?.value).toBe('')
    expect(dialog().textContent).toContain('A key is stored. Type to replace it.')
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

  it('opens a row with Enter and with Space from the keyboard', async () => {
    const { wrapper } = mountPage([connection()], [SERVICE])
    await flushPromises()
    const row = root().querySelector<HTMLElement>('li[aria-label="Work"]')!
    expect(row.getAttribute('tabindex')).toBe('0')

    row.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    await flushPromises()

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
    await click(root().querySelector('li[aria-label="Work"]')!)
    expect(dialog().textContent).not.toContain('Changing the address removes the stored key')

    const address = textInputs(dialog()).find((one) => one.value === 'https://a.example/v1')!
    await enterText(address, 'https://b.example/v1')

    expect(dialog().textContent).toContain('Changing the address removes the stored key')
    dispose(wrapper)
  })

  it('sends an empty key when the person removes it', async () => {
    const updateConnection = vi.fn(() => Promise.resolve(asValue(connection({ revision: 3 }))))
    const { wrapper } = mountPage([connection()], [SERVICE], { update: updateConnection })
    await flushPromises()
    await click(root().querySelector('li[aria-label="Work"]')!)

    await click(findButton(/remove key/i, dialog()))
    await click(findButton(/^save$/i))

    expect(updateConnection).toHaveBeenCalledWith(
      expect.objectContaining({ secrets: { apiKey: '' } }),
    )
    dispose(wrapper)
  })

  it('asks before it deletes, and says the key is removed', async () => {
    const deleteConnection = vi.fn(() => Promise.resolve(asValue(undefined)))
    const { wrapper } = mountPage([connection()], [SERVICE], { delete: deleteConnection })
    await flushPromises()
    await click(root().querySelector('li[aria-label="Work"]')!)
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
        operation: 'complete',
      },
      { pluginId: 'md.numen.engines.embeddings', pluginName: 'Embeddings', operation: 'embed' },
    ]
    const { wrapper } = mountPage(
      [connection({ uses }), connection({ id: 'c00000002', name: 'Spare' })],
      [SERVICE],
    )
    await flushPromises()

    const rows = root().querySelectorAll('.connection-row')
    expect(rows[0]?.textContent).toContain('Used by Proofreading and Embeddings.')
    expect(rows[1]?.textContent).not.toContain('Used by')
    await click(root().querySelector('li[aria-label="Work"]')!)
    await click(findButton(/^delete$/i, dialog()))
    expect(root().querySelector('[role="alertdialog"]')?.textContent).toContain(
      'Used by Proofreading and Embeddings. Their rows for it are removed.',
    )
    dispose(wrapper)
  })

  it('keeps the connection when the person cancels the deletion', async () => {
    const { installation, wrapper } = mountPage([connection()], [SERVICE])
    await flushPromises()
    await click(root().querySelector('li[aria-label="Work"]')!)
    await click(findButton(/^delete$/i, dialog()))

    await click(findButton(/^cancel$/i))

    expect(installation.connections.delete).not.toHaveBeenCalled()
    expect(root().querySelector('[role="alertdialog"]')).toBeNull()
    dispose(wrapper)
  })
})
