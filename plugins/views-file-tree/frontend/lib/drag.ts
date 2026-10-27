/**
 * What a drag from outside the window carries, as the files tree reads it.
 *
 * A file dropped in is a path and reaches the window's own drag and drop. Any other data, such as
 * an address dragged out of a browser under `text/uri-list`, is read here and handed to the plugin
 * that takes it.
 */
import type { DroppedData } from '@numen/plugin-sdk'

/** The type a drag of files names; files reach the window's own drag and drop. */
const FILES_TYPE = 'Files'

/** The types of data a drag carries, besides files. */
export const getDragTypes = (transfer: DataTransfer | null | undefined): readonly string[] =>
  [...(transfer?.types ?? [])].filter((type) => type !== FILES_TYPE)

/** The data a drag let go over the tree carries, by type, leaving out types that carry nothing. */
export const readDroppedData = (transfer: DataTransfer | null | undefined): DroppedData => {
  const data: Record<string, string> = {}
  for (const type of getDragTypes(transfer)) {
    const value = transfer?.getData(type) ?? ''
    if (value) data[type] = value
  }
  return data
}
