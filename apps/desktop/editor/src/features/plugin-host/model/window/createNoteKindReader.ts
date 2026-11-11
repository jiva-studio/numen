import type { FileKind } from '@/shared/model/fileTypes'

interface FileKindReader {
  readonly fileKinds: (paths: readonly string[]) => Promise<ReadonlyMap<string, FileKind>>
}

/** The type and properties of the note at a path, as the menus of a tab read them. */
export const createNoteKindReader =
  (itemLists: FileKindReader) => async (path: string) => {
    const kind = (await itemLists.fileKinds([path])).get(path)
    return {
      noteType: kind?.type ?? '',
      ...(kind?.properties ? { properties: kind.properties } : {}),
    }
  }
