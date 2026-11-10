import { describe, expect, it, vi } from 'vitest'
import { useTabLinks } from './useTabLinks'
import type { TabLinksSnapshot } from '@/entities/tab'

const SNAPSHOT: TabLinksSnapshot = {
  groups: [
    { id: 'group-1', colorIndex: 1, tabs: ['plex:1', 'files:2'], path: '', title: '' },
    { id: 'group-2', colorIndex: 2, tabs: ['plex:3', 'plex:4', 'plex:5'], path: '', title: '' },
  ],
  tabs: [
    { id: 'plex:1', kind: 'plex', title: 'Orbits', windowId: 'window-a' },
    { id: 'files:2', kind: 'files', title: 'Files', windowId: 'window-b' },
  ],
  windows: [
    { id: 'window-a', isMain: true, activeTitle: 'Orbits', activeKind: 'plex', tabCount: 1 },
    { id: 'window-b', isMain: false, activeTitle: 'Files', activeKind: 'files', tabCount: 1 },
  ],
}

const setup = () => {
  const write = vi.fn((_change: unknown) => Promise.resolve())
  const onError = vi.fn()
  return { write, onError, tabLinks: useTabLinks({ write, onError }) }
}

describe('useTabLinks snapshot', () => {
  it('has no groups before the first snapshot', () => {
    const { tabLinks } = setup()

    expect(tabLinks.groups.size).toBe(0)
    expect(tabLinks.tabs).toStrictEqual([])
    expect(tabLinks.getGroupOf('plex:1')).toBeNull()
    expect(tabLinks.getLinkedTargets('plex:1')).toStrictEqual([])
  })

  it('stores the groups and tabs of the last snapshot', () => {
    const { tabLinks } = setup()
    tabLinks.applySnapshot(SNAPSHOT)

    expect(tabLinks.getGroupOf('files:2')?.id).toBe('group-1')
    expect(tabLinks.getLinkedTargets('plex:4')).toStrictEqual(['plex:3', 'plex:5'])
    expect(tabLinks.tabs.map((tab) => tab.windowId)).toStrictEqual(['window-a', 'window-b'])
    expect(tabLinks.windows).toStrictEqual(SNAPSHOT.windows)

    tabLinks.applySnapshot({ groups: [SNAPSHOT.groups[1]!], tabs: [], windows: [] })

    expect(tabLinks.getGroupOf('plex:1')).toBeNull()
    expect(tabLinks.groups.size).toBe(1)
    expect(tabLinks.tabs).toStrictEqual([])
    expect(tabLinks.windows).toStrictEqual([])
  })
})

describe('useTabLinks changes', () => {
  it('writes a change to the core and updates the groups on the next snapshot', async () => {
    const { tabLinks, write } = setup()

    tabLinks.linkTabs('plex:1', 'files:2')

    await vi.waitFor(() =>
      expect(write).toHaveBeenCalledWith({ link: { source: 'plex:1', target: 'files:2' } }),
    )
    expect(tabLinks.getGroupOf('plex:1')).toBeNull()
  })

  it('writes no link for an empty tab id or a tab linked to itself', () => {
    const { tabLinks, write } = setup()

    tabLinks.linkTabs('', 'files:2')
    tabLinks.linkTabs('plex:1', '')
    tabLinks.linkTabs('plex:1', 'plex:1')

    expect(write).not.toHaveBeenCalled()
  })

  it('unlinks a linked tab and writes nothing for an unlinked tab', async () => {
    const { tabLinks, write } = setup()
    tabLinks.applySnapshot(SNAPSHOT)

    tabLinks.unlinkTab('plex:9')
    expect(write).not.toHaveBeenCalled()

    tabLinks.unlinkTab('plex:1')
    await vi.waitFor(() => expect(write).toHaveBeenCalledWith({ unlink: 'plex:1' }))
  })

  it('reports an error when the core rejects a change', async () => {
    const { tabLinks, write, onError } = setup()
    const failure = new Error('the core is gone')
    write.mockRejectedValueOnce(failure)

    tabLinks.linkTabs('plex:1', 'files:2')
    await vi.waitFor(() => expect(onError).toHaveBeenCalledWith(failure))
  })
})

describe('useTabLinks write order', () => {
  it('writes a link and then an unlink in the order they were made', async () => {
    const { tabLinks, write } = setup()
    tabLinks.applySnapshot(SNAPSHOT)
    let finishLink = (): void => undefined
    write.mockImplementationOnce(
      () =>
        new Promise<void>((resolve) => {
          finishLink = resolve
        }),
    )

    tabLinks.linkTabs('plex:1', 'plex:3')
    tabLinks.unlinkTab('plex:1')
    await vi.waitFor(() => expect(write).toHaveBeenCalledOnce())
    finishLink()

    await vi.waitFor(() => expect(write).toHaveBeenCalledTimes(2))
    expect(write.mock.calls).toStrictEqual([
      [{ link: { source: 'plex:1', target: 'plex:3' } }],
      [{ unlink: 'plex:1' }],
    ])
  })

  it('writes the next change after the core rejects one', async () => {
    const { tabLinks, write, onError } = setup()
    tabLinks.applySnapshot(SNAPSHOT)
    write.mockRejectedValueOnce(new Error('the core is gone'))

    tabLinks.linkTabs('plex:1', 'plex:3')
    tabLinks.unlinkTab('plex:1')

    await vi.waitFor(() => expect(write).toHaveBeenCalledTimes(2))
    expect(onError).toHaveBeenCalledOnce()
  })
})
