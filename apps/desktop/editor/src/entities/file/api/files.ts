/**
 * The calls of the file service: what a folder holds, which file is which, and what moves or
 * leaves the vault.
 */
import { findErrorCode, asFailure, asValue } from '@numen/core-client'
import type { WindowClients } from '@/shared/api/clients'
import type { ErrorCode } from '@/shared/api/errors'
import type { Result } from '@/shared/lib/result'
import { mapMoveResult, mapNoteType } from '@/shared/api/wire'
import type { Entry, FileKind, Movement, RemoveResult } from '@/shared/model/fileTypes'
import { mapEntry } from './fileWords'

/** The calls of the file service over `files`. */
export const createFileApi = (files: WindowClients['files']) => ({
  /** A file or folder moved out of the vault into the trash, from which it can be restored. */
  removeFile: async (path: string): Promise<RemoveResult> => {
    const answer = await files.removeFile({ path })
    const error = findErrorCode(answer)
    if (error) return asFailure(error)
    return asValue({ trashed: answer.trashed, dangling: answer.dangling })
  },

  /** Deletes everything the vault's trash holds for good; how many paths could not be deleted. */
  emptyTrash: async (): Promise<number> => (await files.emptyTrash({})).undeleted.length,

  /**
   * What one folder of the vault holds, in the order to draw it: folders first and then files, each
   * group by name with case ignored. The root is the empty path, and hidden files are in none of
   * the answers.
   */
  listFiles: async (folder: string): Promise<readonly Entry[]> =>
    (await files.listFiles({ path: folder })).entries.map(mapEntry),

  /**
   * A file or a folder filed somewhere else. The last segment of `to` is what it is called from now
   * on, so a name changed within one folder is a move.
   */
  moveFile: async (from: string, to: string): Promise<Movement> => {
    const answer = await files.moveFile({ from, to })
    const error = findErrorCode(answer)
    if (error) return asFailure(error)
    return asValue(answer.moved ? mapMoveResult(answer.moved) : null)
  },

  /** An empty folder. The folders above it are made with it. */
  createFolder: async (path: string): Promise<Result<null, ErrorCode>> => {
    const error = findErrorCode(await files.createFolder({ path }))
    return error ? asFailure(error) : asValue(null)
  },

  /**
   * Kind is read from the vault itself, so a path not yet scanned returns what exists there; a path
   * with nothing at it is absent.
   */
  getFileKinds: async (paths: readonly string[]): Promise<ReadonlyMap<string, FileKind>> => {
    const answer = await files.listFileKinds({ paths: [...paths] })
    return new Map(
      answer.kinds.map((one) => [
        one.path,
        {
          isNote: one.isNote,
          type: mapNoteType(one.type),
          ...(one.tabKind ? { tabKind: one.tabKind } : {}),
          ...(one.properties && Object.keys(one.properties).length > 0
            ? { properties: one.properties }
            : {}),
        },
      ]),
    )
  },
})

export type FileApi = ReturnType<typeof createFileApi>
