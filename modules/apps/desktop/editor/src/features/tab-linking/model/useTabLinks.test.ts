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
    expect(tabLinks.groups.size).toBe(1)
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

  it('merges two separate groups when linking tabs from each', () => {
    const tabLinks = useTabLinks()
    tabLinks.linkTabs('tab-1', 'tab-2')
    tabLinks.linkTabs('tab-3', 'tab-4')

    expect(tabLinks.groups.size).toBe(2)

    tabLinks.linkTabs('tab-1', 'tab-3')

    expect(tabLinks.groups.size).toBe(1)
    expect(tabLinks.getLinkedTargets('tab-1')).toEqual(['tab-2', 'tab-3', 'tab-4'])
  })

  it('adds source to existing target group when only target is in a group', () => {
    const tabLinks = useTabLinks()
    tabLinks.linkTabs('tab-2', 'tab-3')
    tabLinks.linkTabs('tab-1', 'tab-2')

    expect(tabLinks.getLinkedTargets('tab-1')).toEqual(['tab-2', 'tab-3'])
  })

  it('ignores linking when tabs are invalid or already in same group', () => {
    const tabLinks = useTabLinks()
    tabLinks.linkTabs('', 'tab-2')
    tabLinks.linkTabs('tab-1', '')
    tabLinks.linkTabs('tab-1', 'tab-1')
    expect(tabLinks.groups.size).toBe(0)

    tabLinks.linkTabs('tab-1', 'tab-2')
    tabLinks.linkTabs('tab-1', 'tab-2')
    expect(tabLinks.getLinkedTargets('tab-1')).toEqual(['tab-2'])
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

  it('handles unlinking a tab that is not in any group', () => {
    const tabLinks = useTabLinks()
    tabLinks.unlinkTab('tab-unlinked')
    expect(tabLinks.getLinkedTargets('tab-unlinked')).toEqual([])
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

  it('allocates and recycles color indices correctly', () => {
    const tabLinks = useTabLinks()
    tabLinks.linkTabs('a1', 'a2')
    tabLinks.linkTabs('b1', 'b2')
    tabLinks.linkTabs('c1', 'c2')
    tabLinks.linkTabs('d1', 'd2')
    tabLinks.linkTabs('e1', 'e2')

    expect(tabLinks.getGroupOf('a1')?.colorIndex).toBe(1)
    expect(tabLinks.getGroupOf('b1')?.colorIndex).toBe(2)
    expect(tabLinks.getGroupOf('c1')?.colorIndex).toBe(3)
    expect(tabLinks.getGroupOf('d1')?.colorIndex).toBe(4)
    expect(tabLinks.getGroupOf('e1')?.colorIndex).toBe(1)
  })
})
