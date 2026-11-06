/**
 * A press on a button of the title bar over the runner of the palette: what the plugin host and
 * the palette receive, and the message a failure leaves.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { formatErrorMessage } from '@numen/core-client'
import type { SlotItem } from '@numen/plugin-host'
import type { CommandTarget } from '@/features/command-palette'
import { useCommands, type CommandsDepsOptions } from '../commands/useCommands'
import { useTitleBarCommands } from './useTitleBarCommands'

const h = vi.hoisted(() => ({
  startCommand: vi.fn(),
  getObjection: vi.fn(() => 'Nothing to run here'),
  runInvocation: vi.fn(() => Promise.resolve()),
  isCommandsOpen: { value: false },
  setCommandsOpen: vi.fn(),
}))

vi.mock('@/features/command-palette', () => ({
  ANSWER_WORDS: { name: 'answer words' },
  isChord: vi.fn(),
  findChordCommand: vi.fn(),
  findShortcutKeys: () => undefined,
  runInvocation: h.runInvocation,
  registerCommand: vi.fn(),
  useSearch: () => ({ setOpen: vi.fn() }),
  useCommandPalette: () => ({
    startCommand: h.startCommand,
    getObjection: h.getObjection,
    setOpen: h.setCommandsOpen,
    open: h.isCommandsOpen,
  }),
}))
vi.mock('../commands/commandDeps', () => ({ createCommandDeps: () => ({ name: 'command deps' }) }))

const TIDE = 'org.example.tide'
const OPEN = `${TIDE}:chart.open`

const front = {
  tab: 'tab-1',
  kind: 'note',
  path: 'Kelp.md',
  title: 'Kelp',
  file: 'Kelp.md',
  vault: { id: 'v', name: 'Brine notes' },
  isReady: true,
} as unknown as CommandTarget

const entry = (command: string): SlotItem => ({
  kind: 'command',
  key: `${TIDE}/command/${command}`,
  pluginId: TIDE,
  command,
  title: command,
  icon: { render: () => null },
  shortcut: '',
})

function setup(executeCommand: () => Promise<boolean> = async () => true) {
  const execute = vi.fn(executeCommand)
  const writeMessage = vi.fn()
  const pluginHost = {
    getPaletteCommands: () => [],
    hasCommand: (id: string) => id === OPEN,
    executeCommand: execute,
    handleShortcut: () => false,
  }
  const used = useCommands({
    core: { fileKinds: vi.fn(async () => new Map()) },
    words: {},
    windowTabs: { onKeyPress: () => false },
    getTarget: () => front,
    knows: {},
    kept: {},
    runs: {},
    coverage: () => ({}),
    rest: {},
    writeMessage,
    pluginHost,
  } as unknown as CommandsDepsOptions)
  const bar = useTitleBarCommands({
    slots: { getEntries: () => [entry(OPEN), entry('app:note.create')] },
    isVaultOpen: () => true,
    runCommand: used.runCommand,
    getTarget: () => front,
    agent: 'Linux',
  })
  return { bar, execute, writeMessage }
}

beforeEach(() => {
  vi.clearAllMocks()
  h.isCommandsOpen.value = false
})

describe('a press on a button of a plugin command', () => {
  it('runs the command once in the plugin host over the note in front', async () => {
    const one = setup()

    one.bar.runCommand(OPEN)
    await vi.waitFor(() => expect(one.execute).toHaveBeenCalled())

    expect(one.execute).toHaveBeenCalledExactlyOnceWith(
      OPEN,
      expect.objectContaining({ path: 'Kelp.md' }),
    )
    expect(h.startCommand).not.toHaveBeenCalled()
  })

  it('returns at once while the command still waits for its plugin, and runs it once', async () => {
    let release: (value: boolean) => void = () => undefined
    const one = setup(() => new Promise<boolean>((resolve) => (release = resolve)))

    one.bar.runCommand(OPEN)
    await vi.waitFor(() => expect(one.execute).toHaveBeenCalledTimes(1))
    release(true)
    await Promise.resolve()

    expect(one.execute).toHaveBeenCalledTimes(1)
    expect(one.writeMessage).not.toHaveBeenCalled()
  })

  it('writes the message of the error a failing command raises, as the palette does', async () => {
    const failure = new Error('The tide store is locked')
    const one = setup(async () => {
      throw failure
    })

    one.bar.runCommand(OPEN)

    await vi.waitFor(() =>
      expect(one.writeMessage).toHaveBeenCalledExactlyOnceWith(
        formatErrorMessage(failure),
        'error',
      ),
    )
  })
})

describe('a press on a button of a command of the editor', () => {
  it('starts the command in the palette with the target in front and runs the invocation', () => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    const one = setup()

    one.bar.runCommand('app:note.create')

    expect(h.startCommand).toHaveBeenCalledExactlyOnceWith('app:note.create', front)
    expect(h.runInvocation).toHaveBeenCalledTimes(1)
    expect(one.execute).not.toHaveBeenCalled()
  })

  it('leaves the palette open at the step of a command that needs input', () => {
    h.startCommand.mockReturnValue(null)
    h.isCommandsOpen.value = true
    const one = setup()

    one.bar.runCommand('app:note.create')

    expect(h.runInvocation).not.toHaveBeenCalled()
    expect(one.writeMessage).not.toHaveBeenCalled()
  })
})
