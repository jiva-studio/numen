/**
 * The tree is handed rows, not files. A folder opens; everything else does not,
 * and a folder holding nothing still opens onto nothing.
 */
import { describe, expect, it } from 'vitest'
import type { Entry } from '../types'
import type { ListingRow } from '../types'
import { getRows } from './rows'

const entry = (path: string, isFolder = false): Entry => ({
  path,
  name: path.split('/').at(-1) ?? path,
  isFolder,
  isNote: true,
  type: 'note',
})

const createListingRow = (path: string, isFolder = false, rows: ListingRow[] = []): ListingRow => ({
  entry: entry(path, isFolder),
  rows,
})

describe('the listing as rows', () => {
  it('is addressed by the path each file is filed under', () => {
    expect(getRows([createListingRow('physics/Optics.md')])[0]).toStrictEqual({
      id: 'physics/Optics.md',
      name: 'Optics.md',
      hasChildren: false,
      rows: [],
    })
  })

  it('opens a folder and nothing else', () => {
    const rows = getRows([createListingRow('physics', true), createListingRow('Note.md')])

    expect(rows.map((one) => one.hasChildren)).toStrictEqual([true, false])
  })

  it('carries what is under a folder as its own rows, however deep', () => {
    const rows = getRows([
      createListingRow('a', true, [createListingRow('a/b', true, [createListingRow('a/b/c.md')])]),
    ])

    expect(rows[0]?.rows?.[0]?.rows?.[0]?.id).toBe('a/b/c.md')
  })

  it('opens a folder holding nothing', () => {
    expect(getRows([createListingRow('empty', true)])[0]).toMatchObject({
      hasChildren: true,
      rows: [],
    })
  })
})

describe('the status of a row', () => {
  it('is a cloud for a file not on this device, a loading ring while it is brought, and none for a file here', () => {
    const rows = getRows([
      { entry: { ...entry('a.mp4'), isAbsent: true }, rows: [] },
      { entry: { ...entry('b.mp4'), isAbsent: true, isDownloading: true }, rows: [] },
      createListingRow('c.png'),
    ])

    expect(rows.map((one) => one.status)).toStrictEqual([
      { kind: 'cloud', label: 'Not on this device' },
      { kind: 'loading', label: 'Downloading…' },
      undefined,
    ])
  })

  it('carries the progress of a download into its loading ring, and none where it is not known', () => {
    const rows = getRows([
      {
        entry: { ...entry('a.mp4'), isAbsent: true, isDownloading: true, downloadProgress: 0.3 },
        rows: [],
      },
      { entry: { ...entry('b.mp4'), isAbsent: true, isDownloading: true }, rows: [] },
    ])

    expect(rows.map((one) => one.status)).toStrictEqual([
      { kind: 'loading', label: 'Downloading…', progress: 0.3 },
      { kind: 'loading', label: 'Downloading…' },
    ])
  })
})
