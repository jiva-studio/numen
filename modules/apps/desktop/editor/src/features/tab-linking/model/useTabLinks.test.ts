import { describe, expect, it } from 'vitest'
import { useTabLinks } from './useTabLinks'

describe('useTabLinks', () => {
  it('links two tabs into a new group', () => {
    const tabLinks = useTabLinks()
    tabLinks.linkTabs('tab-1', 'tab-2')

    expect(tabLinks.getLinkedTargets('tab-1')).toEqual(['tab-2'])
    expect(tabLinks.getLinkedTargets('tab-2')).toEqual(['tab-1'])

    const group = tabLinks.getGroupOf('tab-1')
    expect(group).not.toBeNull()
    expect(group?.colorIndex).toBe(1)
    expect(group?.tabs).toEqual(['tab-1', 'tab-2'])
  })

  it('links three tabs into the same group', () => {
    const tabLinks = useTabLinks()
    tabLinks.linkTabs('tab-1', 'tab-2')
    tabLinks.linkTabs('tab-1', 'tab-3')

    expect(tabLinks.getLinkedTargets('tab-1')).toEqual(['tab-2', 'tab-3'])
    expect(tabLinks.getLinkedTargets('tab-2')).toEqual(['tab-1', 'tab-3'])
    expect(tabLinks.getLinkedTargets('tab-3')).toEqual(['tab-1', 'tab-2'])

    const group = tabLinks.getGroupOf('tab-3')
    expect(group?.colorIndex).toBe(1)
    expect(group?.tabs).toEqual(['tab-1', 'tab-2', 'tab-3'])
  })

  it('unlinks a single tab leaving remaining tabs linked', () => {
    const tabLinks = useTabLinks()
    tabLinks.linkTabs('tab-1', 'tab-2')
    tabLinks.linkTabs('tab-1', 'tab-3')

    tabLinks.unlinkTab('tab-2')

    expect(tabLinks.getLinkedTargets('tab-2')).toEqual([])
    expect(tabLinks.getLinkedTargets('tab-1')).toEqual(['tab-3'])
    expect(tabLinks.getLinkedTargets('tab-3')).toEqual(['tab-1'])
  })

  it('dissolves group when only one tab remains', () => {
    const tabLinks = useTabLinks()
    tabLinks.linkTabs('tab-1', 'tab-2')
    tabLinks.unlinkTab('tab-1')

    expect(tabLinks.getLinkedTargets('tab-1')).toEqual([])
    expect(tabLinks.getLinkedTargets('tab-2')).toEqual([])
    expect(tabLinks.getGroupOf('tab-2')).toBeNull()
  })

  it('removes closed tab correctly', () => {
    const tabLinks = useTabLinks()
    tabLinks.linkTabs('tab-1', 'tab-2')
    tabLinks.removeClosedTab('tab-1')

    expect(tabLinks.getLinkedTargets('tab-2')).toEqual([])
    expect(tabLinks.getGroupOf('tab-2')).toBeNull()
  })
})
