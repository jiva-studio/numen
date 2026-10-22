/**
 * What the vault says about itself, read into the window's own words: a part
 * of the answer the vault left out reads as nothing done and nothing counted.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { FlushResult, Unit } from '@numen/protocol'
import { sessionCore } from './session'

const request = vi.hoisted(() => ({
  getVaultState: vi.fn(),
  writeOpenTabs: vi.fn(),
  reportFlush: vi.fn(),
  watchVaultChanges: vi.fn(),
  watchFocus: vi.fn(),
  watchTasks: vi.fn(),
  watchQuit: vi.fn(),
  detachTab: vi.fn(),
}))

vi.mock('@/shared/clients', () => ({
  vault: request,
  agentService: request,
  workspace: request,
  windowService: request,
}))

beforeEach(() => {
  vi.clearAllMocks()
})

describe('the state of the vault', () => {
  it('is a scan not begun and nothing indexed where the answer carries neither', async () => {
    request.getVaultState.mockResolvedValue({ id: 'v1', name: 'Zoo', path: '/zoo' })

    expect(await sessionCore.state()).toStrictEqual({
      id: 'v1',
      name: 'Zoo',
      path: '/zoo',
      scan: { isReady: false, error: '', unwatchedPath: '' },
      coverage: { chunkCount: 0n, embeddedCount: 0n, isEmbedding: false },
    })
  })

  it('carries how far the scan got and how much the index holds', async () => {
    request.getVaultState.mockResolvedValue({
      id: 'v1',
      name: 'Zoo',
      path: '/zoo',
      scan: { isReady: true, error: 'permission denied', watchError: '/zoo/deep' },
      coverage: { chunkCount: 12n, embeddedCount: 5n, isEmbedding: true },
    })

    const state = await sessionCore.state()

    expect(state.scan).toStrictEqual({
      isReady: true,
      error: 'permission denied',
      unwatchedPath: '/zoo/deep',
    })
    expect(state.coverage).toStrictEqual({ chunkCount: 12n, embeddedCount: 5n, isEmbedding: true })
  })
})

describe('the tabs a window holds open', () => {
  it('are written as they stand, with the one in front', async () => {
    const tab = { id: 't1', kind: 'note', path: 'Llama.md' }

    await sessionCore.writeOpenTabs({ tabs: [tab], front: 't1' } as never)

    expect(request.writeOpenTabs).toHaveBeenCalledWith({ tabs: [tab], front: 't1' })
  })
})

describe('a flush reported', () => {
  it('says nothing was left where the window owed nothing', async () => {
    await sessionCore.reportFlush('token-1')

    expect(request.reportFlush).toHaveBeenCalledWith({
      window: 'editor',
      token: 'token-1',
      result: FlushResult.NOTHING,
    })
  })

  it('says what the window wrote', async () => {
    await sessionCore.reportFlush('token-1', 'written')

    expect(request.reportFlush).toHaveBeenCalledWith({
      window: 'editor',
      token: 'token-1',
      result: FlushResult.WRITTEN,
    })
  })
})

/** Everything a stream gave, read to its end. */
const readAll = async <T>(stream: AsyncIterable<T>): Promise<T[]> => {
  const readItem: T[] = []
  for await (const one of stream) readItem.push(one)
  return readItem
}

describe('the streams a window follows', () => {
  const signal = new AbortController().signal

  it('notifies each change of the vault, and where every note named went', async () => {
    request.watchVaultChanges.mockImplementation(async function* () {
      yield { paths: ['Llama.md'], reload: true, renamed: [{ from: 'A.md', to: 'B.md' }] }
    })

    expect(await readAll(sessionCore.watchVaultChanges(signal))).toStrictEqual([
      { paths: ['Llama.md'], shouldReload: true, renamed: [{ from: 'A.md', to: 'B.md' }] },
    ])
    expect(request.watchVaultChanges).toHaveBeenCalledWith({}, { signal })
  })

  it('follows the focus, and lets go of it on the signal', () => {
    const focus = (async function* () {})()
    request.watchFocus.mockReturnValue(focus)

    expect(sessionCore.watchFocus(signal)).toBe(focus)
    expect(request.watchFocus).toHaveBeenCalledWith({}, { signal })
  })

  it('follows a quit requested of this window', () => {
    const quit = (async function* () {})()
    request.watchQuit.mockReturnValue(quit)

    expect(sessionCore.watchQuit(signal)).toBe(quit)
    expect(request.watchQuit).toHaveBeenCalledWith({ window: 'editor' }, { signal })
  })

  it('notifies the work under way, counted in what each counts', async () => {
    request.watchTasks.mockImplementation(async function* () {
      yield {
        tasks: [
          {
            id: 'scan',
            title: 'Recognising a scan',
            about: 'library/scan.pdf',
            done: 3n,
            total: 10n,
            unit: Unit.BYTES,
            error: '',
            isUserStarted: true,
          },
          {
            id: 'later',
            title: 'Later',
            about: '',
            done: 0n,
            total: 0n,
            unit: 99,
            error: 'stopped',
            isUserStarted: false,
          },
        ],
      }
    })

    expect(await readAll(sessionCore.watchTasks(signal))).toStrictEqual([
      [
        {
          id: 'scan',
          label: 'Recognising a scan',
          about: 'library/scan.pdf',
          done: 3,
          total: 10,
          counting: 'bytes',
          error: '',
          isUserStarted: true,
        },
        {
          id: 'later',
          label: 'Later',
          about: '',
          done: 0,
          total: 0,
          counting: 'things',
          error: 'stopped',
          isUserStarted: false,
        },
      ],
    ])
    expect(request.watchTasks).toHaveBeenCalledWith({ window: 'editor' }, { signal })
  })
})

describe('detaching a tab', () => {
  it('asks the workspace to open a new window for the tab', async () => {
    request.detachTab.mockResolvedValue({})
    await sessionCore.detachTab('tab-42', { x: 100, y: 200 })

    expect(request.detachTab).toHaveBeenCalledWith({ tabId: 'tab-42', x: 100, y: 200 })
  })
})
