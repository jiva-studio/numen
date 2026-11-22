import { describe, expect, it, vi } from 'vitest'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import type { ApiError, CommandNote, Result, VaultEntry, VaultFiles } from '@numen/plugin-sdk'
import type {
  CreateNoteRequest,
  CreateNoteResponse,
  Field,
  ListFieldsResponse,
} from './api/templates'
import type { Answers } from './ui/askFields'
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

const folder = (path: string): VaultEntry => ({
  path,
  name: path.split('/').pop() ?? path,
  isFolder: true,
  isNote: false,
  type: '',
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
    if (found === undefined) {
      throw Object.assign(new Error(`no folder ${folder}`), { code: 'not_found' })
    }
    return found
  },
  move: () => Promise.reject(new Error('unused')),
  createFolder: () => Promise.reject(new Error('unused')),
  createNote: () => Promise.reject(new Error('unused')),
})

const invented = {
  '': [folder('Templates'), folder('Models')],
  Templates: [note('Templates/Meeting.md'), note('Templates/Book review.md')],
  Models: [note('Models/Recipe.md')],
}

const noFields: Result<ListFieldsResponse, ApiError> = { ok: true, value: { fields: [] } }

const askingFor = (fields: readonly Field[]): Result<ListFieldsResponse, ApiError> => ({
  ok: true,
  value: { fields },
})

const CLIENT: Field = { kind: 'text', label: 'Client', options: [], default: 'Acme' }

const setup = (
  answer: Result<CreateNoteResponse, ApiError>,
  folders: Record<string, readonly VaultEntry[]> = invented,
  options: {
    fields?: Result<ListFieldsResponse, ApiError>
    answers?: Answers
    isDenied?: boolean
  } = {},
) => {
  const opened: string[] = []
  const errors: string[] = []
  const createNote = vi.fn(async (_request: CreateNoteRequest) => answer)
  const listFields = vi.fn(async () => options.fields ?? noFields)
  const ask = vi.fn(async (_title: string, _fields: readonly Field[]) => options.answers)
  const context = createMockPluginContext({
    pluginId: PLUGIN_ID,
    initialSettings: { templates: 'Templates/**' },
  })
  const host = createMockHostServices({
    files: options.isDenied
      ? { ...createFiles(folders), list: () => Promise.reject(new Error('access denied')) }
      : createFiles(folders),
    workspace: {
      files: { open: (path: string) => opened.push(path) },
      messages: { showError: (text: string) => errors.push(text) },
    } as never,
  })
  activate(context, host, { createNote, listFields }, ask)
  return { context, createNote, listFields, ask, opened, errors }
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

  it('lists the templates the masks of the setting name', async () => {
    const { context } = setup(created('notes/Meeting.md'))

    const items = await context.getRegisteredCommands()[0]?.listItems?.(targetOf('a.md', false))

    expect(items?.map((item) => item.id)).toEqual([
      'Templates/Book review.md',
      'Templates/Meeting.md',
    ])
  })

  it('lists other templates after the setting changes', async () => {
    const { context } = setup(created('notes/Meeting.md'))

    context.setSetting('templates', ' /Models/** ')
    const items = await context.getRegisteredCommands()[0]?.listItems?.(targetOf('a.md', false))

    expect(items?.map((item) => item.id)).toEqual(['Models/Recipe.md'])
  })

  it('offers no items while no note is a template', async () => {
    const { context } = setup(created('notes/Meeting.md'), {
      '': [folder('Templates')],
      Templates: [],
    })

    const items = await context.getRegisteredCommands()[0]?.listItems?.(targetOf('a.md', false))

    expect(items).toEqual([])
  })

  it('tells the user when the templates cannot be listed', async () => {
    const { context, errors } = setup(created('notes/Meeting.md'), invented, { isDenied: true })

    const items = await context.getRegisteredCommands()[0]?.listItems?.(targetOf('a.md', false))

    expect(items).toEqual([])
    expect(errors).toEqual(['cannot list the folder "": access denied'])
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
      { template: 'Templates/Meeting.md', folder: 'notes/People', values: {} },
      { template: 'Templates/Meeting.md', folder: 'journal', values: {} },
      { template: 'Templates/Meeting.md', folder: '', values: {} },
      { template: 'Templates/Meeting.md', folder: '', values: {} },
    ])
  })

  it('makes the note at once when the template asks nothing', async () => {
    const { context, ask, createNote } = setup(created('notes/Meeting.md'))

    await context.triggerCommand(COMMAND, targetOf('notes', true), 'Templates/Meeting.md')

    expect(ask).not.toHaveBeenCalled()
    expect(createNote).toHaveBeenCalledTimes(1)
  })

  it('asks the fields of the template and makes the note with the answers', async () => {
    const { context, ask, createNote, opened } = setup(created('notes/Visit.md'), invented, {
      fields: askingFor([CLIENT]),
      answers: { Client: 'Globex' },
    })

    await context.triggerCommand(COMMAND, targetOf('notes', true), 'Templates/Visit.md')

    expect(ask).toHaveBeenCalledWith('Visit', [CLIENT])
    expect(createNote).toHaveBeenCalledWith({
      template: 'Templates/Visit.md',
      folder: 'notes',
      values: { Client: 'Globex' },
    })
    expect(opened).toEqual(['notes/Visit.md'])
  })

  it('makes no note when the person cancels the questions', async () => {
    const { context, createNote, opened, errors } = setup(created('notes/Visit.md'), invented, {
      fields: askingFor([CLIENT]),
      answers: undefined,
    })

    await context.triggerCommand(COMMAND, targetOf('notes', true), 'Templates/Visit.md')

    expect(createNote).not.toHaveBeenCalled()
    expect(opened).toEqual([])
    expect(errors).toEqual([])
  })

  it('shows the error and asks nothing when the template cannot be read', async () => {
    const { context, ask, createNote, errors } = setup(created('x.md'), invented, {
      fields: {
        ok: false,
        error: {
          code: 'failed_precondition',
          message: 'template does not produce a valid note: T.md:2',
        },
      },
    })

    await context.triggerCommand(COMMAND, targetOf('notes', true), 'Templates/Visit.md')

    expect(errors).toEqual(['template does not produce a valid note: T.md:2'])
    expect(ask).not.toHaveBeenCalled()
    expect(createNote).not.toHaveBeenCalled()
  })

  it('throws when the window gives no vault files', () => {
    const context = createMockPluginContext({ pluginId: PLUGIN_ID })

    expect(() =>
      activate(context, createMockHostServices(), {
        createNote: () => Promise.reject(new Error('unused')),
        listFields: () => Promise.reject(new Error('unused')),
      }),
    ).toThrow()
  })
})
