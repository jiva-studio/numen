/**
 * The tabs a window keeps open, written to the core and read back.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { installCoreHandlers } from '@numen/core-client/testing'
import { readOpenTabs, writeOpenTabs } from './saved-tabs'

const request = {
  writeOpenTabs: vi.fn(),
  readOpenTabs: vi.fn(),
}

let restore: () => void

beforeEach(() => {
  vi.clearAllMocks()
  restore = installCoreHandlers(request)
})

afterEach(() => {
  restore()
})

describe('the tabs a window holds open', () => {
  it('are written as they stand, with the one in front', async () => {
    const tab = { id: 't1', kind: 'note', path: 'Llama.md', title: 'Llama', state: '' }

    await writeOpenTabs({ tabs: [tab], activeId: 't1' })

    expect(request.writeOpenTabs.mock.lastCall?.[0]).toMatchObject({ tabs: [tab], activeId: 't1' })
  })

  it('are read back with the one that was in front', async () => {
    const tab = { id: 't2', kind: 'plex', path: 'Orbits.md', title: 'Orbits', state: '{"zoom":2}' }
    request.readOpenTabs.mockResolvedValueOnce({ activeId: 't2', tabs: [tab] })

    expect(await readOpenTabs()).toStrictEqual({ activeId: 't2', tabs: [tab] })
  })
})
