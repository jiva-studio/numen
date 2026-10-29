/**
 * The open files a command can access, across every store the window keeps them in.
 * An identity is resolved by the store holding it, and an identity no store holds
 * resolves to nothing.
 */
import type { Notes, Store } from '../../types'

export const createNotes = (
  stores: readonly Store[],
  noteOpeners: Pick<Notes, 'openFile' | 'openNewFile'>,
): Notes => {
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
    openFile: noteOpeners.openFile,
    openNewFile: noteOpeners.openNewFile,
  }
}
