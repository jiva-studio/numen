/**
 * The modules the slices reach the application through.
 *
 * A slice hands its answers out through its own `index.ts`. A mock of the
 * module behind that door does not stand in what the door passes on, so the
 * door is mocked too, over the rest of what it gives.
 */
import { vi } from 'vitest'
import { asked } from './asked'
import { held, maker, said } from './answers'

const documentsSaid = {
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

const booksSaid = {
  books: {
    getBook: async (path: string) => ({
      title: path,
      span: { begins: 0, ends: 900 },
      documents: [{ path: 'text/one.xhtml', span: { begins: 0, ends: 900 } }],
      parts: [],
      printed: [],
      pages: 1,
      pageBytes: 900,
      at: '20480 1700000000000000000 book.epub',
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
      asked.listened.push(path)
      const { duration, mediaUrl, mediaType } = said.transcribed
      return { duration, mediaUrl, mediaType, url: '' }
    },
    getTaskStates: async (path: string) => said.carries[path] ?? { transcript: 'done' },
    readTranscript: async () => ({
      cues: said.transcribed.cues,
      prose: '',
      editable: said.transcribed.editable,
    }),
    readArticle: async () => ({ cues: [], prose: '', editable: said.transcribed.editable }),
    writeTranscript: async (path: string, cues: readonly { text: string }[]) => {
      asked.transcribed.push(`${path} ${cues.map((one) => one.text).join(' / ')}`)
    },
    findCueTime: async (path: string, span: { from: number }) =>
      said.transcribed.cues.find((one) => one.from >= span.from)?.from ?? null,
  },
}

vi.mock('@/entities/media/wire', () => recordingsSaid)

vi.mock('@/entities/media', async (original) => ({
  ...(await original<typeof import('@/entities/media')>()),
  ...recordingsSaid,
}))

vi.mock('@/shared/artifacts', () => ({
  running: {
    getArtifactStates: async (path: string) => {
      asked.carried.push(path)
      if (!said.carrying) throw new Error('what the file carries cannot be asked')
      return said.carries[path] ?? {}
    },
    createArtifact: async (path: string, of: string) => {
      asked.ran.push(`${of} ${path}`)
      return { able: true, of, made: 'queued', error: '' }
    },
    correctArtifact: async (path: string) => {
      asked.ran.push(`transcript.corrected ${path}`)
      return { able: true, of: 'transcript.corrected' as const, made: 'queued' as const, error: '' }
    },
    fetchArtifact: async (path: string) => {
      asked.ran.push(`transcript ${path}`)
      return { able: true, of: 'transcript' as const, made: 'queued' as const, error: '' }
    },
    deleteTranscript: async (path: string) => {
      asked.ran.push(`drop ${path}`)
      return true
    },
    deleteCopy: async (path: string) => {
      asked.ran.push(`drop ${path}`)
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
      asked.cards.push(`deck ${folder || '/'} ${title}`)
      return maker.createFile(title, folder)
    },
    makeDeck: async (title: string, folder: string) => {
      asked.cards.push(`deck ${folder || '/'} ${title}`)
      return maker.createFile(title, folder)
    },
    createStencil: async (title: string, folder: string, fields: readonly string[]) => {
      asked.cards.push(`stencil ${folder || '/'} ${title} [${fields.join(', ')}]`)
      return maker.createFile(title, folder)
    },
    makeStencil: async (title: string, folder: string, fields: readonly string[]) => {
      asked.cards.push(`stencil ${folder || '/'} ${title} [${fields.join(', ')}]`)
      return maker.createFile(title, folder)
    },
    renameField: async (path: string, from: string, to: string) => {
      asked.renamedField.push(`${path} ${from} ${to}`)
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
      asked.cards.push(`deck ${path}`)
      asked.wrote.push(deck.cards.map((card) => card.values[0]?.text ?? '').join(', '))
      return { error: null, changed: false, at: 'a2', bound: 0 }
    },
    readStencil: async (path: string) => ({
      stencil: { path, title: path, fields: [], faces: [], problems: [] },
      error: null,
      at: 'a1',
    }),
    writeStencil: async (path: string) => {
      asked.cards.push(`stencil ${path}`)
      return { error: null, changed: false, at: 'a2' }
    },
  },
}

vi.mock('@/entities/deck/cards', () => cardsSaid)

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
    appearance: async () => ({
      themes: [
        { name: 'preset:numen', title: 'numen', isBuiltIn: true, isPinned: false },
        { name: 'mine:sea', title: 'sea', isBuiltIn: false, isPinned: false },
      ],
      applied: said.applied,
      mode: said.mode,
      sizes: said.sizes,
      bounds: { interfaceScale: { least: 0.8, most: 2 }, textScale: { least: 0.8, most: 1.75 } },
    }),
    text: async (name: string) => `:root { --numen-surface: ${name} }`,
    chooses: async (
      name: string,
      mode: string,
      sizes: { interfaceScale: number; textScale: number },
    ) => {
      asked.worn.push(`${name} ${mode} ${sizes.interfaceScale}/${sizes.textScale}`)
      return said.writeError
    },
    changed: held,
  },
}

vi.mock('@/entities/settings/theme', () => themesSaid)

vi.mock('@/entities/settings', async (original) => ({
  ...(await original<typeof import('@/entities/settings')>()),
  ...themesSaid,
}))
