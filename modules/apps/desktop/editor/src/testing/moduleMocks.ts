/**
 * The modules the slices reach the application through.
 *
 * A slice hands its answers out through its own `index.ts`. A mock of the
 * module behind that door does not stand in what the door passes on, so the
 * door is mocked too, over the rest of what it gives.
 */
import { vi } from 'vitest'
import { asValue } from '@numen/wire'
import type { Cards } from '@plugins/views.flashcards.studio/frontend/cards'
import type { Documents } from '@plugins/readers.pdf/frontend'
import { requests } from './requests'
import { maker, mockVaultState, pendingStream } from './answers'

const documentsSaid: { documents: Documents } = {
  documents: {
    getDocumentLayout: async () => ({ pages: [{ width: 100, height: 100 }], fingerprint: '' }),
    getPageUrl: () => '',
    getHighlights: async () => [],
  },
}

vi.mock('@plugins/readers.pdf/frontend/api/wire', () => documentsSaid)

vi.mock('@plugins/readers.pdf/frontend', async (original) => ({
  ...(await original<typeof import('@plugins/readers.pdf/frontend')>()),
  ...documentsSaid,
}))

// The runs are answered here; what they are called is the window's own words,
// and a test that reads one reads what the window would have shown.
vi.mock('@/entities/artifact', async (original) => ({
  ...(await original<typeof import('@/entities/artifact')>()),
  running: {
    getArtifactStates: async (path: string) => {
      requests.carried.push(path)
      if (!mockVaultState.carrying) throw new Error('what the file carries cannot be asked')
      return mockVaultState.carries[path] ?? {}
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
    renameField: async (path: string, { from, to }: { from: string; to: string }) => {
      requests.renamedField.push(`${path} ${from} ${to}`)
      return mockVaultState.renaming
    },
    readDeck: async (path: string) =>
      asValue({
        deck: {
          path,
          title: path,
          preamble: '',
          cards: [],
          sections: [],
          tail: '',
          problems: [],
        },
        at: 'a1',
      }),
    writeDeck: async (
      path: string,
      deck: { cards: readonly { values: readonly { text: string }[] }[] },
    ) => {
      requests.cards.push(`deck ${path}`)
      requests.wrote.push(deck.cards.map((card) => card.values[0]?.text ?? '').join(', '))
      return asValue({ at: 'a2' })
    },
    readStencil: async (path: string) =>
      asValue({
        stencil: {
          path,
          title: path,
          fields: [],
          preamble: '',
          faces: [],
          tail: '',
          problems: [],
        },
        at: 'a1',
      }),
    writeStencil: async (path: string) => {
      requests.cards.push(`stencil ${path}`)
      return asValue({ at: 'a2' })
    },
  } satisfies Partial<Cards>,
}

vi.mock('@plugins/views.flashcards.studio/frontend/cards/api/cards', () => cardsSaid)

vi.mock('@plugins/views.flashcards.studio/frontend/cards', async (original) => ({
  ...(await original<typeof import('@plugins/views.flashcards.studio/frontend/cards')>()),
  ...cardsSaid,
}))

vi.mock('@/app/agentPort', () => ({
  agentPort: { ask: pendingStream, finish: () => Promise.resolve() },
}))

const themesSaid = {
  themes: {
    getAppearance: async () => ({
      themes: [
        { name: 'preset:numen', title: 'numen', isBuiltIn: true, isPinned: false },
        { name: 'mine:sea', title: 'sea', isBuiltIn: false, isPinned: false },
      ],
      applied: mockVaultState.applied,
      mode: mockVaultState.mode,
      sizes: mockVaultState.sizes,
      bounds: { interfaceScale: { least: 0.8, most: 2 }, textScale: { least: 0.8, most: 1.75 } },
    }),
    readTheme: async (name: string) => `:root { --numen-surface: ${name} }`,
    writeAppearance: async (
      name: string,
      mode: string,
      sizes: { interfaceScale: number; textScale: number },
    ) => {
      requests.worn.push(`${name} ${mode} ${sizes.interfaceScale}/${sizes.textScale}`)
      return mockVaultState.writeError
    },
    watchThemes: pendingStream,
  },
}

vi.mock('@/entities/settings/api/theme', () => themesSaid)

vi.mock('@/entities/settings', async (original) => ({
  ...(await original<typeof import('@/entities/settings')>()),
  ...themesSaid,
}))
