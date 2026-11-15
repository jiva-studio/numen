import { describe, expect, it, vi } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import type { ApiError, CommandNote, Result, VaultEntry, VaultFiles } from '@numen/plugin-sdk'
import type { CreateNoteRequest, CreateNoteResponse } from './api/templates'
import { activate } from './index'

const PLUGIN_ID = 'md.numen.tools.templates'
const COMMAND = 'note.createFromTemplate'

const note = (path: string): VaultEntry => ({
  path,
  name: path.split('/').pop() ?? path,
  isFolder: false,
  isNote: true,
  type: 'note',
})

const targetOf = (path: string, isFolder: boolean): CommandNote => ({
  path,
  title: path.split('/').pop() ?? path,
  isFolder,
  others: [],
  tab: '',
  type: 'note',
})

const createFiles = (folders: Record<string, readonly VaultEntry[]>): VaultFiles => ({
  list: async (folder) => {
    const found = folders[folder]
    if (found === undefined) throw new Error(`no folder ${folder}`)
    return found
  },
  move: () => Promise.reject(new Error('unused')),
  createFolder: () => Promise.reject(new Error('unused')),
  createNote: () => Promise.reject(new Error('unused')),
})

const invented = {
  Templates: [note('Templates/Meeting.md'), note('Templates/Book review.md')],
  Models: [note('Models/Recipe.md')],
}

const setup = (
  answer: Result<CreateNoteResponse, ApiError>,
  folders: Record<string, readonly VaultEntry[]> = invented,
) => {
  const opened: string[] = []
  const errors: string[] = []
  const createNote = vi.fn(async (_request: CreateNoteRequest) => answer)
  const context = createMockPluginContext({
    pluginId: PLUGIN_ID,
    files: createFiles(folders),
    initialSettings: { folder: 'Templates' },
    workspace: {
      files: { open: (path: string) => opened.push(path) },
      messages: { showError: (text: string) => errors.push(text) },
    } as never,
  })
  activate(context, { createNote })
  return { context, createNote, opened, errors }
}

const created = (path: string): Result<CreateNoteResponse, ApiError> => ({
  ok: true,
  value: { path },
})

describe('the templates frontend', () => {
  it('registers the command the manifest declares', () => {
    const { context } = setup(created('notes/Meeting.md'))

    expect(context.getRegisteredCommands().map((command) => command.id)).toEqual([COMMAND])
  })

  it('lists the templates of the folder in the setting', async () => {
    const { context } = setup(created('notes/Meeting.md'))

    const items = await context.getRegisteredCommands()[0]?.listItems?.(targetOf('a.md', false))

    expect(items?.map((item) => item.id)).toEqual([
      'Templates/Book review.md',
      'Templates/Meeting.md',
    ])
  })

  it('lists other templates after the setting changes', async () => {
    const { context } = setup(created('notes/Meeting.md'))

    context.setSetting('folder', ' /Models/ ')
    const items = await context.getRegisteredCommands()[0]?.listItems?.(targetOf('a.md', false))

    expect(items?.map((item) => item.id)).toEqual(['Models/Recipe.md'])
  })

  it('offers no items while the folder holds no template', async () => {
    const { context } = setup(created('notes/Meeting.md'), { Templates: [] })

    const items = await context.getRegisteredCommands()[0]?.listItems?.(targetOf('a.md', false))

    expect(items).toEqual([])
  })

  it('opens the note the backend made', async () => {
    const { context, opened, errors } = setup(created('notes/Meeting.md'))

    await context.triggerCommand(COMMAND, targetOf('notes', true), 'Templates/Meeting.md')

    expect(opened).toEqual(['notes/Meeting.md'])
    expect(errors).toEqual([])
  })

  it('shows the error and opens nothing when the backend fails', async () => {
    const { context, opened, errors } = setup({
      ok: false,
      error: { code: 'not_found', message: 'template not found: Templates/Meeting.md' },
    })

    await context.triggerCommand(COMMAND, targetOf('notes', true), 'Templates/Meeting.md')

    expect(errors).toEqual(['template not found: Templates/Meeting.md'])
    expect(opened).toEqual([])
  })

  it('does nothing without an item id', async () => {
    const { context, createNote, opened, errors } = setup(created('notes/Meeting.md'))

    await context.triggerCommand(COMMAND, targetOf('notes', true))

    expect(createNote).not.toHaveBeenCalled()
    expect(opened).toEqual([])
    expect(errors).toEqual([])
  })

  it('makes the note in the folder of the target', async () => {
    const { context, createNote } = setup(created('x.md'))

    await context.triggerCommand(COMMAND, targetOf('notes/People', true), 'Templates/Meeting.md')
    await context.triggerCommand(
      COMMAND,
      targetOf('journal/2026-11-14.md', false),
      'Templates/Meeting.md',
    )
    await context.triggerCommand(COMMAND, targetOf('', true), 'Templates/Meeting.md')
    await context.triggerCommand(COMMAND, targetOf('', false), 'Templates/Meeting.md')

    expect(createNote.mock.calls.map(([request]) => request)).toEqual([
      { template: 'Templates/Meeting.md', folder: 'notes/People' },
      { template: 'Templates/Meeting.md', folder: 'journal' },
      { template: 'Templates/Meeting.md', folder: '' },
      { template: 'Templates/Meeting.md', folder: '' },
    ])
  })

  it('throws when the window gives no vault files', () => {
    const context = createMockPluginContext({ pluginId: PLUGIN_ID })

    expect(() =>
      activate(context, { createNote: () => Promise.reject(new Error('unused')) }),
    ).toThrow()
  })
})
