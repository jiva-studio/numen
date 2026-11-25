/**
 * The listing as the tree draws it.
 *
 * What a file is — where it is filed, what kind it is — stays this window's.
 * The tree takes an identity, a name, and whether a row opens.
 */
import type { Row, RowBadge } from '@numen/ui'
import type { ListingRow } from '../types'
import { WORDS as words } from '../words'

/** What marks a file at the corner of its icon: a turning ring while it comes, a cloud while it is not here. */
function getBadge(entry: ListingRow['entry']): RowBadge | null {
  if (entry.isDownloading) return { kind: 'loading', label: words.downloading }
  if (entry.isAbsent) return { kind: 'cloud', label: words.absent }
  return null
}

/** Every entry as a row, and the rows under it as its own. */
export function getRows(entries: readonly ListingRow[]): Row[] {
  return entries.map((one) => {
    const badge = getBadge(one.entry)
    return {
      id: one.entry.path,
      name: one.entry.name,
      hasChildren: one.entry.isFolder,
      rows: getRows(one.rows),
      ...(one.entry.isAbsent ? { isMuted: true } : {}),
      ...(badge ? { badge } : {}),
    }
  })
}
