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

describe('the link groups mirrored from the core', () => {
  it('are empty until the core sends a snapshot', () => {
    const { tabLinks } = setup()

    expect(tabLinks.groups.size).toBe(0)
    expect(tabLinks.tabs).toStrictEqual([])
    expect(tabLinks.getGroupOf('plex:1')).toBeNull()
    expect(tabLinks.getLinkedTargets('plex:1')).toStrictEqual([])
  })

  it('are the groups and tabs of the last snapshot, across windows', () => {
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

describe('a change to the links', () => {
  it('is written to the core and changes nothing until the snapshot comes back', () => {
    const { tabLinks, write } = setup()

    tabLinks.linkTabs('plex:1', 'files:2')

    expect(write).toHaveBeenCalledWith({ link: { source: 'plex:1', target: 'files:2' } })
    expect(tabLinks.getGroupOf('plex:1')).toBeNull()
  })

  it('links nothing for a missing tab or a tab to itself', () => {
    const { tabLinks, write } = setup()

    tabLinks.linkTabs('', 'files:2')
    tabLinks.linkTabs('plex:1', '')
    tabLinks.linkTabs('plex:1', 'plex:1')

    expect(write).not.toHaveBeenCalled()
  })

  it('unlinks a linked tab, and writes nothing for a tab in no group', () => {
    const { tabLinks, write } = setup()
    tabLinks.applySnapshot(SNAPSHOT)

    tabLinks.unlinkTab('plex:9')
    expect(write).not.toHaveBeenCalled()

    tabLinks.unlinkTab('plex:1')
    expect(write).toHaveBeenCalledWith({ unlink: 'plex:1' })
  })

  it('reports a change the core did not take', async () => {
    const { tabLinks, write, onError } = setup()
    const refused = new Error('the core is gone')
    write.mockRejectedValueOnce(refused)

    tabLinks.linkTabs('plex:1', 'files:2')
    await vi.waitFor(() => expect(onError).toHaveBeenCalledWith(refused))
  })
})
