/** The commands the plugins declare, as the window runs them. */
import { describe, expect, it, vi } from 'vitest'
import type { CommandNote, PluginHost } from '@numen/plugin-loader'
import type { CommandTarget } from '@/entities/command'
import { createPluginCommandPort } from './pluginCommandPort'

const TARGET = {
  tab: 'note:1',
  kind: 'note',
  path: 'Tides.md',
  title: 'Tides',
  file: 'Tides.md',
  vault: { id: 'sea', name: 'Sea' },
  isReady: true,
} as CommandTarget

const NOTE: CommandNote = {
  path: 'Tides.md',
  title: 'Tides',
  type: 'note',
  isFolder: false,
  others: [],
  tab: 'note:1',
}

const setup = () => {
  const commands = {
    hasCommand: (id: string) => id === 'org.example.tide:chart',
    handlers: { isSharedCommand: (id: string) => id === 'app:shared.open' },
    executeCommand: vi.fn(() => Promise.resolve(true)),
    handleShortcut: vi.fn(() => true),
  }
  const getCommandNote = vi.fn((_target: { readonly path: string }) => Promise.resolve(NOTE))
  const port = createPluginCommandPort(
    { commands } as unknown as Pick<PluginHost, 'commands'>,
    getCommandNote,
  )
  return { port, commands, getCommandNote }
}

describe('the commands of the plugins', () => {
  it('are the ones a plugin registered and the shared ones the plugins declare handlers for', () => {
    const { port } = setup()

    expect(port.isPluginCommand('org.example.tide:chart')).toBe(true)
    expect(port.isPluginCommand('app:shared.open')).toBe(true)
    expect(port.isPluginCommand('app:note.save')).toBe(false)
  })

  it('run over the note of the target, with how they are run', async () => {
    const { port, commands } = setup()

    await port.run('org.example.tide:chart', TARGET, { values: { kind: 'plex' }, isAwaited: true })

    expect(commands.executeCommand).toHaveBeenCalledExactlyOnceWith(
      'org.example.tide:chart',
      NOTE,
      { values: { kind: 'plex' }, isAwaited: true },
    )
  })

  it('take a key with the target in front, and read the note only when it is needed', async () => {
    const { port, commands, getCommandNote } = setup()
    const runApp = vi.fn()
    const event = new KeyboardEvent('keydown', { code: 'KeyL' })

    expect(port.handleShortcut(event, () => TARGET, runApp)).toBe(true)

    expect(getCommandNote).not.toHaveBeenCalled()
    const [pressed, getNote, context] = commands.handleShortcut.mock.calls[0] as unknown as [
      KeyboardEvent,
      () => Promise<CommandNote>,
      { target: { path: string }; runApp: unknown },
    ]
    expect(pressed).toBe(event)
    expect(context).toStrictEqual({ target: expect.objectContaining({ path: 'Tides.md' }), runApp })
    expect(await getNote()).toBe(NOTE)
  })
})
