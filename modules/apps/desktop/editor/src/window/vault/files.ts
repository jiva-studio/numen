/**
 * File tree and filesystem domain methods for the window core.
 */
import { files } from './clients'
import { bookFormat, mapEntry, mapMoveResult, noteType, sourceKind } from './words'
import { errorIn } from '../../shared/answers'
import type { Core } from '../../shared/core'

export type FilesCore = Pick<
  Core,
  'remove' | 'list' | 'move' | 'makeFolder' | 'makeURL' | 'fileKinds'
>

export const filesCore: FilesCore = {
  remove: async (path, destroy) => {
    const answer = await files.removeFile({ path, destroy: destroy ?? false })
    const error = errorIn(answer)
    return {
      trashed: answer.trashed,
      dangling: answer.dangling,
      error,
    }
  },
  list: async (folder) => (await files.listFiles({ path: folder })).entries.map(mapEntry),
  move: async (from, to) => {
    const answer = await files.moveFile({ from, to })
    const error = errorIn(answer)
    return {
      moved: answer.moved ? mapMoveResult(answer.moved) : null,
      error,
    }
  },
  makeFolder: async (path) => errorIn(await files.createFolder({ path })),
  makeURL: async (url, folder) => {
    const answer = await files.createURL({ url, path: folder })
    const error = errorIn(answer)
    return { path: answer.path, error }
  },
  fileKinds: async (paths) => {
    const answer = await files.listFileKinds({ paths: [...paths] })
    return new Map(
      answer.kinds.map((one) => {
        const format = bookFormat(one.format)
        return [
          one.path,
          { kind: sourceKind(one.kind), type: noteType(one.type), ...(format ? { format } : {}) },
        ]
      }),
    )
  },
}
