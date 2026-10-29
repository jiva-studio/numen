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

vi.mock('@/app/plugins/agentPort', () => ({
  agentPort: { ask: pendingStream, finish: () => Promise.resolve() },
}))

const themesResponse = {
  themes: {
    getAppearance: async () => ({
      themes: [
        { name: 'preset:numen', title: 'numen', isBuiltIn: true, isPinned: false },
        { name: 'mine:sea', title: 'sea', isBuiltIn: false, isPinned: false },
      ],
      applied: mockVaultState.applied,
      mode: mockVaultState.mode,
      sizes: mockVaultState.sizes,
      bounds: { interfaceScale: { min: 0.8, max: 2 }, textScale: { min: 0.8, max: 1.75 } },
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

vi.mock('@/entities/settings/api/theme', () => themesResponse)

vi.mock('@/entities/settings', async (original) => ({
  ...(await original<typeof import('@/entities/settings')>()),
  ...themesResponse,
}))

// The window is shown with no plugin declaring a command.
const declaredCommandsResponse = {
  listDeclaredCommands: async () => [],
  runDeclaredCommand: async () => ({ ok: false as const, error: 'no plugin declares it' }),
}

vi.mock('@/entities/plugin/api/declaredCommands', async (original) => ({
  ...(await original<typeof import('@/entities/plugin/api/declaredCommands')>()),
  ...declaredCommandsResponse,
}))

vi.mock('@/entities/plugin', async (original) => ({
  ...(await original<typeof import('@/entities/plugin')>()),
  ...declaredCommandsResponse,
}))
