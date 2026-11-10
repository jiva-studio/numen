import { describe, expect, it, vi } from 'vitest'
import type { TabLinkGroup, TabPathSync } from '../lib/links'
import { createTabSync, type TabSyncDeps } from './useTabSync'

type Navigate = (state: unknown, path: string, title: string) => void | Promise<void>

const createPendingNavigation = () => new Promise<void>(() => undefined)

const createWindowTabs = (
  tabs: Record<string, { state?: unknown; navigate?: Navigate; path?: string }>,
) =>
  ({
    tabs: {
      getTab: (id: string) => {
        const one = tabs[id]
        if (!one) return null
        return {
          state: one.state,
          kind: {
            kind: id.split(':')[0],
            address: {
              ...(one.navigate ? { navigate: one.navigate } : {}),
              ...(one.path ? { getOpenTab: () => ({ path: one.path }) } : {}),
            },
          },
        }
      },
    },
  }) as unknown as TabSyncDeps['windowTabs']

const createLinkedTabs = (...targets: string[]): TabSyncDeps['tabLinks'] => {
  const group: TabLinkGroup = {
    id: 'group-1',
    colorIndex: 1,
    tabs: ['s', ...targets],
    path: '',
    title: '',
  }
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
  it('moves the linked tabs of this window and sends the sync to the core', () => {
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

  it('returns true when the only other tab of the group is in another window', () => {
    const { sync, sendSync } = setup({ s: {} }, createLinkedTabs('plex:elsewhere'))

    expect(sync.syncPathFromTab('s', 'a.md')).toBe(true)
    expect(sendSync).toHaveBeenCalledOnce()
  })

  it('skips a linked tab whose kind has no navigate', () => {
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

  it('suppresses one echo from a moved tab and syncs the same path selected later', () => {
    const navigate = vi.fn<Navigate>(createPendingNavigation)
    const { sync, sendSync } = setup(
      { s: { navigate: vi.fn() }, 'files:1': { navigate } },
      createLinkedTabs('files:1'),
    )

    sync.syncPathFromTab('s', 'a.md')
    expect(sync.syncPathFromTab('files:1', 'a.md')).toBe(true)
    expect(navigate).toHaveBeenCalledOnce()
    expect(sendSync).toHaveBeenCalledOnce()

    sync.syncPathFromTab('files:1', 'a.md')
    expect(sendSync).toHaveBeenCalledTimes(2)
  })

  it('syncs the path again after a moved tab fails to navigate to it', async () => {
    const failure = new Error('the plugin is gone')
    const navigate = vi.fn<Navigate>(() => Promise.reject(failure))
    const { sync, sendSync, onError } = setup(
      { s: { navigate: vi.fn() }, 'files:1': { navigate } },
      createLinkedTabs('files:1'),
    )

    sync.syncPathFromTab('s', 'a.md')
    await vi.waitFor(() => expect(onError).toHaveBeenCalledWith(failure))
    sync.syncPathFromTab('files:1', 'a.md')

    expect(sendSync).toHaveBeenCalledTimes(2)
  })

  it('syncs the path again after navigate throws on it', () => {
    const navigate = vi.fn<Navigate>(() => {
      throw new Error('boom')
    })
    const { sync, sendSync, onError } = setup(
      { 'files:1': { navigate } },
      createLinkedTabs('files:1'),
    )

    sync.applySync({ groupId: 'group-1', sourceTabId: 'plex:elsewhere', path: 'a.md', title: '' })
    sync.syncPathFromTab('files:1', 'a.md')

    expect(onError).toHaveBeenCalledWith(new Error('boom'))
    expect(sendSync).toHaveBeenCalledOnce()
  })
})

describe('applySync', () => {
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

  it('skips the tabs that show the path of a sync from this window', () => {
    const navigate = vi.fn<Navigate>()
    const { sync } = setup({ s: {}, 'files:1': { navigate } }, createLinkedTabs('files:1'))
    sync.syncPathFromTab('s', 'b.md', 'B')

    sync.applySync({ groupId: 'group-1', sourceTabId: 's', path: 'b.md', title: 'B' })

    expect(navigate).toHaveBeenCalledOnce()
  })

  it('moves the source tab back to the path of an earlier sync from this window', () => {
    const source = vi.fn<Navigate>(createPendingNavigation)
    const { sync, sendSync } = setup(
      { s: { navigate: source }, 'files:1': { navigate: vi.fn() } },
      createLinkedTabs('files:1'),
    )
    sync.syncPathFromTab('s', 'a.md', 'A')
    sync.syncPathFromTab('s', 'b.md', 'B')

    sync.applySync({ groupId: 'group-1', sourceTabId: 's', path: 'a.md', title: 'A' })
    sync.syncPathFromTab('s', 'a.md', 'A')

    expect(source).toHaveBeenCalledExactlyOnceWith(undefined, 'a.md', 'A')
    expect(sendSync).toHaveBeenCalledTimes(2)
  })

  it('does not send the sync back when a moved tab reports the path', () => {
    const navigate = vi.fn<Navigate>(createPendingNavigation)
    const { sync, sendSync } = setup({ 'files:1': { navigate } }, createLinkedTabs('files:1'))
    navigate.mockImplementationOnce(() => {
      sync.syncPathFromTab('files:1', 'b.md', 'B')
    })

    sync.applySync({ groupId: 'group-1', sourceTabId: 's', path: 'b.md', title: 'B' })
    sync.applySync({ groupId: 'group-1', sourceTabId: 's', path: 'c.md', title: 'C' })
    expect(sync.syncPathFromTab('files:1', 'c.md', 'C')).toBe(true)

    expect(navigate).toHaveBeenCalledTimes(2)
    expect(sendSync).not.toHaveBeenCalled()
  })

  it('ignores a sync for an unknown group', () => {
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
    const { sync, onError } = setup({ 'note:1': { navigate } }, createLinkedTabs('note:1'))

    expect(sync.syncPathFromTab('s', 'a.md')).toBe(true)
    expect(onError).toHaveBeenCalledWith(new Error('boom'))
    fail = false
    expect(sync.syncPathFromTab('s', 'b.md')).toBe(true)
    expect(navigate).toHaveBeenCalledTimes(2)
  })
})

describe('a linked tab that fails to move', () => {
  it('is sent to the window, and the rest still move', async () => {
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

  it('reports an error when the core rejects the sync', async () => {
    const failure = new Error('the core is gone')
    const { sync, sendSync, onError } = setup({}, createLinkedTabs('plex:elsewhere'))
    sendSync.mockRejectedValueOnce(failure)

    sync.syncPathFromTab('s', 'a.md')

    await vi.waitFor(() => expect(onError).toHaveBeenCalledWith(failure))
  })
})

describe('linked tabs in two windows', () => {
  const GROUP: TabLinkGroup = {
    id: 'group-1',
    colorIndex: 1,
    tabs: ['plex:a', 'plex:b'],
    path: '',
    title: '',
  }

  /** Returns one window with one plex tab whose navigate reports the path back at once. */
  const createWindow = (tabId: string, paths: Map<string, string>, outbox: TabPathSync[]) => {
    const window = {} as { sync: ReturnType<typeof createTabSync> }
    const navigate: Navigate = (_state, path) => {
      paths.set(tabId, path)
      window.sync.syncPathFromTab(tabId, path)
    }
    window.sync = createTabSync({
      windowTabs: createWindowTabs({ [tabId]: { navigate } }),
      tabLinks: {
        groups: new Map([[GROUP.id, GROUP]]),
        getGroupOf: (id) => (GROUP.tabs.includes(id) ? GROUP : null),
      },
      sendSync: (sync) => {
        outbox.push({ groupId: GROUP.id, ...sync })
        return Promise.resolve()
      },
      onError: () => undefined,
    })
    return window.sync
  }

  it('ends on one path after both windows navigate before either sync arrives', () => {
    const paths = new Map<string, string>()
    const outbox: TabPathSync[] = []
    const windowA = createWindow('plex:a', paths, outbox)
    const windowB = createWindow('plex:b', paths, outbox)

    paths.set('plex:a', 'first.md')
    windowA.syncPathFromTab('plex:a', 'first.md')
    paths.set('plex:b', 'second.md')
    windowB.syncPathFromTab('plex:b', 'second.md')
    for (const sync of outbox.splice(0)) {
      windowA.applySync(sync)
      windowB.applySync(sync)
    }

    expect(paths.get('plex:a')).toBe('second.md')
    expect(paths.get('plex:b')).toBe('second.md')
    expect(outbox).toEqual([])
  })
})

const createGroupAt = (path: string, ...tabs: string[]): TabLinkGroup => ({
  id: 'group-1',
  colorIndex: 1,
  tabs,
  path,
  title: path,
})

describe('applyGroupPaths', () => {
  it('moves a tab to the group path from the snapshot after its sync was dropped', () => {
    const navigate = vi.fn<Navigate>(createPendingNavigation)
    const { sync } = setup({ 'files:1': { navigate } }, createLinkedTabs('files:1'))
    sync.applySync({ groupId: 'group-1', sourceTabId: 's', path: 'a.md', title: 'A' })

    sync.applyGroupPaths([createGroupAt('b.md', 's', 'files:1')])
    sync.applyGroupPaths([createGroupAt('b.md', 's', 'files:1')])

    expect(navigate.mock.calls).toStrictEqual([
      [undefined, 'a.md', 'A'],
      [undefined, 'b.md', 'b.md'],
    ])
  })

  it('moves the tabs of a reconnected window that do not show the group path', () => {
    const staleTab = vi.fn<Navigate>()
    const currentTab = vi.fn<Navigate>()
    const { sync } = setup(
      {
        'files:1': { navigate: staleTab, path: 'a.md' },
        'plex:1': { navigate: currentTab, path: 'b.md' },
      },
      createLinkedTabs('files:1', 'plex:1'),
    )

    sync.applyGroupPaths([createGroupAt('b.md', 's', 'files:1', 'plex:1')])

    expect(staleTab).toHaveBeenCalledExactlyOnceWith(undefined, 'b.md', 'b.md')
    expect(currentTab).not.toHaveBeenCalled()
  })

  it('moves nothing for a group with no synced path', () => {
    const navigate = vi.fn<Navigate>()
    const { sync } = setup({ 'files:1': { navigate } }, createLinkedTabs('files:1'))

    sync.applyGroupPaths([createGroupAt('', 's', 'files:1')])

    expect(navigate).not.toHaveBeenCalled()
  })

  it('keeps the path of a sync from this window until the snapshot carries it', () => {
    const navigate = vi.fn<Navigate>(createPendingNavigation)
    const { sync } = setup({ s: {}, 'files:1': { navigate } }, createLinkedTabs('files:1'))
    sync.syncPathFromTab('s', 'b.md', 'B')

    sync.applyGroupPaths([createGroupAt('a.md', 's', 'files:1')])
    sync.applyGroupPaths([createGroupAt('b.md', 's', 'files:1')])
    sync.applyGroupPaths([createGroupAt('c.md', 's', 'files:1')])

    expect(navigate.mock.calls).toStrictEqual([
      [undefined, 'b.md', 'B'],
      [undefined, 'c.md', 'c.md'],
    ])
  })
})

describe('a linked tab that fails to navigate', () => {
  it('does not stop the other tabs, and the sync is still sent', () => {
    const other = vi.fn<Navigate>()
    const { sync, sendSync, onError } = setup(
      {
        'plex:1': {
          navigate: () => {
            throw new Error('the plugin is gone')
          },
        },
        'files:1': { navigate: other },
      },
      createLinkedTabs('plex:1', 'files:1'),
    )

    expect(sync.syncPathFromTab('s', 'a.md')).toBe(true)

    expect(sendSync).toHaveBeenCalledWith({ sourceTabId: 's', path: 'a.md', title: '' })
    expect(other).toHaveBeenCalledOnce()
    expect(onError).toHaveBeenCalledWith(new Error('the plugin is gone'))
  })

  it('sends the sync before the local moves', () => {
    const order: string[] = []
    const { sync, sendSync } = setup(
      {
        'files:1': {
          navigate: () => {
            order.push('navigate')
          },
        },
      },
      createLinkedTabs('files:1'),
    )
    sendSync.mockImplementation(() => {
      order.push('send')
      return Promise.resolve()
    })

    sync.syncPathFromTab('s', 'a.md')

    expect(order).toStrictEqual(['send', 'navigate'])
  })
})

describe('a tab renamed outside a sync', () => {
  it('moves to the old path on a later sync', () => {
    const navigate = vi.fn<Navigate>(createPendingNavigation)
    const { sync } = setup({ 'files:1': { navigate } }, createLinkedTabs('files:1'))
    sync.applySync({ groupId: 'group-1', sourceTabId: 's', path: 'a.md', title: 'A' })

    sync.setShownPath('files:1', 'renamed.md')
    sync.applySync({ groupId: 'group-1', sourceTabId: 's', path: 'a.md', title: 'A' })

    expect(navigate).toHaveBeenCalledTimes(2)
  })
})

describe('a moved tab that reports another path', () => {
  it('sends the reported path as a sync from the tab', () => {
    const { sync, sendSync } = setup(
      {
        'files:1': {
          navigate: (_state, path) => {
            sync.syncPathFromTab('files:1', `${path}/index.md`, 'Index')
          },
        },
      },
      createLinkedTabs('files:1'),
    )

    sync.applySync({ groupId: 'group-1', sourceTabId: 's', path: 'notes', title: 'Notes' })

    expect(sendSync).toHaveBeenCalledExactlyOnceWith({
      sourceTabId: 'files:1',
      path: 'notes/index.md',
      title: 'Index',
    })
  })
})

describe('a moved tab that settles without reporting the path', () => {
  it('syncs a later navigation to that path', async () => {
    let settle = (): void => undefined
    const navigate = vi.fn<Navigate>(() => new Promise<void>((resolve) => (settle = resolve)))
    const { sync, sendSync } = setup({ 'files:1': { navigate } }, createLinkedTabs('files:1'))
    sync.applySync({ groupId: 'group-1', sourceTabId: 's', path: 'a.md', title: 'A' })
    settle()
    await new Promise((resolve) => setTimeout(resolve, 0))

    sync.syncPathFromTab('files:1', 'a.md')

    expect(sendSync).toHaveBeenLastCalledWith({ sourceTabId: 'files:1', path: 'a.md', title: '' })
  })
})
