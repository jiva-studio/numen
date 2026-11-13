/**
 * The vault state response mapped to values of the vault entity: a field omitted from the
 * response maps to an empty or zero value.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { installCoreHandlers } from '@numen/core-client/testing'
import { getVaultState, watchVaultChanges } from './state'

const request = {
  getVaultState: vi.fn(),
  watchVaultChanges: vi.fn(),
}

let restore: () => void

beforeEach(() => {
  vi.clearAllMocks()
  restore = installCoreHandlers(request)
})

afterEach(() => {
  restore()
})

describe('the state of the vault', () => {
  it('is a scan not begun where the answer carries none', async () => {
    request.getVaultState.mockResolvedValue({ id: 'v1', name: 'Zoo', path: '/zoo' })

    expect(await getVaultState()).toStrictEqual({
      id: 'v1',
      name: 'Zoo',
      path: '/zoo',
      scan: { isReady: false, error: '', unwatchedPath: '' },
    })
  })

  it('carries how far the scan got', async () => {
    request.getVaultState.mockResolvedValue({
      id: 'v1',
      name: 'Zoo',
      path: '/zoo',
      scan: { isReady: true, error: 'permission denied', watchError: '/zoo/deep' },
    })

    const state = await getVaultState()

    expect(state.scan).toStrictEqual({
      isReady: true,
      error: 'permission denied',
      unwatchedPath: '/zoo/deep',
    })
  })
})

const readAll = async <T>(stream: AsyncIterable<T>): Promise<T[]> => {
  const readItem: T[] = []
  for await (const one of stream) readItem.push(one)
  return readItem
}

describe('the changes to the vault a window follows', () => {
  const signal = new AbortController().signal

  it('notifies each change of the vault, where every note named went, and the files whose text versions changed', async () => {
    request.watchVaultChanges.mockImplementation(async function* () {
      yield {
        paths: ['Llama.md'],
        reload: true,
        renamed: [{ from: 'A.md', to: 'B.md' }],
        versionPaths: ['talks/a.mp3'],
      }
    })

    expect(await readAll(watchVaultChanges(signal))).toStrictEqual([
      {
        paths: ['Llama.md'],
        shouldReload: true,
        renamed: [{ from: 'A.md', to: 'B.md' }],
        versionPaths: ['talks/a.mp3'],
      },
    ])
  })
})
