import { describe, expect, it } from 'vitest'
import {
  ASKING_DETAIL,
  createLinkMenuItems,
  DETACH_ID,
  getMoveTarget,
  getWindowName,
  MOVE_ID,
  UNLINK_ID,
} from './menu'

describe('createLinkMenuItems', () => {
  it('creates items for unlinked tab', () => {
    const tabs = [
      { id: 'tab-1', title: 'Note 1', kind: 'note', isLinked: false },
      { id: 'tab-2', title: '', kind: 'files', isLinked: false },
    ]
    const items = createLinkMenuItems({ eligibleTabs: tabs })
    expect(items).toHaveLength(2)
    expect(items[0]).toEqual({
      id: 'tab-1',
      text: 'Note 1',
      detail: 'note',
      group: 'Link with tab',
    })
    expect(items[1]).toEqual({
      id: 'tab-2',
      text: 'files',
      detail: 'files',
      group: 'Link with tab',
    })
  })

  it('includes detach item when canDetach is true', () => {
    const tabs = [{ id: 'tab-1', title: 'Note 1', kind: 'note', isLinked: false }]
    const items = createLinkMenuItems({ eligibleTabs: tabs, canDetach: true })

    expect(items).toEqual([
      { id: DETACH_ID, text: 'Detach tab' },
      { id: 'tab-1', text: 'Note 1', detail: 'note', group: 'Link with tab' },
    ])
  })

  it('includes unlink item when tab is already linked with or without target title', () => {
    const tabs = [{ id: 'tab-2', title: 'Files', kind: 'files', isLinked: false }]
    const itemsWithTitle = createLinkMenuItems({
      eligibleTabs: tabs,
      isLinked: true,
      linkedTargetTitle: 'Note 1',
    })
    expect(itemsWithTitle[0]).toEqual({
      id: UNLINK_ID,
      text: 'Unlink tab',
      detail: 'Linked with Note 1',
    })

    const itemsWithoutTitle = createLinkMenuItems({
      eligibleTabs: tabs,
      isLinked: true,
      linkedTargetTitle: '',
    })
    expect(itemsWithoutTitle[0]).toEqual({
      id: UNLINK_ID,
      text: 'Unlink tab',
    })
  })

  it('offers the unlink item without a detail when no target title is given', () => {
    const items = createLinkMenuItems({ eligibleTabs: [], isLinked: true })

    expect(items).toEqual([{ id: UNLINK_ID, text: 'Unlink tab' }])
  })

  it('offers no unlink item when the tab is not returned to be linked', () => {
    const tabs = [{ id: 'tab-1', title: 'Note 1', kind: 'note', isLinked: false }]

    expect(createLinkMenuItems({ eligibleTabs: tabs }).map((item) => item.id)).toEqual(['tab-1'])
  })
})

const MAIN = {
  id: 'window-a',
  isMain: true,
  activeTitle: 'Orbits',
  activeKind: 'note',
  tabCount: 3,
}
const OTHER = {
  id: 'window-b',
  isMain: false,
  activeTitle: 'Tides',
  activeKind: 'plex',
  tabCount: 1,
}

describe('the Move to window items', () => {
  it('lists the other windows, marks the main window, and names each by its active tab and other tab count', () => {
    const items = createLinkMenuItems({ eligibleTabs: [], windows: [MAIN, OTHER] })

    expect(items).toEqual([
      {
        id: MOVE_ID,
        text: 'Move to window',
        children: [
          { id: '__move__:window-a', text: 'Orbits (+2)', detail: 'Main window' },
          { id: '__move__:window-b', text: 'Tides' },
        ],
      },
    ])
  })

  it('omits the submenu when no other window is open', () => {
    expect(
      createLinkMenuItems({ eligibleTabs: [], canDetach: true }).map((item) => item.id),
    ).toEqual([DETACH_ID])
  })

  it('disables Move to window and Detach tab with the reason when the tab is asking', () => {
    const items = createLinkMenuItems({
      eligibleTabs: [],
      canDetach: true,
      windows: [OTHER],
      isAsking: true,
    })

    expect(items).toEqual([
      { id: DETACH_ID, text: 'Detach tab', isDisabled: true, detail: ASKING_DETAIL },
      { id: MOVE_ID, text: 'Move to window', isDisabled: true, detail: ASKING_DETAIL },
    ])
  })

  it('names a window without an active title by its kind, or as empty', () => {
    expect(getWindowName({ ...OTHER, activeTitle: '' })).toBe('plex')
    expect(getWindowName({ ...OTHER, activeTitle: '', activeKind: '', tabCount: 0 })).toBe(
      'Empty window',
    )
  })

  it('returns the window id for a move item and null for other items', () => {
    expect(getMoveTarget('__move__:window-b')).toBe('window-b')
    expect(getMoveTarget(MOVE_ID)).toBeNull()
    expect(getMoveTarget('tab-1')).toBeNull()
    expect(getMoveTarget(null)).toBeNull()
  })
})
