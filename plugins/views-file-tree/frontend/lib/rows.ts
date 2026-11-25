/**
 * The listing as the tree draws it.
 *
 * What a file is — where it is filed, what kind it is — stays this window's.
 * The tree takes an identity, a name, and whether a row opens.
 */
import type { Row, RowStatus } from '@numen/ui'
import type { ListingRow } from '../types'
import { WORDS as words } from '../words'

function getStatus(entry: ListingRow['entry']): RowStatus | null {
  if (entry.isDownloading) {
    const { downloadProgress } = entry
    return {
      kind: 'loading',
      label: words.downloading,
      ...(downloadProgress === undefined ? {} : { progress: downloadProgress }),
    }
  }
  if (entry.isAbsent) return { kind: 'cloud', label: words.absent }
  return null
}

/** Every entry as a row, and the rows under it as its own. */
export function getRows(entries: readonly ListingRow[]): Row[] {
  return entries.map((one) => {
    const status = getStatus(one.entry)
    return {
      id: one.entry.path,
      name: one.entry.name,
      hasChildren: one.entry.isFolder,
      rows: getRows(one.rows),
      ...(status ? { status } : {}),
    }
  })
}
