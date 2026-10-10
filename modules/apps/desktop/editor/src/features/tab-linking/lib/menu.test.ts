import { describe, expect, it } from 'vitest'
import { createLinkMenuItems, UNLINK_ID } from './menu'

describe('createLinkMenuItems', () => {
  it('creates items for unlinked tab', () => {
    const tabs = [
      { id: 'tab-1', title: 'Note 1', kind: 'note', isLinked: false },
      { id: 'tab-2', title: '', kind: 'files', isLinked: false },
    ]
    const items = createLinkMenuItems(tabs, false)
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

  it('includes unlink item when tab is already linked with or without target title', () => {
    const tabs = [{ id: 'tab-2', title: 'Files', kind: 'files', isLinked: false }]
    const itemsWithTitle = createLinkMenuItems(tabs, true, 'Note 1')
    expect(itemsWithTitle[0]).toEqual({
      id: UNLINK_ID,
      text: 'Unlink tab',
      detail: 'Linked with Note 1',
    })

    const itemsWithoutTitle = createLinkMenuItems(tabs, true, '')
    expect(itemsWithoutTitle[0]).toEqual({
      id: UNLINK_ID,
      text: 'Unlink tab',
    })
  })
})
