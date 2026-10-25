/**
 * The modules the slices reach the application through.
 *
 * A slice hands its answers out through its own `index.ts`. A mock of the
 * module behind that door does not stand in what the door passes on, so the
 * door is mocked too, over the rest of what it gives.
 */
import { vi } from 'vitest'
import { requests } from './requests'
import { mockVaultState, pendingStream } from './answers'

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

vi.mock('@/app/plugins/agentPort', () => ({
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

// The window is shown with no plugin declaring a command.
const declaredCommandsSaid = {
  listDeclaredCommands: async () => [],
  runDeclaredCommand: async () => ({ ok: false as const, error: 'no plugin declares it' }),
}

vi.mock('@/entities/plugin/api/declaredCommands', async (original) => ({
  ...(await original<typeof import('@/entities/plugin/api/declaredCommands')>()),
  ...declaredCommandsSaid,
}))

vi.mock('@/entities/plugin', async (original) => ({
  ...(await original<typeof import('@/entities/plugin')>()),
  ...declaredCommandsSaid,
}))
