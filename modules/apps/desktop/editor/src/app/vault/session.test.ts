/**
 * What the vault says about itself, read into the window's own words: a part
 * of the answer the vault left out reads as nothing done and nothing counted.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { FlushResult } from '@numen/protocol'
import { sessionCore } from './session'

const asked = vi.hoisted(() => ({
  getVaultState: vi.fn(),
  writeOpenTabs: vi.fn(),
  reportFlush: vi.fn(),
}))

vi.mock('@/shared/clients', () => ({
  vault: asked,
  agentService: asked,
  workspace: asked,
  windowService: asked,
}))

beforeEach(() => {
  vi.clearAllMocks()
})

describe('the state of the vault', () => {
  it('is a scan not begun and nothing indexed where the answer carries neither', async () => {
    asked.getVaultState.mockResolvedValue({ id: 'v1', name: 'Zoo', path: '/zoo' })

    expect(await sessionCore.state()).toStrictEqual({
      id: 'v1',
      name: 'Zoo',
      path: '/zoo',
      scan: { isReady: false, error: '', unwatchedPath: '' },
      coverage: { chunkCount: 0n, embeddedCount: 0n, isEmbedding: false },
    })
  })

  it('carries how far the scan got and how much the index holds', async () => {
    asked.getVaultState.mockResolvedValue({
      id: 'v1',
      name: 'Zoo',
      path: '/zoo',
      scan: { isReady: true, error: 'permission denied', unwatched: '/zoo/deep' },
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

    expect(asked.writeOpenTabs).toHaveBeenCalledWith({ tabs: [tab], front: 't1' })
  })
})

describe('a flush reported', () => {
  it('says nothing was left where the window owed nothing', async () => {
    await sessionCore.reportFlush('token-1')

    expect(asked.reportFlush).toHaveBeenCalledWith({
      window: 'editor',
      token: 'token-1',
      result: FlushResult.NOTHING,
    })
  })

  it('says what the window wrote', async () => {
    await sessionCore.reportFlush('token-1', 'written')

    expect(asked.reportFlush).toHaveBeenCalledWith({
      window: 'editor',
      token: 'token-1',
      result: FlushResult.WRITTEN,
    })
  })
})
