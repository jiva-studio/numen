import { describe, expect, it, vi } from 'vitest'
import { useTabLinks } from './useTabLinks'
import type { TabLinksSnapshot } from '../types'

const SNAPSHOT: TabLinksSnapshot = {
  groups: [
    { id: 'group-1', colorIndex: 1, tabs: ['plex:1', 'files:2'] },
    { id: 'group-2', colorIndex: 2, tabs: ['plex:3', 'plex:4', 'plex:5'] },
  ],
  tabs: [
    { id: 'plex:1', kind: 'plex', title: 'Orbits', windowId: 'window-a' },
    { id: 'files:2', kind: 'files', title: 'Files', windowId: 'window-b' },
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

    tabLinks.applySnapshot({ groups: [SNAPSHOT.groups[1]!], tabs: [] })

    expect(tabLinks.getGroupOf('plex:1')).toBeNull()
    expect(tabLinks.groups.size).toBe(1)
    expect(tabLinks.tabs).toStrictEqual([])
  })
})

describe('useTabLinks changes', () => {
  it('writes a change to the core and updates the groups on the next snapshot', () => {
    const { tabLinks, write } = setup()

    tabLinks.linkTabs('plex:1', 'files:2')

    expect(write).toHaveBeenCalledWith({ link: { source: 'plex:1', target: 'files:2' } })
    expect(tabLinks.getGroupOf('plex:1')).toBeNull()
  })

  it('writes no link for an empty tab id or a tab linked to itself', () => {
    const { tabLinks, write } = setup()

    tabLinks.linkTabs('', 'files:2')
    tabLinks.linkTabs('plex:1', '')
    tabLinks.linkTabs('plex:1', 'plex:1')

    expect(write).not.toHaveBeenCalled()
  })

  it('unlinks a linked tab and writes nothing for an unlinked tab', () => {
    const { tabLinks, write } = setup()
    tabLinks.applySnapshot(SNAPSHOT)

    tabLinks.unlinkTab('plex:9')
    expect(write).not.toHaveBeenCalled()

    tabLinks.unlinkTab('plex:1')
    expect(write).toHaveBeenCalledWith({ unlink: 'plex:1' })
  })

  it('reports an error when the core rejects a change', async () => {
    const { tabLinks, write, onError } = setup()
    const failure = new Error('the core is gone')
    write.mockRejectedValueOnce(failure)

    tabLinks.linkTabs('plex:1', 'files:2')
    await vi.waitFor(() => expect(onError).toHaveBeenCalledWith(failure))
  })
})
