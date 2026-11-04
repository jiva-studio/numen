import { describe, expect, it, vi } from 'vitest'
import type { TabLinkGroup } from '@/features/tab-linking'
import { createTabSync, type TabSyncDeps } from './useTabSync'

type Navigate = (state: unknown, path: string, title: string) => void | Promise<void>

const createWindowTabs = (tabs: Record<string, { state?: unknown; navigate?: Navigate }>) =>
  ({
    getTab: (id: string) => {
      const one = tabs[id]
      if (!one) return null
      return {
        state: one.state,
        kind: { kind: id.split(':')[0], ...(one.navigate ? { navigate: one.navigate } : {}) },
      }
    },
  }) as unknown as TabSyncDeps['windowTabs']

/** One group holding the source `s` and the tabs given. */
const createLinkedTabs = (...targets: string[]): TabSyncDeps['tabLinks'] => {
  const group: TabLinkGroup = { id: 'group-1', colorIndex: 1, tabs: ['s', ...targets] }
  return {
    groups: new Map([[group.id, group]]),
    getGroupOf: (tab) => (group.tabs.includes(tab) ? group : null),
  }
}

const setup = (
  tabs: Parameters<typeof createWindowTabs>[0],
  tabLinks: TabSyncDeps['tabLinks'],
  onError = vi.fn(),
) => {
  const sendSync = vi.fn((_sync: unknown) => Promise.resolve())
  const sync = createTabSync({ windowTabs: createWindowTabs(tabs), tabLinks, sendSync, onError })
  return { sync, sendSync, onError }
}

describe('a path chosen in a linked tab', () => {
  it('moves every tab of this window linked to it, and sends the move to the core', () => {
    const files = vi.fn<Navigate>()
    const other = vi.fn<Navigate>()
    const { sync, sendSync } = setup(
      {
        s: { navigate: vi.fn() },
        'files:1': { state: 'files state', navigate: files },
        'other:1': { state: 'other state', navigate: other },
      },
      createLinkedTabs('files:1', 'other:1', 'plex:elsewhere'),
    )

    expect(sync.syncPathFromTab('s', 'test.md', 'Test')).toBe(true)
    expect(files).toHaveBeenCalledWith('files state', 'test.md', 'Test')
    expect(other).toHaveBeenCalledWith('other state', 'test.md', 'Test')
    expect(sendSync).toHaveBeenCalledWith({ sourceTabId: 's', path: 'test.md', title: 'Test' })
  })

  it('is linked when the only other tab of the group is in another window', () => {
    const { sync, sendSync } = setup({ s: {} }, createLinkedTabs('plex:elsewhere'))

    expect(sync.syncPathFromTab('s', 'a.md')).toBe(true)
    expect(sendSync).toHaveBeenCalledOnce()
  })

  it('leaves alone a linked tab whose kind cannot move', () => {
    const navigate = vi.fn<Navigate>()
    const { sync } = setup(
      { 'agent:1': {}, 'files:1': { navigate } },
      createLinkedTabs('agent:1', 'files:1'),
    )

    expect(sync.syncPathFromTab('s', 'a.md')).toBe(true)
    expect(navigate).toHaveBeenCalledExactlyOnceWith(undefined, 'a.md', '')
  })

  it('moves nothing from a tab linked to none', () => {
    const navigate = vi.fn<Navigate>()
    const { sync, sendSync } = setup(
      { 'files:1': { navigate } },
      { groups: new Map(), getGroupOf: () => null },
    )

    expect(sync.syncPathFromTab('s', 'a.md')).toBe(false)
    expect(navigate).not.toHaveBeenCalled()
    expect(sendSync).not.toHaveBeenCalled()
  })

  it('moves nothing again for the path the group last moved to', () => {
    const navigate = vi.fn<Navigate>()
    const { sync, sendSync } = setup({ 'files:1': { navigate } }, createLinkedTabs('files:1'))

    sync.syncPathFromTab('s', 'a.md')
    expect(sync.syncPathFromTab('files:1', 'a.md')).toBe(true)

    expect(navigate).toHaveBeenCalledOnce()
    expect(sendSync).toHaveBeenCalledOnce()
  })
})

