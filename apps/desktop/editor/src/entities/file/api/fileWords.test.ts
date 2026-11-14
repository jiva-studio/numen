/**
 * A listing row, read into the window's own words.
 */
import { describe, expect, it } from 'vitest'
import type { Entry } from '@numen/core-client'
import { mapEntry } from './fileWords'

describe('a row of a listing', () => {
  it('maps an entry to path, name, folder flag, note flag and type', () => {
    const row = {
      path: 'Samples/Animals.md',
      name: 'Animals',
      isFolder: false,
      isNote: true,
      type: 'sample',
    } as Entry

    expect(mapEntry(row)).toStrictEqual({
      path: 'Samples/Animals.md',
      name: 'Animals',
      isFolder: false,
      isNote: true,
      type: 'sample',
    })
  })

  it('keeps a folder a folder', () => {
    const row = {
      path: 'Samples',
      name: 'Samples',
      isFolder: true,
      isNote: false,
      type: '',
    } as Entry

    expect(mapEntry(row)).toStrictEqual({
      path: 'Samples',
      name: 'Samples',
      isFolder: true,
      isNote: false,
      type: 'note',
    })
  })

  it('keeps the modified date of an entry that has one', () => {
    const row = {
      path: 'Samples/Animals.md',
      name: 'Animals',
      isFolder: false,
      isNote: true,
      type: 'sample',
      modifiedAtMs: 1762079400000n,
    } as Entry

    expect(mapEntry(row)).toStrictEqual({
      path: 'Samples/Animals.md',
      name: 'Animals',
      isFolder: false,
      isNote: true,
      type: 'sample',
      modifiedAt: 1762079400000,
    })
  })

  it('leaves the modified date out when the entry has none', () => {
    const row = {
      path: 'Samples/Animals.md',
      name: 'Animals',
      isFolder: false,
      isNote: true,
      type: 'sample',
      modifiedAtMs: 0n,
    } as Entry

    expect(mapEntry(row)).not.toHaveProperty('modifiedAt')
  })
})
