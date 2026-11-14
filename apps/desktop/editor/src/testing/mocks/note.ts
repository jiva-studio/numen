/**
 * What the core answers about notes: reading, writing and finding them. The calls go through the
 * real note client and its error mapping; only the transport under them is the test's.
 */
import { NAME_SOURCE } from '@numen/core-client'
import type { CoreAnswers } from '@numen/core-client/testing'
import { requests } from '../requests'
import { mockVaultState, pendingStream } from '../answers'

export const noteAnswers = {
  notes: {
    getOpeningNote: () => ({
      ...(mockVaultState.opening ? { note: { path: mockVaultState.opening, title: '' } } : {}),
    }),
    listRelatedNotes: (request) => ({
      focus: { path: request.path, title: request.path.replace(/\.md$/, '') },
      focusType: 'note',
      related: [],
      connections: [],
    }),
    readNote: (request) => ({
      body: 'what is written',
      at: { path: request.path, size: 1n, mtime: 1n },
    }),
    writeNote: (request) => ({ body: '', at: { path: request.path, size: 2n, mtime: 2n } }),
    createNote: (request) => {
      requests.made.push(request.title)
      return { path: `${request.title}.md` }
    },
    renameNote: (request) => {
      requests.renamed.push(`${request.path} ${request.title}`)
      return { path: request.path, title: request.title, nameSource: NAME_SOURCE.FRONTMATTER }
    },
    watchEdits: pendingStream,
    listHeadings: () => ({ headings: [] }),
  },
} satisfies CoreAnswers
