/**
 * The plugin entity's part of the mocks: the agent a plugin asks and the commands the plugins
 * declare. A slice hands its answers out through its own `index.ts`, and a mock of the module
 * behind that door does not stand in what the door passes on, so the door is mocked too.
 */
import { vi } from 'vitest'
import { declared, pendingStream } from '../answers'
import type * as DeclaredCommandsModule from '@/entities/plugin/api/declaredCommands'
import type * as PluginModule from '@/entities/plugin'

vi.mock('@/app/plugins/agentPort', () => ({
  agentPort: { ask: pendingStream, finish: () => Promise.resolve() },
}))

// The window is shown with the plugins `declared` names; none declares a command to begin with.
const declaredCommandsResponse = {
  listDeclaredCommands: () =>
    Promise.resolve({ commands: declared.commands, slots: declared.slots }),
  runDeclaredCommand: () => Promise.resolve({ ok: false as const, error: 'no plugin declares it' }),
}

vi.mock('@/entities/plugin/api/declaredCommands', async (original) => ({
  ...(await original<typeof DeclaredCommandsModule>()),
  ...declaredCommandsResponse,
}))

vi.mock('@/entities/plugin', async (original) => ({
  ...(await original<typeof PluginModule>()),
  ...declaredCommandsResponse,
}))
