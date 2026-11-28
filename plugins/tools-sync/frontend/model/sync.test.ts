import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { VaultsClient } from '../api/vaults'
import { createSyncState } from './sync'

type GetStatusResponse = Awaited<ReturnType<VaultsClient['getStatus']>>

const createStatus = (patch: Partial<GetStatusResponse> = {}): GetStatusResponse => ({
  hasIdentity: true,
  isReachable: true,
  items: [],
  ...patch,
})

const createClient = (answer: () => Promise<GetStatusResponse>) => ({
  listVaults: vi.fn<VaultsClient['listVaults']>(() => Promise.resolve({ vaults: [] })),
  getStatus: vi.fn<VaultsClient['getStatus']>(answer),
})

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('createSyncState', () => {
  it('is hidden before it starts', () => {
    const state = createSyncState(createClient(() => Promise.resolve(createStatus())))

    expect(state.phase.value).toBe('hidden')
  })

  it('is syncing until the first status arrives and then synced', async () => {
    let answer: (status: GetStatusResponse) => void = () => undefined
    const client = createClient(
      () =>
        new Promise((resolve) => {
          answer = resolve
        }),
    )
    const state = createSyncState(client)

    state.start()
    expect(state.phase.value).toBe('syncing')
    answer(createStatus())
    await vi.advanceTimersByTimeAsync(0)

    expect(state.phase.value).toBe('synced')
    state.stop()
  })

  it('is hidden while the device has no identity', async () => {
    const state = createSyncState(
      createClient(() => Promise.resolve(createStatus({ hasIdentity: false }))),
    )

    state.start()
    await vi.advanceTimersByTimeAsync(0)

    expect(state.phase.value).toBe('hidden')
    state.stop()
  })

  it('asks for the status every 10 seconds', async () => {
    const client = createClient(() => Promise.resolve(createStatus()))
    const state = createSyncState(client)

    state.start()
    await vi.advanceTimersByTimeAsync(0)
    expect(client.getStatus).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(9999)
    expect(client.getStatus).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(1)
    expect(client.getStatus).toHaveBeenCalledTimes(2)
    await vi.advanceTimersByTimeAsync(10_000)

    expect(client.getStatus).toHaveBeenCalledTimes(3)
    state.stop()
  })

  it('is hidden again when the identity is removed and shown again when it is restored', async () => {
    let hasIdentity = true
    const state = createSyncState(
      createClient(() => Promise.resolve(createStatus({ hasIdentity }))),
    )
    state.start()
    await vi.advanceTimersByTimeAsync(0)
    expect(state.phase.value).toBe('synced')

    hasIdentity = false
    await vi.advanceTimersByTimeAsync(10_000)
    expect(state.phase.value).toBe('hidden')
    hasIdentity = true
    await vi.advanceTimersByTimeAsync(10_000)

    expect(state.phase.value).toBe('synced')
    state.stop()
  })

  it('asks no more after it stops', async () => {
    const client = createClient(() => Promise.resolve(createStatus()))
    const state = createSyncState(client)
    state.start()
    await vi.advanceTimersByTimeAsync(0)

    state.stop()
    await vi.advanceTimersByTimeAsync(60_000)

    expect(client.getStatus).toHaveBeenCalledTimes(1)
  })
})

describe('createSyncState when the server does not answer', () => {
  it('shows waiting, like the rows of the vaults, and synced again once it answers', async () => {
    let status = createStatus({ isReachable: false })
    const state = createSyncState(
      createClient(() => Promise.resolve(status)),
      1000,
    )

    state.start()
    await vi.advanceTimersByTimeAsync(0)
    expect(state.phase.value).toBe('waiting')

    status = createStatus()
    await vi.advanceTimersByTimeAsync(1000)
    expect(state.phase.value).toBe('synced')
    state.stop()
  })
})

describe('createSyncState when the status cannot be read', () => {
  it('stops showing the spinner, and shows the state again once the status can be read', async () => {
    let isFailing = true
    const state = createSyncState(
      createClient(() =>
        isFailing ? Promise.reject(new Error('internal')) : Promise.resolve(createStatus()),
      ),
      1000,
    )

    state.start()
    await vi.advanceTimersByTimeAsync(0)
    expect(state.phase.value).toBe('hidden')

    isFailing = false
    await vi.advanceTimersByTimeAsync(1000)
    expect(state.phase.value).toBe('synced')
    state.stop()
  })
})
