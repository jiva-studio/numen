/**
 * What the core answers about notes: reading, writing and finding them. The calls go through the
 * real note client and its error mapping; only the transport under them is the test's.
 */
import { NAME_SOURCE } from '@numen/core-client'
import type { CoreAnswers } from '@numen/core-client/testing'
import { requests } from '../requests'
import { mockVaultState, createPendingStream } from '../answers'

export const noteAnswers = {
  notes: {
    getOpeningNote: () => ({
      ...(mockVaultState.opening ? { note: { path: mockVaultState.opening, title: '' } } : {}),
    }),
    listRelatedNotes: (request) => ({
      graphs: request.paths.map((path) => ({
        focus: { path, title: path.replace(/\.md$/, '') },
        focusType: 'note',
        relatedNotes: [],
        connections: [],
      })),
    }),
    readNote: (request) => ({
      body: 'what is written',
      fileStat: { path: request.path, size: 1n, mtime: 1n },
    }),
    writeNote: (request) => ({ body: '', fileStat: { path: request.path, size: 2n, mtime: 2n } }),
    createNote: (request) => {
      requests.made.push(request.title)
      return { path: `${request.title}.md` }
    },
    renameNote: (request) => {
      requests.renamed.push(`${request.path} ${request.title}`)
      return { path: request.path, title: request.title, nameSource: NAME_SOURCE.FRONTMATTER }
    },
    watchEdits: createPendingStream,
    listHeadings: () => ({ headings: [] }),
  },
} satisfies CoreAnswers
