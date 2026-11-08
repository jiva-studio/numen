import { describe, expect, it, vi } from 'vitest'
import {
  asFailure,
  asValue,
  formatErrorMessage,
  type ConnectionClient,
  type ConnectionInfo,
} from '@numen/core-client'
import { createConnectionStore } from './connections'

const connection = (overrides: Partial<ConnectionInfo> = {}): ConnectionInfo => ({
  id: 'c00000001',
  typeId: 'a.b/x',
  name: 'Work',
  revision: 1,
  fields: {},
  secretKeysSet: [],
  isTypeAvailable: true,
  isComplete: true,
  isPrivateNetworkAllowed: false,
  ...overrides,
})

const createClient = (overrides: Partial<ConnectionClient> = {}): ConnectionClient => ({
  listTypes: () => Promise.resolve([]),
  list: () => Promise.resolve([connection()]),
  create: () => Promise.resolve(asValue(connection({ id: 'c00000002' }))),
  update: () => Promise.resolve(asValue(connection({ revision: 2 }))),
  delete: () => Promise.resolve(asValue(undefined)),
  ...overrides,
})

const describeFailure = (failure: { errorCode: string; detail: string }) =>
  `${failure.errorCode}: ${failure.detail}`

const createStore = (client: ConnectionClient) => {
  const notify = vi.fn()
  return {
    notify,
    store: createConnectionStore(client, {
      describeFailure,
      loadFailed: 'Could not read the connections.',
      notify,
    }),
  }
}

describe('createConnectionStore', () => {
  it('loads the types and the connections', async () => {
    const { store } = createStore(createClient())

    await store.load()

    expect(store.connections.value.map((one) => one.name)).toStrictEqual(['Work'])
  })

  it('reports a failed read and keeps the connections', async () => {
    let isFailing = false
    const { notify, store } = createStore(
      createClient({
        list: () =>
          isFailing
            ? Promise.reject(new Error('core unreachable'))
            : Promise.resolve([connection()]),
      }),
    )
    await store.load()
    isFailing = true

    await store.load()

    expect(store.connections.value).toHaveLength(1)
    expect(notify).toHaveBeenLastCalledWith(
      expect.stringContaining('Could not read the connections.'),
      'error',
    )
  })

  it('reads the list again after a create, and answers with the connection', async () => {
    const list = vi.fn(() =>
      Promise.resolve([connection(), connection({ id: 'c00000002', name: 'New' })]),
    )
    const { store } = createStore(createClient({ list }))

    const answer = await store.create({
      typeId: 'a.b/x',
      name: 'New',
      fields: {},
      secrets: {},
      isPrivateNetworkAllowed: false,
    })

    expect(answer).toStrictEqual({ ok: true, value: expect.objectContaining({ id: 'c00000002' }) })
    expect(store.connections.value).toHaveLength(2)
  })

  it('answers a refused create with the words of the failure and writes no message', async () => {
    const { notify, store } = createStore(
      createClient({
        create: () =>
          Promise.resolve(asFailure({ errorCode: 'name_taken' as const, detail: 'Work' })),
      }),
    )

    const answer = await store.create({
      typeId: 'a.b/x',
      name: 'Work',
      fields: {},
      secrets: {},
      isPrivateNetworkAllowed: false,
    })

    expect(answer).toStrictEqual({ ok: false, error: 'name_taken: Work' })
    expect(notify).not.toHaveBeenCalled()
  })

  it('answers a call that could not reach the core with the reason', async () => {
    const { store } = createStore(
      createClient({ update: () => Promise.reject(new Error('core unreachable')) }),
    )

    const answer = await store.update({
      id: 'c00000001',
      revision: 1,
      name: 'N',
      fields: {},
      secrets: {},
      isPrivateNetworkAllowed: false,
    })

    expect(answer).toStrictEqual({
      ok: false,
      error: formatErrorMessage(new Error('core unreachable')),
    })
  })

  it('deletes a connection and reads the list again', async () => {
    const list = vi.fn(() => Promise.resolve([]))
    const remove = vi.fn(() => Promise.resolve(asValue(undefined)))
    const { store } = createStore(createClient({ list, delete: remove }))

    const answer = await store.delete('c00000001')

    expect(remove).toHaveBeenCalledWith('c00000001')
    expect(answer.ok).toBe(true)
    expect(store.connections.value).toHaveLength(0)
  })

  it('reloads only after the first load', async () => {
    const list = vi.fn(() => Promise.resolve([connection()]))
    const { store } = createStore(createClient({ list }))

    await store.reload()
    expect(list).not.toHaveBeenCalled()

    await store.load()
    await store.reload()
    expect(list).toHaveBeenCalledTimes(2)
  })
})
