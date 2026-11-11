/**
 * The open files a command can access, across every store the window keeps them in.
 * An identity is resolved by the store holding it, and an identity no store holds
 * resolves to nothing.
 */
import type { Selector } from '@numen/plugin-loader'
import type { Store } from '../../types'
import type { CommandPlace, Notes } from '@/shared/model/command'

/** What opens a file: in the current tab or beside it, at a line of the note where one is given. */
export interface NoteOpeners {
  openFile(
    path: string,
    title: string,
    options:
      'current' | 'split' | { where: 'current' | 'split'; line?: number; selector?: Selector },
  ): void
}

const createPlacement = (
  placement: 'current' | 'split',
  place: CommandPlace | undefined,
): Parameters<NoteOpeners['openFile']>[2] => {
  const line = place?.line
  const selector = place?.selector
  if (line === undefined && selector === undefined) return placement
  const options: { where: 'current' | 'split'; line?: number; selector?: Selector } = {
    where: placement,
  }
  if (line !== undefined) options.line = line
  if (selector !== undefined) options.selector = selector
  return options
}

export const createNotes = (stores: readonly Store[], noteOpeners: NoteOpeners): Notes => {
  const holder = (id: string): Store | undefined => stores.find((one) => one.has(id))
  return {
    getTabAt: (path) => {
      for (const one of stores) {
        const tab = one.getTabAt(path)
        if (tab !== null) return tab
      }
      return null
    },
    getPath: (id) => holder(id)?.getPath(id) ?? id,
    asking: (id) => holder(id)?.asking(id) ?? false,
    flushPending: async (id) => {
      await holder(id)?.flushPending(id)
    },
    close: (id) => holder(id)?.close(id),
    openFile: (path, title, placement, place) =>
      noteOpeners.openFile(path, title, createPlacement(placement, place)),
  }
}
