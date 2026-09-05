/**
 * What the menu on a row of the tree offers, asked without a screen.
 *
 * A run is offered on the rows the vault holds that kind of file at and on no
 * others, so the negatives are the ones worth having.
 */
import { describe, expect, it } from 'vitest'
import { banded } from '@numen/ui'
import type { Source } from '../core'
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
const anything: RunGuard = () => true

/** What a row of that kind offers, by the identity of each item. */
const on = (source: Source, folder = false, canRun: RunGuard = anything): readonly string[] =>
  itemsFor({ source, folder }, false, canRun).map((one) => one.id)

/** The same, as it is drawn: each item, and the rule standing above it. */
const drawn = (source: Source): readonly string[] =>
  banded(itemsFor({ source, folder: false }, false, anything)).map(
    (one) => `${one.rule ? '— ' : ''}${one.id}`,
  )

describe('the menu on a row standing for a recording', () => {
  it('offers the recording to be transcribed', () => {
    expect(on('recording')).toContain('transcribe')
  })

  it('offers nothing to recognise, which is asked of a scan', () => {
    expect(on('recording')).not.toContain('recognise')
  })
})

describe('the menu on a row standing for a scanned document', () => {
  it('offers the text of it to be recognised', () => {
    expect(on('book')).toContain('recognise')
  })

  it('offers nothing to transcribe, which is asked of a recording', () => {
    expect(on('book')).not.toContain('transcribe')
  })
})

describe('the menu on a row standing for anything else', () => {
  it('offers neither run on a note', () => {
    expect(on('note')).not.toContain('transcribe')
    expect(on('note')).not.toContain('recognise')
  })

  it('offers neither run on a file the vault holds no source for', () => {
    expect(on('other')).not.toContain('transcribe')
    expect(on('other')).not.toContain('recognise')
  })

  // A folder of recordings is not a recording, and a run is over one file.
  it('offers neither run on a folder', () => {
    expect(on('recording', true)).not.toContain('transcribe')
    expect(on('book', true)).not.toContain('recognise')
  })

  it('offers neither off every row, where there is nothing to run it over', () => {
    const made = itemsFor(null, false, anything).map((one) => one.id)
    expect(made).not.toContain('transcribe')
    expect(made).not.toContain('recognise')
  })
})

describe('where a run stands in the menu', () => {
  it('stands after the path and before the remove, parted from both by a rule', () => {
    expect(drawn('recording')).toStrictEqual([
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
    expect(drawn('other')).toStrictEqual([
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

    expect(on('recording', false, but)).not.toContain('transcribe')
    expect(on('book', false, but)).toContain('recognise')
  })
})

describe('the four files the menu makes', () => {
  it('offers a preset off every row, beside the note, the deck and the stencil', () => {
    expect(itemsFor(null, false, anything).map((one) => one.id)).toStrictEqual([
      NEW_NOTE,
      NEW_DECK,
      NEW_STENCIL,
      NEW_PRESET,
      NEW_FOLDER,
    ])
  })

  it('offers a preset on a row of every kind', () => {
    for (const source of ['note', 'book', 'recording', 'other'] as Source[]) {
      expect(on(source), source).toContain(NEW_PRESET)
      expect(on(source, true), source).toContain(NEW_PRESET)
    }
  })

  it('offers a preset in the band the files stand in', () => {
    expect(drawn('note')).toContain(NEW_PRESET)
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
