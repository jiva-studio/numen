/**
 * File tree and filesystem domain methods for the window core.
 */
import { asFailure, asValue, findErrorCode } from '@numen/wire'
import { files } from '@/shared/clients'
import { mapBookFormat, mapEntry, mapMoveResult, mapNoteType, mapSourceKind } from './words'
import type { FilePort } from '@/app/ports/files'
import type { NotePort } from '@/app/ports/notes'

export type FilesCore = Pick<
  FilePort,
  'list' | 'move' | 'createFolder' | 'fileKinds'
> &
  Pick<NotePort, 'remove'>

export const filesCore: FilesCore = {
  remove: async (path, isPermanent) => {
    const answer = await files.removeFile({ path, destroy: isPermanent ?? false })
    const error = findErrorCode(answer)
    if (error) return asFailure(error)
    return asValue({ trashed: answer.trashed, dangling: answer.dangling })
  },
  list: async (folder) => (await files.listFiles({ path: folder })).entries.map(mapEntry),
  move: async (from, to) => {
    const answer = await files.moveFile({ from, to })
    const error = findErrorCode(answer)
    if (error) return asFailure(error)
    return asValue(answer.moved ? mapMoveResult(answer.moved) : null)
  },
  createFolder: async (path) => findErrorCode(await files.createFolder({ path })),
  fileKinds: async (paths) => {
    const answer = await files.listFileKinds({ paths: [...paths] })
    return new Map(
      answer.kinds.map((one) => {
        const format = mapBookFormat(one.format)
        return [
          one.path,
          {
            kind: mapSourceKind(one.kind),
            type: mapNoteType(one.type),
            ...(format ? { format } : {}),
          },
        ]
      }),
    )
  },
}
