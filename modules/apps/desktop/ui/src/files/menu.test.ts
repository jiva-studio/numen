/**
 * What the menu on a row of the tree offers, asked without a screen.
 *
 * A run is offered on the rows the vault holds that kind of file at and on no
 * others, so the negatives are the ones worth having.
 */
import { describe, expect, it } from 'vitest'
import { banded } from '@numen/ui'
import { cannotRun, runsAgain } from '../commanding'
import type { Source } from '../core'
import { itemsFor, OFFERED } from './menu'

/** What a row of that kind offers, by the identity of each item. */
const on = (source: Source, folder = false): readonly string[] =>
  itemsFor({ source, folder }, false).map((one) => one.id)

/** The same, as it is drawn: each item, and the rule standing above it. */
const drawn = (source: Source): readonly string[] =>
  banded(itemsFor({ source, folder: false }, false)).map(
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
    const made = itemsFor(null, false).map((one) => one.id)
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
      'newFolder',
      'rename',
      'copy',
      '— remove',
    ])
  })
})

describe('the menu where this build cannot do a run at all', () => {
  it('offers the recording nothing, and leaves the scan its own run', () => {
    cannotRun('transcribe')
    try {
      expect(on('recording')).not.toContain('transcribe')
      expect(on('book')).toContain('recognise')
    } finally {
      runsAgain()
    }
  })
})

describe('what the menu offers anywhere', () => {
  it('holds both runs, so choosing one is the menu’s own', () => {
    expect(OFFERED.has('transcribe')).toBe(true)
    expect(OFFERED.has('recognise')).toBe(true)
  })
})
