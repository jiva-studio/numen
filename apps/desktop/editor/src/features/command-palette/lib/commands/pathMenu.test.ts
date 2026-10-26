/**
 * The window's commands over a row of the vault, asked without a screen.
 *
 * A run is offered on the rows the vault holds that kind of file at and on no
 * others, so the negatives are the ones worth having.
 */
import { describe, expect, it } from 'vitest'
import { groupItems } from '@numen/ui'
import type { Source } from '@/entities/file'
import { createPathCommands, type RunGuard } from './pathMenu'

/** A window that has been told nothing, which can do every run. */
const canRunAnything: RunGuard = () => true

/** What a row of that kind offers, by the identity of each item. */
const getItemIds = (
  source: Source,
  isFolder = false,
  canRun: RunGuard = canRunAnything,
): readonly string[] => createPathCommands({ source, isFolder }, false, canRun).map((one) => one.id)

/** The same, as it is drawn: each item, and the rule standing above it. */
const getGroupedIds = (source: Source): readonly string[] =>
  groupItems(createPathCommands({ source, isFolder: false }, false, canRunAnything)).map(
    (one) => `${one.isRule ? '— ' : ''}${one.id}`,
  )

describe('the commands over a row standing for a recording', () => {
  it('offer the file commands only', () => {
    expect(getItemIds('recording')).toStrictEqual(getItemIds('other'))
  })
})

describe('the commands over a row standing for an address', () => {
  it('offer the runs over what is at the address', () => {
    expect(getItemIds('url')).toStrictEqual(
      expect.arrayContaining(['downloadText', 'downloadCopy', 'deleteText', 'deleteCopy']),
    )
  })

  // A build that cannot reach one offers none of them, which is how a person
  // is told this build does not do it.
  it('offer none of them where the build cannot do them', () => {
    const proposed = getItemIds('url', false, () => false)
    for (const one of ['downloadText', 'downloadCopy', 'deleteText', 'deleteCopy']) {
      expect(proposed).not.toContain(one)
    }
  })

  it('come back to the plain file commands where none of them can be run', () => {
    expect(getItemIds('url', false, () => false)).toStrictEqual(
      getItemIds('book', false, () => false),
    )
  })
})

describe('the commands over a row standing for a scanned document', () => {
  it('offers the file commands only', () => {
    expect(getItemIds('book')).toStrictEqual(getItemIds('other'))
  })
})

describe('the commands over a row standing for anything else', () => {
  it('offer no run on a note', () => {
    expect(getItemIds('note')).not.toContain('downloadText')
  })

  it('offer no run on a file the vault holds no source for', () => {
    expect(getItemIds('other')).not.toContain('downloadText')
  })

  // A folder of addresses is not an address, and a run is over one file.
  it('offer no run on a folder', () => {
    expect(getItemIds('url', true)).not.toContain('downloadText')
  })

  it('offer only the removal over a selection of several', () => {
    expect(
      createPathCommands({ source: 'note', isFolder: false }, true, canRunAnything).map(
        (one) => one.id,
      ),
    ).toStrictEqual(['remove'])
  })
})

describe('the commands over a note', () => {
  it('offers core note commands without plugin contributions', () => {
    expect(getGroupedIds('note')).toStrictEqual(['read', '— copy', '— remove'])
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
      createPathCommands({ source: 'note', isFolder: false }, false, canRunAnything, contributions),
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

describe('where a run remains among the commands', () => {
  it('remains after the path and before the remove, parted from both by a rule', () => {
    expect(getGroupedIds('url')).toStrictEqual([
      'copy',
      '— downloadText',
      'downloadCopy',
      'deleteText',
      'deleteCopy',
      '— remove',
    ])
  })

  it('leaves no rule behind on a row that has no run', () => {
    expect(getGroupedIds('other')).toStrictEqual(['copy', '— remove'])
  })
})
