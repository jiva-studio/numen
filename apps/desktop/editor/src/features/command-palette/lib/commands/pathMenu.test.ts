/**
 * The window's commands over a row of the vault, asked without a screen.
 */
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
    expect(getItemIds(false)).toStrictEqual(['copy', 'remove'])
  })

  it('offer the path and the removal, parted by a rule', () => {
    expect(getGroupedIds(false)).toStrictEqual(['copy', '— remove'])
  })

  it('offer only the removal over a selection of several', () => {
    expect(
      createPathCommands({ isNote: true, isFolder: false }, true).map((one) => one.id),
    ).toStrictEqual(['remove'])
  })

  it('includes file menu contributions before copy and remove on a file that is no note', () => {
    const contributions = [
      { id: 'file-menu:create-text:whisper:', text: 'Transcribe with Whisper', group: 'file' },
      { id: 'file-menu:create-text:proofread:v1', text: 'Proofread with Claude', group: 'file' },
    ]
    expect(
      createPathCommands({ isNote: false, isFolder: false }, false, contributions).map(
        (one) => one.id,
      ),
    ).toStrictEqual([
      'file-menu:create-text:whisper:',
      'file-menu:create-text:proofread:v1',
      'copy',
      'remove',
    ])
  })
})

describe('the commands over a note', () => {
  it('offers core note commands without plugin contributions', () => {
    expect(getGroupedIds(true)).toStrictEqual(['read', '— copy', '— remove'])
  })

  it('includes plugin contributions when provided', () => {
    const contributions = [
      { id: 'show', text: 'Show in the graph', group: 'open' },
      { id: 'child', text: 'New child note', parentId: 'relation' },
      { id: 'parent', text: 'New parent note', parentId: 'relation' },
      { id: 'jump', text: 'New jump note', parentId: 'relation' },
      { id: 'question', text: 'Ask about it', group: 'chat' },
    ]
    const grouped = groupItems(
      createPathCommands({ isNote: true, isFolder: false }, false, contributions),
    )
    expect(grouped.map((one) => `${one.isRule ? '— ' : ''}${one.id}`)).toStrictEqual([
      'read',
      'show',
      '— copy',
      '— relation',
      '— question',
      '— remove',
    ])
    const relationItem = grouped.find((one) => one.id === 'relation')
    expect(relationItem?.children?.map((c) => c.id)).toStrictEqual(['child', 'parent', 'jump'])
  })
})
