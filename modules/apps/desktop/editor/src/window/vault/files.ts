/**
 * File tree and filesystem domain methods for the window core.
 */
import { files } from './clients'
import { bookFormat, filed, listed, noteType, sourceKind } from './words'
import { refusalIn } from '../../shared/answers'
import type { Core } from '../../shared/core'

export type FilesCore = Pick<
  Core,
  'remove' | 'list' | 'move' | 'makeFolder' | 'makeURL' | 'fileKinds'
>

export const filesCore: FilesCore = {
  remove: async (path, destroy) => {
    const answer = await files.removeFile({ path, destroy: destroy ?? false })
    return {
      trashed: answer.trashed,
      dangling: answer.dangling,
      refusal: refusalIn(answer),
    }
  },
  list: async (folder) => (await files.listFiles({ path: folder })).entries.map(listed),
  move: async (from, to) => {
    const answer = await files.moveFile({ from, to })
    return {
      moved: answer.moved ? filed(answer.moved) : null,
      refusal: refusalIn(answer),
    }
  },
  makeFolder: async (path) => refusalIn(await files.createFolder({ path })),
  makeURL: async (url, folder) => {
    const answer = await files.createURL({ url, path: folder })
    return { path: answer.path, refusal: refusalIn(answer) }
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
