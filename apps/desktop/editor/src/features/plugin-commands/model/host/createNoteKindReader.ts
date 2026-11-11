import type { CommandItemLists } from '../commandItemLists'

/** The type and properties of the note at a path, as the menus of a tab read them. */
export const createNoteKindReader =
  (itemLists: Pick<CommandItemLists, 'fileKinds'>) => async (path: string) => {
    const kind = (await itemLists.fileKinds([path])).get(path)
    return {
      noteType: kind?.type ?? '',
      ...(kind?.properties ? { properties: kind.properties } : {}),
    }
  }
