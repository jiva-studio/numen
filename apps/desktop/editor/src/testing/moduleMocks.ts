/**
 * The modules the slices reach the application through.
 *
 * A slice hands its answers out through its own `index.ts`. A mock of the
 * module behind that door does not stand in what the door passes on, so the
 * door is mocked too, over the rest of what it gives.
 */
import { vi } from 'vitest'
import type { BackgroundTask, BackgroundTaskHost } from '@numen/plugin-host'
import { declared, mockVaultState, pendingStream } from './answers'

vi.mock('@/app/plugins/agentPort', () => ({
  agentPort: { ask: pendingStream, finish: () => Promise.resolve() },
}))

const themesResponse = {
  themes: {
    getAppearance: async () => ({
      themes: [
        { name: 'preset:numen', title: 'numen', isBuiltIn: true, pluginId: '', isPinned: false },
        { name: 'mine:sea', title: 'sea', isBuiltIn: false, pluginId: '', isPinned: false },
      ],
      applied: mockVaultState.applied,
      mode: mockVaultState.mode,
      sizes: mockVaultState.sizes,
      bounds: { interfaceScale: { min: 0.8, max: 2 }, textScale: { min: 0.8, max: 1.75 } },
    }),
    readTheme: async (name: string) => `:root { --numen-surface: ${name} }`,
    watchThemes: pendingStream,
  },
}

vi.mock('@/entities/settings/api/theme', () => themesResponse)

vi.mock('@/entities/settings', async (original) => ({
  ...(await original<typeof import('@/entities/settings')>()),
  ...themesResponse,
}))

// The window is shown with the plugins `declared` names; none declares a command to begin with.
const declaredCommandsResponse = {
  listDeclaredCommands: async () => ({ commands: declared.commands, menus: declared.menus }),
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

/** The background tasks the window last made, as its plugins read them. */
const windowTasks = vi.hoisted(() => ({ current: undefined as BackgroundTaskHost | undefined }))

vi.mock('@/app/shell/backgroundTasks', async (original) => {
  const module = await original<typeof import('@/app/shell/backgroundTasks')>()
  return {
    ...module,
    createBackgroundTaskHost: (deps: Parameters<typeof module.createBackgroundTaskHost>[0]) =>
      (windowTasks.current = module.createBackgroundTaskHost(deps)),
  }
})

/** The tasks the window lists for its plugins now; empty before a window is mounted. */
export const readWindowTasks = (): readonly BackgroundTask[] => {
  let tasks: readonly BackgroundTask[] = []
  windowTasks.current?.watch((list) => (tasks = list)).dispose()
  return tasks
}
