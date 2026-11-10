/**
 * What the core answers about the files and folders of the vault, through the real file client.
 */
import type { CoreAnswers } from '@numen/core-client/testing'
import { requests } from '../requests'
import { folders, getFileKind } from '../answers'

export const fileAnswers = {
  files: {
    removeFile: (request) => {
      requests.removed.push(request.path)
      return { trashed: `.trash/${request.path}`, dangling: [] }
    },
    listFiles: (request) => ({ entries: [...(folders[request.path] ?? [])] }),
    moveFile: (request) => {
      requests.moved.push(`${request.from} ${request.to}`)
      return {}
    },
    createFolder: (request) => {
      requests.folders.push(request.path)
      return {}
    },
    listFileKinds: (request) => ({
      kinds: request.paths.map((path) => ({ path, ...getFileKind(path) })),
    }),
  },
} satisfies CoreAnswers
