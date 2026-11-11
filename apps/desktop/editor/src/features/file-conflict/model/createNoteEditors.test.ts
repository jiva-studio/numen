import { describe, expect, it } from 'vitest'
import { createTestTabs } from '@/testing/tabs'
import { createMessageLog } from '@/shared/model/notices/messages'
import { createNoteEditors } from './createNoteEditors'

const create = () => createNoteEditors({ log: createMessageLog(), windowTabs: createTestTabs() })

describe('the editors of a window', () => {
  it('hold no note before one is opened', () => {
    const editors = create()

    expect(editors.store.has('note:1')).toBe(false)
    expect(editors.store.getTabAt('Tides.md')).toBeNull()
    expect(editors.getTitle('note:1')).toBe('')
  })

  it('stop flushing the notes once closed', () => {
    const editors = create()

    editors.close()

    expect(editors.fileFlush.addHandler(() => Promise.resolve())).toBeTypeOf('function')
  })
})
