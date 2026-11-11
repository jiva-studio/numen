import { describe, expect, it } from 'vitest'
import { answerNotes } from '@/testing/noteCore'
import { asValue } from '@numen/core-client'
import { createTestTabs } from '@/testing/tabs'
import { createMessageLog } from '@/shared/model/notices/messages'
import type { AnyTabKind } from '@numen/plugin-loader'
import { createNoteEditors } from './createNoteEditors'

const create = (windowTabs = createTestTabs()) => {
  answerNotes({
    read: async () => asValue({ body: 'High tide at six.', at: '1 1 at:High tide at six.' }),
    write: async () => asValue({ body: '', at: '1 1 at:' }),
  })
  return createNoteEditors({ log: createMessageLog(), windowTabs })
}

describe('the editors of a window', () => {
  it('hold no note before one is opened', () => {
    const editors = create()

    expect(editors.store.has('note:1')).toBe(false)
    expect(editors.store.getTabAt('Tides.md')).toBeNull()
    expect(editors.getTitle('note:1')).toBe('')
  })

  it('hold an open note by its tab, and name it by its path', () => {
    const editors = create()

    editors.notes.open('note:1', 'Tides.md')

    expect(editors.store.has('note:1')).toBe(true)
    expect(editors.store.getPath('note:1')).toBe('Tides.md')
    expect(editors.store.getTabAt('Tides.md')).toBe('note:1')
    expect(editors.store.getTabAt('Moon.md')).toBeNull()
    expect(editors.getTitle('note:1')).toBe('Tides.md')
    expect(editors.store.asking('note:1')).toBe(false)
  })

  it('name a note by the title of the tab it is open in', async () => {
    const windowTabs = createTestTabs()
    windowTabs.groups.registerKinds([
      {
        kind: 'note',
        open: () => ({ title: 'Tide table' }),
        pane: {},
        getTitle: (state: { title: string }) => state.title,
      } as AnyTabKind,
    ])
    const editors = create(windowTabs)
    const id = await windowTabs.tabs.openTabOfKind('note', 'Tides.md')

    editors.notes.open(id, 'Tides.md')

    expect(editors.getTitle(id)).toBe('Tide table')
  })
})
