/**
 * The listing as the tree draws it.
 *
 * What a file is — where it is filed, what kind it is — stays this window's.
 * The tree takes an identity, a name, and whether a row opens.
 */
import type { Row } from '@numen/ui'
import type { ListingRow } from '../types'

/** Every entry as a row, and the rows under it as its own. */
export function getRows(entries: readonly ListingRow[]): Row[] {
  return entries.map((one) => ({
    id: one.entry.path,
    name: one.entry.name,
    hasChildren: one.entry.isFolder,
    rows: getRows(one.rows),
  }))
}
