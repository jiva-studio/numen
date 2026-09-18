/**
 * What the menu on a row of the tree offers, asked without a screen.
 *
 * A run is offered on the rows the vault holds that kind of file at and on no
 * others, so the negatives are the ones worth having.
 */
import { describe, expect, it } from 'vitest'
import { groupItems } from '@numen/ui'
import type { Source } from '@/entities/file'
import {
  itemsFor,
  type RunGuard,
  NEW_DECK,
  NEW_FOLDER,
  NEW_NOTE,
  NEW_PRESET,
  NEW_STENCIL,
  OFFERED,
} from './menu'

/** A window that has been told nothing, which can do every run. */
const canRunAnything: RunGuard = () => true

/** What a row of that kind offers, by the identity of each item. */
const getItemIds = (
  source: Source,
  isFolder = false,
  canRun: RunGuard = canRunAnything,
): readonly string[] => itemsFor({ source, isFolder }, false, canRun).map((one) => one.id)

/** The same, as it is drawn: each item, and the rule standing above it. */
const getGroupedIds = (source: Source): readonly string[] =>
  groupItems(itemsFor({ source, isFolder: false }, false, canRunAnything)).map(
    (one) => `${one.isRule ? '— ' : ''}${one.id}`,
  )

describe('the menu on a row standing for a recording', () => {
  it('offers the recording to be transcribed', () => {
    expect(getItemIds('recording')).toContain('transcribe')
  })

  it('offers nothing to recognise, which is asked of a scan', () => {
    expect(getItemIds('recording')).not.toContain('recognise')
  })
})

describe('the menu on a row standing for a scanned document', () => {
  it('offers the text of it to be recognised', () => {
    expect(getItemIds('book')).toContain('recognise')
  })

  it('offers nothing to transcribe, which is asked of a recording', () => {
    expect(getItemIds('book')).not.toContain('transcribe')
  })
})

describe('the menu on a row standing for anything else', () => {
  it('offers neither run on a note', () => {
    expect(getItemIds('note')).not.toContain('transcribe')
    expect(getItemIds('note')).not.toContain('recognise')
  })

  it('offers neither run on a file the vault holds no source for', () => {
    expect(getItemIds('other')).not.toContain('transcribe')
    expect(getItemIds('other')).not.toContain('recognise')
  })

  // A folder of recordings is not a recording, and a run is over one file.
  it('offers neither run on a folder', () => {
    expect(getItemIds('recording', true)).not.toContain('transcribe')
    expect(getItemIds('book', true)).not.toContain('recognise')
  })

  it('offers neither off every row, where there is nothing to run it over', () => {
    const made = itemsFor(null, false, canRunAnything).map((one) => one.id)
    expect(made).not.toContain('transcribe')
    expect(made).not.toContain('recognise')
  })
})

describe('where a run stands in the menu', () => {
  it('stands after the path and before the remove, parted from both by a rule', () => {
    expect(getGroupedIds('recording')).toStrictEqual([
      'newNote',
      'newDeck',
      'newStencil',
      'newPreset',
      'newFolder',
      'rename',
      'copy',
      '— transcribe',
      '— remove',
    ])
  })

  it('leaves no rule behind on a row that has no run', () => {
    expect(getGroupedIds('other')).toStrictEqual([
      'newNote',
      'newDeck',
      'newStencil',
      'newPreset',
      'newFolder',
      'rename',
      'copy',
      '— remove',
    ])
  })
})

describe('the menu where this build cannot do a run at all', () => {
  it('offers the recording nothing, and leaves the scan its own run', () => {
    const but: RunGuard = (run) => run !== 'transcribe'

    expect(getItemIds('recording', false, but)).not.toContain('transcribe')
    expect(getItemIds('book', false, but)).toContain('recognise')
  })
})

describe('the four files the menu makes', () => {
  it('offers a preset off every row, beside the note, the deck and the stencil', () => {
    expect(itemsFor(null, false, canRunAnything).map((one) => one.id)).toStrictEqual([
      NEW_NOTE,
      NEW_DECK,
      NEW_STENCIL,
      NEW_PRESET,
      NEW_FOLDER,
    ])
  })

  it('offers a preset on a row of every kind', () => {
    for (const source of ['note', 'book', 'recording', 'other'] as Source[]) {
      expect(getItemIds(source), source).toContain(NEW_PRESET)
      expect(getItemIds(source, true), source).toContain(NEW_PRESET)
    }
  })

  it('offers a preset in the group the files stand in', () => {
    expect(getGroupedIds('note')).toContain(NEW_PRESET)
  })
})

describe('what the menu offers anywhere', () => {
  it('holds both runs, so choosing one is the menu’s own', () => {
    expect(OFFERED.has('transcribe')).toBe(true)
    expect(OFFERED.has('recognise')).toBe(true)
  })

  it('holds the preset, so choosing it is the menu’s own', () => {
    expect(OFFERED.has(NEW_PRESET)).toBe(true)
  })
})
