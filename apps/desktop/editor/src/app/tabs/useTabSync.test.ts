import { describe, expect, it, vi } from 'vitest'
import { useTabLinks } from '@/features/tab-linking'
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

const createLinkedTabs = (...targets: string[]): TabSyncDeps['tabLinks'] => ({
  getLinkedTargets: () => targets,
})

describe('a path chosen in a linked tab', () => {
  it('moves every tab linked to it through its kind, with the state the tab holds', () => {
    const tabLinks = useTabLinks()
    tabLinks.linkTabs('plex:1', 'files:1')
    tabLinks.linkTabs('plex:1', 'other:1')
    const files = vi.fn<Navigate>()
    const other = vi.fn<Navigate>()
    const windowTabs = createWindowTabs({
      'plex:1': { navigate: vi.fn() },
      'files:1': { state: 'files state', navigate: files },
      'other:1': { state: 'other state', navigate: other },
    })

    const synced = createTabSync({ windowTabs, tabLinks, onError: vi.fn() }).syncPathFromTab(
      'plex:1',
      'test.md',
      'Test',
    )

    expect(synced).toBe(true)
    expect(files).toHaveBeenCalledWith('files state', 'test.md', 'Test')
    expect(other).toHaveBeenCalledWith('other state', 'test.md', 'Test')
  })

  it('leaves alone a linked tab whose kind cannot move, and one the window no longer holds', () => {
    const navigate = vi.fn<Navigate>()
    const windowTabs = createWindowTabs({ 'agent:1': {}, 'files:1': { navigate } })

    const synced = createTabSync({
      windowTabs,
      tabLinks: createLinkedTabs('agent:1', 'gone:1', 'files:1'),
      onError: vi.fn(),
    }).syncPathFromTab('s', 'a.md')

    expect(synced).toBe(true)
    expect(navigate).toHaveBeenCalledOnce()
    expect(navigate).toHaveBeenCalledWith(undefined, 'a.md', '')
  })

  it('moves nothing from a tab linked to none', () => {
    const navigate = vi.fn<Navigate>()
    const windowTabs = createWindowTabs({ 'files:1': { navigate } })

    expect(
      createTabSync({ windowTabs, tabLinks: useTabLinks(), onError: vi.fn() }).syncPathFromTab(
        's',
        'a.md',
      ),
    ).toBe(false)
    expect(navigate).not.toHaveBeenCalled()
  })
})

describe('createTabSync guards', () => {
  it('refuses an empty source tab id or an empty path even when targets are linked', () => {
    const navigate = vi.fn<Navigate>()
    const sync = createTabSync({
      windowTabs: createWindowTabs({ 'note:1': { navigate } }),
      tabLinks: createLinkedTabs('note:1'),
      onError: vi.fn(),
    })

    expect(sync.syncPathFromTab('', 'a.md')).toBe(false)
    expect(sync.syncPathFromTab('s', '')).toBe(false)
    expect(navigate).not.toHaveBeenCalled()
  })

  it('refuses a sync started while one is running, and accepts the next after it ends', () => {
    const inner: boolean[] = []
    const navigate = vi.fn<Navigate>(() => {
      inner.push(sync.syncPathFromTab('s', 'b.md'))
    })
    const sync = createTabSync({
      windowTabs: createWindowTabs({ 'note:1': { navigate } }),
      tabLinks: createLinkedTabs('note:1'),
      onError: vi.fn(),
    })

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
    const sync = createTabSync({
      windowTabs: createWindowTabs({ 'note:1': { navigate } }),
      tabLinks: createLinkedTabs('note:1'),
      onError: vi.fn(),
    })

    expect(() => sync.syncPathFromTab('s', 'a.md')).toThrow('boom')
    fail = false
    expect(sync.syncPathFromTab('s', 'a.md')).toBe(true)
  })
})

describe('a linked tab that fails to move', () => {
  it('is told to the window, and the rest still move', async () => {
    const onError = vi.fn()
    const failure = new Error('the plugin is gone')
    const other = vi.fn<Navigate>()
    const windowTabs = createWindowTabs({
      'plex:1': { navigate: () => Promise.reject(failure) },
      'files:1': { navigate: other },
    })

    const synced = createTabSync({
      windowTabs,
      tabLinks: createLinkedTabs('plex:1', 'files:1'),
      onError,
    }).syncPathFromTab('s', 'a.md')
    await Promise.resolve()

    expect(synced).toBe(true)
    expect(other).toHaveBeenCalledOnce()
    expect(onError).toHaveBeenCalledWith(failure)
  })
})
