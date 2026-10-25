/**
 * The commands the plugins declare as the window reads them, and a run as the
 * window hands it over and reads what came of it.
 */
import { describe, expect, it, vi } from 'vitest'
import { CommandArgumentType, CommandErrorCode } from '@numen/protocol'

vi.mock('@/shared/clients', () => ({ pluginCommands: request }))

const request = {
  listCommands: vi.fn(),
  runPluginCommand: vi.fn(),
}

const { canRunOn, listDeclaredCommands, runDeclaredCommand } = await import('./declaredCommands')

const convert = {
  pluginId: 'md.numen.pages',
  id: 'convert',
  title: 'Convert',
  description: '',
  fileExtensions: ['pdf'],
  arguments: [],
}

describe('the commands the plugins declare', () => {
  it('are read with the types of their arguments', async () => {
    request.listCommands.mockResolvedValue({
      commands: [
        {
          pluginId: 'md.numen.cards',
          id: 'create-deck',
          title: 'Create a deck',
          description: 'Makes one.',
          fileExtensions: [],
          arguments: [
            { name: 'title', title: 'Title', type: CommandArgumentType.STRING, isRequired: true },
            { name: 'size', title: 'Size', type: CommandArgumentType.NUMBER, isRequired: false },
            { name: 'open', title: 'Open', type: CommandArgumentType.BOOLEAN, isRequired: false },
          ],
        },
      ],
    })

    const [one] = await listDeclaredCommands()

    expect(one?.pluginId).toBe('md.numen.cards')
    expect(one?.arguments.map((argument) => argument.type)).toStrictEqual([
      'string',
      'number',
      'boolean',
    ])
    expect(one?.arguments[0]?.isRequired).toBe(true)
  })

  it('run over a file of one of their extensions, by the name of the file alone', () => {
    expect(canRunOn(convert, 'books/Bees.PDF')).toBe(true)
    expect(canRunOn(convert, 'books/Bees.epub')).toBe(false)
    expect(canRunOn(convert, 'books.pdf/Bees')).toBe(false)
    expect(canRunOn(convert, '')).toBe(false)
  })
})

describe('a run', () => {
  it('is handed the file and the values, and answers what the plugin said', async () => {
    request.runPluginCommand.mockResolvedValue({
      error: CommandErrorCode.UNSPECIFIED,
      message: 'Converted 3 pages',
    })

    const outcome = await runDeclaredCommand(convert, { path: 'a.pdf', values: { pages: '3' } })

    expect(outcome).toStrictEqual({ ok: true, value: 'Converted 3 pages' })
    expect(request.runPluginCommand).toHaveBeenLastCalledWith({
      pluginId: 'md.numen.pages',
      command: 'convert',
      path: 'a.pdf',
      arguments: { pages: '3' },
    })
  })

  it('of a command over no file is handed none', async () => {
    request.runPluginCommand.mockResolvedValue({ error: CommandErrorCode.UNSPECIFIED, message: '' })

    await runDeclaredCommand({ ...convert, fileExtensions: [] }, { path: 'a.md', values: {} })

    expect(request.runPluginCommand).toHaveBeenLastCalledWith(expect.objectContaining({ path: '' }))
  })

  it('that did not succeed answers why', async () => {
    request.runPluginCommand.mockResolvedValue({
      error: CommandErrorCode.FAILED,
      message: 'the disk is full',
    })

    const outcome = await runDeclaredCommand(convert, { path: 'a.pdf', values: {} })

    expect(outcome).toStrictEqual({ ok: false, error: 'the disk is full' })
  })
})
