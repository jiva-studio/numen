/**
 * The open files a command reaches, over every store the window keeps them in.
 * An identity is answered by the store holding it, and one nobody holds by
 * nothing at all.
 */
import type { Notes, Store } from '../types'

export const createNotes = (
  stores: readonly Store[],
  noteOpeners: Pick<Notes, 'openFile' | 'openNewFile'>,
): Notes => {
  const holder = (id: string): Store | undefined => stores.find((one) => one.has(id))
  return {
    getTabAt: (path) => {
      for (const one of stores) {
        const held = one.getTabAt(path)
        if (held !== null) return held
      }
      return null
    },
    getPath: (id) => holder(id)?.getPath(id) ?? id,
    isAsking: (id) => holder(id)?.isAsking(id) ?? false,
    settle: async (id) => {
      await holder(id)?.settle(id)
    },
    close: (id) => holder(id)?.close(id),
    openFile: noteOpeners.openFile,
    openNewFile: noteOpeners.openNewFile,
  }
}
