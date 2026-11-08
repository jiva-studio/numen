/** Delete text, the command of the window the core serves over the text versions of a file. Every path is invented. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { commands, createInvocation, type CommandTarget } from '@/features/command-palette'
import { createVersionCommands } from './versionCommands'

const DELETE_TEXT = 'app:version.delete'

const createTarget = (path: string, isFolder = false): CommandTarget => ({
  tab: '',
  kind: null,
  path,
  title: path,
  file: path,
  isFolder,
  vault: { id: 'v', name: 'V' },
  isReady: true,
})

const setup = (
  outcome:
    { ok: true; value: { message: string; openedPath: string } } | { ok: false; error: string },
) => {
  const listItems = vi.fn(async () => [{ id: 'v1', title: 'transcript (live)' }])
  const run = vi.fn(async () => outcome)
  const versionCommands = createVersionCommands({ listItems, run })
  return { versionCommands, listItems, run }
}

afterEach(() => commands.clear())

describe('Delete text', () => {
  it('lists the versions the core lists over the file, with no plugin', async () => {
    const { versionCommands, listItems } = setup({
      ok: true,
      value: { message: '', openedPath: '' },
    })
    const [command] = versionCommands.commands

    expect(command?.text).toBe('Delete text')
    expect(await command?.listItems?.(createTarget('talks/a.mp3'))).toStrictEqual([
      { id: 'v1', title: 'transcript (live)' },
    ])
    expect(listItems).toHaveBeenCalledWith({ pluginId: '', id: DELETE_TEXT }, 'talks/a.mp3')
  })

  it('is available over a file and over no folder', () => {
    const [command] = setup({ ok: true, value: { message: '', openedPath: '' } }).versionCommands
      .commands

    expect(command?.isAvailable(createTarget('talks/a.mp3'), {} as never)).toBe(true)
    expect(command?.isAvailable(createTarget('talks', true), {} as never)).toBe(false)
    expect(command?.isAvailable(createTarget(''), {} as never)).toBe(false)
  })

  it('has the core delete the version picked, and writes the error of a delete that failed', async () => {
    const { versionCommands, run } = setup({ ok: false, error: 'no such version' })
    versionCommands.register()
    const writeMessage = vi.fn()

    await commands.get(DELETE_TEXT)!(
      createInvocation(DELETE_TEXT, createTarget('talks/a.mp3'), 'v1'),
      { writeMessage } as never,
      {} as never,
    )

    expect(run).toHaveBeenCalledWith(DELETE_TEXT, 'talks/a.mp3', 'v1')
    expect(writeMessage).toHaveBeenCalledWith('no such version', 'error')
  })
})
