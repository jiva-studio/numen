import { describe, expect, it, vi } from 'vitest'
import { formatError } from '@/shared/api/errors'
import type { Result } from '@/shared/lib/result'
import type {
  ConnectionChange,
  ConnectionDraft,
  ConnectionFailure,
  ConnectionInfo,
  ConnectionTypeInfo,
} from '../lib/types'

/** What the core lists of the types. */
interface TypeList {
  readonly types: readonly ConnectionTypeInfo[]
  readonly operations: readonly { readonly name: string; readonly title: string }[]
}
import { createConnectionApi } from '../api/connections'
import { createConnectionStore } from './connections'

const asValue = <T>(value: T): Result<T, ConnectionFailure> => ({ ok: true, value })
const asFailure = (error: ConnectionFailure): Result<never, ConnectionFailure> => ({
  ok: false,
  error,
})

/** The calls of the connection client, which a test answers. */
interface ConnectionClient {
  listTypes(): Promise<TypeList>
  list(): Promise<readonly ConnectionInfo[]>
  create(draft: ConnectionDraft): Promise<Result<ConnectionInfo, ConnectionFailure>>
  update(change: ConnectionChange): Promise<Result<ConnectionInfo, ConnectionFailure>>
  delete(id: string): Promise<Result<undefined, ConnectionFailure>>
}

const connection = (overrides: Partial<ConnectionInfo> = {}): ConnectionInfo => ({
  id: 'c00000001',
  typeId: 'a.b/x',
  name: 'Work',
  revision: 1,
  fields: {},
  secretKeysSet: [],
  isTypeAvailable: true,
  isComplete: true,
  isDetected: false,
  path: '',
  uses: [],
  ...overrides,
})

const createClient = (overrides: Partial<ConnectionClient> = {}): ConnectionClient => ({
  listTypes: () => Promise.resolve({ types: [], operations: [] }),
  list: () => Promise.resolve([connection()]),
  create: () => Promise.resolve(asValue(connection({ id: 'c00000002' }))),
  update: () => Promise.resolve(asValue(connection({ revision: 2 }))),
  delete: () => Promise.resolve(asValue(undefined)),
  ...overrides,
})

const createStore = (client: ConnectionClient) => {
  const notify = vi.fn()
  const api = createConnectionApi({ ...client, listRowOptions: () => Promise.resolve(asValue([])) })
  return { notify, store: createConnectionStore(api, notify) }
}

describe('createConnectionStore', () => {
  it('loads the types and the connections', async () => {
    const { store } = createStore(createClient())

    await store.load()

    expect(store.connections.value.map((one) => one.name)).toStrictEqual(['Work'])
  })

  it('keeps the first title of each operation and leaves out an empty one', async () => {
    const { store } = createStore(
      createClient({
        listTypes: () =>
          Promise.resolve({
            types: [],
            operations: [
              { name: 'acp.task', title: 'run agent tasks' },
              { name: 'acp.task', title: 'do tasks' },
              { name: 'llm.chat', title: '' },
            ],
          }),
      }),
    )

    await store.load()

    expect(store.operationTitles.value).toStrictEqual({ 'acp.task': 'run agent tasks' })
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
      expect.stringContaining('The connections could not be read.'),
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
    })

    expect(answer).toStrictEqual({ ok: true, value: expect.objectContaining({ id: 'c00000002' }) })
    expect(store.connections.value).toHaveLength(2)
  })

  it('answers a refused create with the words of the failure and writes no message', async () => {
    const { notify, store } = createStore(
      createClient({
        create: () => Promise.resolve(asFailure({ errorCode: 'name_taken', detail: 'Work' })),
      }),
    )

    const answer = await store.create({
      typeId: 'a.b/x',
      name: 'Work',
      fields: {},
      secrets: {},
    })

    expect(answer).toStrictEqual({ ok: false, error: 'Another connection has this name.' })
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
    })

    expect(answer).toStrictEqual({
      ok: false,
      error: formatError(new Error('core unreachable')),
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

  it('keeps the answer to the later read when an earlier read answers last', async () => {
    const answers: ((rows: readonly ConnectionInfo[]) => void)[] = []
    const list = () =>
      new Promise<readonly ConnectionInfo[]>((resolve) => {
        answers.push(resolve)
      })
    const { store } = createStore(createClient({ list }))

    const first = store.load()
    const second = store.load()
    answers[1]!([connection({ name: 'Home' })])
    await second
    answers[0]!([connection({ name: 'Work' })])
    await first

    expect(store.connections.value.map((one) => one.name)).toStrictEqual(['Home'])
  })

  it('reads the list again after an update the core refuses as stale', async () => {
    const list = vi.fn(() => Promise.resolve([connection({ revision: 3 })]))
    const update = () => Promise.resolve(asFailure({ errorCode: 'stale_revision', detail: '' }))
    const { store } = createStore(createClient({ list, update }))
    await store.load()

    const answer = await store.update({
      id: 'c00000001',
      revision: 1,
      name: 'Work',
      fields: {},
      secrets: {},
    })

    expect(answer.ok).toBe(false)
    expect(list).toHaveBeenCalledTimes(2)
    expect(store.connections.value.map((one) => one.revision)).toStrictEqual([3])
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