describe('a move another window sent', () => {
  it('moves the tabs of this window in the group', () => {
    const navigate = vi.fn<Navigate>()
    const { sync, sendSync } = setup(
      { 'files:1': { state: 'files state', navigate } },
      createLinkedTabs('files:1'),
    )

    sync.applySync({ groupId: 'group-1', sourceTabId: 's', path: 'b.md', title: 'B' })

    expect(navigate).toHaveBeenCalledWith('files state', 'b.md', 'B')
    expect(sendSync).not.toHaveBeenCalled()
  })

  it('is dropped when its source is a tab of this window', () => {
    const navigate = vi.fn<Navigate>()
    const { sync } = setup({ s: {}, 'files:1': { navigate } }, createLinkedTabs('files:1'))

    sync.applySync({ groupId: 'group-1', sourceTabId: 's', path: 'b.md', title: 'B' })

    expect(navigate).not.toHaveBeenCalled()
  })

  it('is not sent back when a moved tab reports the same path', () => {
    const navigate = vi.fn<Navigate>()
    const { sync, sendSync } = setup({ 'files:1': { navigate } }, createLinkedTabs('files:1'))
    navigate.mockImplementation(() => {
      sync.syncPathFromTab('files:1', 'b.md', 'B')
    })

    sync.applySync({ groupId: 'group-1', sourceTabId: 's', path: 'b.md', title: 'B' })
    expect(sync.syncPathFromTab('files:1', 'b.md', 'B')).toBe(true)
    sync.applySync({ groupId: 'group-1', sourceTabId: 's', path: 'b.md', title: 'B' })

    expect(navigate).toHaveBeenCalledOnce()
    expect(sendSync).not.toHaveBeenCalled()
  })

  it('is dropped for a group this window does not know', () => {
    const navigate = vi.fn<Navigate>()
    const { sync } = setup({ 'files:1': { navigate } }, createLinkedTabs('files:1'))

    sync.applySync({ groupId: 'group-9', sourceTabId: 'x', path: 'b.md', title: '' })

    expect(navigate).not.toHaveBeenCalled()
  })
})

describe('createTabSync guards', () => {
  it('refuses an empty source tab id or an empty path even when targets are linked', () => {
    const navigate = vi.fn<Navigate>()
    const { sync } = setup({ 'note:1': { navigate } }, createLinkedTabs('note:1'))

    expect(sync.syncPathFromTab('', 'a.md')).toBe(false)
    expect(sync.syncPathFromTab('s', '')).toBe(false)
    expect(navigate).not.toHaveBeenCalled()
  })

  it('refuses a sync started while one is running, and accepts the next after it ends', () => {
    const inner: boolean[] = []
    const navigate = vi.fn<Navigate>(() => {
      inner.push(sync.syncPathFromTab('s', 'b.md'))
    })
    const { sync } = setup({ 'note:1': { navigate } }, createLinkedTabs('note:1'))

    expect(sync.syncPathFromTab('s', 'a.md')).toBe(true)
    expect(inner).toEqual([false])
    expect(navigate).toHaveBeenCalledTimes(1)

    expect(sync.syncPathFromTab('s', 'c.md')).toBe(true)
    expect(navigate).toHaveBeenCalledTimes(2)
  })

  it('ends the running state when a target throws', () => {
    let fail = true
    const navigate = vi.fn<Navigate>(() => {
      if (fail) throw new Error('boom')
    })
    const { sync } = setup({ 'note:1': { navigate } }, createLinkedTabs('note:1'))

    expect(() => sync.syncPathFromTab('s', 'a.md')).toThrow('boom')
    fail = false
    expect(sync.syncPathFromTab('s', 'b.md')).toBe(true)
    expect(navigate).toHaveBeenCalledTimes(2)
  })
})

describe('a linked tab that fails to move', () => {
  it('is told to the window, and the rest still move', async () => {
    const failure = new Error('the plugin is gone')
    const other = vi.fn<Navigate>()
    const { sync, onError } = setup(
      {
        'plex:1': { navigate: () => Promise.reject(failure) },
        'files:1': { navigate: other },
      },
      createLinkedTabs('plex:1', 'files:1'),
    )

    expect(sync.syncPathFromTab('s', 'a.md')).toBe(true)
    expect(other).toHaveBeenCalledOnce()
    await vi.waitFor(() => expect(onError).toHaveBeenCalledWith(failure))
  })

  it('reports a move the core did not take', async () => {
    const failure = new Error('the core is gone')
    const { sync, sendSync, onError } = setup({}, createLinkedTabs('plex:elsewhere'))
    sendSync.mockRejectedValueOnce(failure)

    sync.syncPathFromTab('s', 'a.md')

    await vi.waitFor(() => expect(onError).toHaveBeenCalledWith(failure))
  })
})
