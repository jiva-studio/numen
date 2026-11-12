/**
 * What a drag from outside the window carries, as the files tree reads it.
 *
 * A dropped file is a path and is handled by the window's own drag and drop.
 * Any other data, such as an address dragged out of a browser under
 * `text/uri-list`, is read here and passed to the plugin that handles it.
 */
import type { DroppedData } from '@numen/plugin-sdk'

/** The part of a drag's data transfer that the tree reads; a `DataTransfer` has it. */
export interface DragTransfer {
  readonly types: readonly string[]
  getData(type: string): string
}

/** The type a drag of files names; files reach the window's own drag and drop. */
const FILES_TYPE = 'Files'

/** The types of data a drag carries, besides files. */
export const getDragTypes = (transfer: DragTransfer | null | undefined): readonly string[] =>
  [...(transfer?.types ?? [])].filter((type) => type !== FILES_TYPE)

/** The data a drag dropped over the tree includes, by type, leaving out types that carry nothing. */
export const readDroppedData = (transfer: DragTransfer | null | undefined): DroppedData => {
  const data: Record<string, string> = {}
  for (const type of getDragTypes(transfer)) {
    const value = transfer?.getData(type) ?? ''
    if (value) data[type] = value
  }
  return data
}
