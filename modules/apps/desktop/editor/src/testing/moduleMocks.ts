/**
 * The modules the slices reach the application through.
 *
 * A slice hands its answers out through its own `index.ts`. A mock of the
 * module behind that door does not stand in what the door passes on, so the
 * door is mocked too, over the rest of what it gives.
 */
import { vi } from 'vitest'
import type { Books } from '@/pages/book-reader/types'
import type { Documents } from '@/pages/document-viewer/types'
import { requests } from './requests'
import { held, maker, said } from './answers'

const documentsSaid: { documents: Documents } = {
  documents: {
    getDocumentLayout: async () => ({ pages: [{ width: 100, height: 100 }], fingerprint: '' }),
    getPageUrl: () => '',
    getHighlights: async () => [],
  },
}

vi.mock('@/pages/document-viewer/api/wire', () => documentsSaid)

vi.mock('@/pages/document-viewer', async (original) => ({
  ...(await original<typeof import('@/pages/document-viewer')>()),
  ...documentsSaid,
}))

// The port is named in the type, so a member the slice renames stops the
// typecheck here instead of answering undefined to a test that still passes.
const booksSaid: { books: Books } = {
  books: {
    getBook: async (path: string) => ({
      title: path,
      span: { from: 0, to: 900 },
      documents: [{ path: 'text/one.xhtml', span: { from: 0, to: 900 } }],
      parts: [],
      printedPages: [],
      pages: 1,
      pageBytes: 900,
      fingerprint: '20480 1700000000000000000 book.epub',
    }),
    readMarkup: async () => '<p data-offset="0">the book</p>',
    getEntryUrl: () => '',
  },
}

vi.mock('@/pages/book-reader/api/wire', () => booksSaid)

vi.mock('@/pages/book-reader', async (original) => ({
  ...(await original<typeof import('@/pages/book-reader')>()),
  ...booksSaid,
}))

const recordingsSaid = {
  recordings: {
    getSummary: async (path: string) => {
      requests.listened.push(path)
      const { duration, mediaUrl, mediaType } = said.transcribed
      return { duration, mediaUrl, mediaType, url: '' }
    },
    getTaskStates: async (path: string) => said.carries[path] ?? { transcript: 'done' },
    readTranscript: async () => ({
      cues: said.transcribed.cues,
      prose: '',
      isEditable: said.transcribed.isEditable,
    }),
    readArticle: async () => ({ cues: [], prose: '', isEditable: said.transcribed.isEditable }),
    writeTranscript: async (path: string, cues: readonly { text: string }[]) => {
      requests.transcribed.push(`${path} ${cues.map((one) => one.text).join(' / ')}`)
    },
    findCueTime: async (path: string, span: { from: number }) =>
      said.transcribed.cues.find((one) => one.from >= span.from)?.from ?? null,
  },
}

vi.mock('@/entities/media/api/wire', () => recordingsSaid)

vi.mock('@/entities/media', async (original) => ({
  ...(await original<typeof import('@/entities/media')>()),
  ...recordingsSaid,
}))

vi.mock('@/shared/artifacts', () => ({
  running: {
    getArtifactStates: async (path: string) => {
      requests.carried.push(path)
      if (!said.carrying) throw new Error('what the file carries cannot be asked')
      return said.carries[path] ?? {}
    },
    createArtifact: async (path: string, of: string) => {
      requests.ran.push(`${of} ${path}`)
      return { able: true, of, made: 'queued', error: '' }
    },
    correctArtifact: async (path: string) => {
      requests.ran.push(`transcript.corrected ${path}`)
      return { able: true, of: 'transcript.corrected' as const, made: 'queued' as const, error: '' }
    },
    fetchArtifact: async (path: string) => {
      requests.ran.push(`transcript ${path}`)
      return { able: true, of: 'transcript' as const, made: 'queued' as const, error: '' }
    },
    deleteTranscript: async (path: string) => {
      requests.ran.push(`drop ${path}`)
      return true
    },
    deleteCopy: async (path: string) => {
      requests.ran.push(`drop ${path}`)
      return true
    },
  },
}))

const cardsSaid = {
  cards: {
    // A card is named by the first field of the stencil it is cut by, so the
    // window is told of one.
    stencils: async () => ({
      stencils: [{ path: 'Animal.md', title: 'Animal', fields: ['Name'] }],
      held: 1,
    }),
    createDeck: async (title: string, folder: string) => {
      requests.cards.push(`deck ${folder || '/'} ${title}`)
      return maker.createFile(title, folder)
    },
    createStencil: async (title: string, folder: string, fields: readonly string[]) => {
      requests.cards.push(`stencil ${folder || '/'} ${title} [${fields.join(', ')}]`)
      return maker.createFile(title, folder)
    },
    renameField: async (path: string, from: string, to: string) => {
      requests.renamedField.push(`${path} ${from} ${to}`)
      return said.renaming
    },
    readDeck: async (path: string) => ({
      deck: {
        path,
        title: path,
        preamble: '',
        cards: [],
        sections: [],
        tail: '',
        problems: [],
      },
      error: null,
      at: 'a1',
      bound: 0,
    }),
    writeDeck: async (
      path: string,
      deck: { cards: readonly { values: readonly { text: string }[] }[] },
    ) => {
      requests.cards.push(`deck ${path}`)
      requests.wrote.push(deck.cards.map((card) => card.values[0]?.text ?? '').join(', '))
      return { error: null, changed: false, at: 'a2', bound: 0 }
    },
    readStencil: async (path: string) => ({
      stencil: { path, title: path, fields: [], faces: [], problems: [] },
      error: null,
      at: 'a1',
    }),
    writeStencil: async (path: string) => {
      requests.cards.push(`stencil ${path}`)
      return { error: null, changed: false, at: 'a2' }
    },
  },
}

vi.mock('@/entities/deck/api/cards', () => cardsSaid)

vi.mock('@/entities/deck', async (original) => ({
  ...(await original<typeof import('@/entities/deck')>()),
  ...cardsSaid,
}))

const agentSaid = { core: { ask: held, finish: async () => {} } }

vi.mock('@/pages/agent-chat/api/core', () => agentSaid)

vi.mock('@/pages/agent-chat', async (original) => ({
  ...(await original<typeof import('@/pages/agent-chat')>()),
  ...agentSaid,
}))

const themesSaid = {
  themes: {
    getAppearance: async () => ({
      themes: [
        { name: 'preset:numen', title: 'numen', isBuiltIn: true, isPinned: false },
        { name: 'mine:sea', title: 'sea', isBuiltIn: false, isPinned: false },
      ],
      applied: said.applied,
      mode: said.mode,
      sizes: said.sizes,
      bounds: { interfaceScale: { least: 0.8, most: 2 }, textScale: { least: 0.8, most: 1.75 } },
    }),
    readTheme: async (name: string) => `:root { --numen-surface: ${name} }`,
    writeAppearance: async (
      name: string,
      mode: string,
      sizes: { interfaceScale: number; textScale: number },
    ) => {
      requests.worn.push(`${name} ${mode} ${sizes.interfaceScale}/${sizes.textScale}`)
      return said.writeError
    },
    watchThemes: held,
  },
}

vi.mock('@/entities/settings/api/theme', () => themesSaid)

vi.mock('@/entities/settings', async (original) => ({
  ...(await original<typeof import('@/entities/settings')>()),
  ...themesSaid,
}))
