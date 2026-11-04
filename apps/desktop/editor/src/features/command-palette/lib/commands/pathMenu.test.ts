/** The window's commands on a vault row, run without a screen. */
import { describe, expect, it } from 'vitest'
import { groupItems } from '@numen/ui'
import { createPathCommands } from './pathMenu'

/** What a row of that kind offers, by the identity of each item. */
const getItemIds = (isNote: boolean, isFolder = false): readonly string[] =>
  createPathCommands({ isNote, isFolder }, false).map((one) => one.id)

/** The same, as it is drawn: each item, and the rule standing above it. */
const getGroupedIds = (isNote: boolean): readonly string[] =>
  groupItems(createPathCommands({ isNote, isFolder: false }, false)).map(
    (one) => `${one.isRule ? '— ' : ''}${one.id}`,
  )

describe('the commands over a row standing for a file', () => {
  it('offer the note menu only on a note', () => {
    expect(getItemIds(true)).not.toStrictEqual(getItemIds(false))
    expect(getItemIds(false)).toStrictEqual(['app:note.copyPath', 'app:note.trash'])
  })

  it('offer the path and the removal, parted by a rule', () => {
    expect(getGroupedIds(false)).toStrictEqual(['app:note.copyPath', '— app:note.trash'])
  })

  it('offer only the removal over a selection of several', () => {
    expect(
      createPathCommands({ isNote: true, isFolder: false }, true).map((one) => one.id),
    ).toStrictEqual(['app:note.trash'])
  })

  it('includes file menu contributions before copy and remove on a file that is no note', () => {
    const contributions = [
      {
        id: 'md.example.whisper:text.create@',
        text: 'Transcribe',
        detail: 'Whisper',
        group: 'file',
      },
      {
        id: 'md.example.proofread:text.create@v1',
        text: 'Proofread',
        detail: 'Claude',
        group: 'file',
      },
    ]
    const items = createPathCommands({ isNote: false, isFolder: false }, false, contributions)
    expect(items.map((one) => one.id)).toStrictEqual([
      'md.example.whisper:text.create@',
      'md.example.proofread:text.create@v1',
      'app:note.copyPath',
      'app:note.trash',
    ])
    expect(items[0]?.detail).toBe('Whisper')
    expect(items[1]?.detail).toBe('Claude')
  })
})

describe('the commands over a note', () => {
  it('offers core note commands without plugin contributions', () => {
    expect(getGroupedIds(true)).toStrictEqual([
      'app:note.open',
      '— app:note.copyPath',
      '— app:note.trash',
    ])
  })

  it('includes plugin contributions when provided', () => {
    const contributions = [
      { id: 'show', text: 'Show in the graph', group: 'open' },
      { id: 'app:note.createChild', text: 'New child note', parentId: 'relation' },
      { id: 'app:note.createParent', text: 'New parent note', parentId: 'relation' },
      { id: 'app:note.createJump', text: 'New jump note', parentId: 'relation' },
      { id: 'question', text: 'Ask about it', group: 'chat' },
    ]
    const grouped = groupItems(
      createPathCommands({ isNote: true, isFolder: false }, false, contributions),
    )
    expect(grouped.map((one) => `${one.isRule ? '— ' : ''}${one.id}`)).toStrictEqual([
      'app:note.open',
      'show',
      '— app:note.copyPath',
      '— relation',
      '— question',
      '— app:note.trash',
    ])
    const relationItem = grouped.find((one) => one.id === 'relation')
    expect(relationItem?.children?.map((c) => c.id)).toStrictEqual([
      'app:note.createChild',
      'app:note.createParent',
      'app:note.createJump',
    ])
  })
})
