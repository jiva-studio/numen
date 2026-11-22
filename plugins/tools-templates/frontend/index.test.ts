import { describe, expect, it, vi } from 'vitest'
import {
  createMockHostServices,
  createMockPickerHost,
  createMockPluginContext,
} from '@numen/plugin-sdk/testing'
import type { ApiError, CommandNote, Result, VaultEntry, VaultFiles } from '@numen/plugin-sdk'
import type {
  CreateNoteRequest,
  CreateNoteResponse,
  Field,
  ListFieldsResponse,
} from './api/templates'
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

const noFields: Result<ListFieldsResponse, ApiError> = {
  ok: true,
  value: { fields: [], etag: 'etag-none', name: 'Meeting' },
}

const askingFor = (fields: readonly Field[]): Result<ListFieldsResponse, ApiError> => ({
  ok: true,
  value: { fields, etag: 'etag-asked', name: 'Visit' },
})

const CLIENT: Field = { kind: 'text', label: 'Client', options: [], default: 'Acme' }

const setup = (
  answer: Result<CreateNoteResponse, ApiError>,
  folders: Record<string, readonly VaultEntry[]> = invented,
  options: {
    fields?: Result<ListFieldsResponse, ApiError>
    isDenied?: boolean
    hasNoPicker?: boolean
  } = {},
) => {
  const opened: string[] = []
  const errors: string[] = []
  const notices: { text: string; severity: string | undefined }[] = []
  const createNote = vi.fn(async (_request: CreateNoteRequest) => answer)
  const listFields = vi.fn(async () => options.fields ?? noFields)
  const picker = createMockPickerHost()
  const context = createMockPluginContext({
    pluginId: PLUGIN_ID,
    initialSettings: { templates: 'Templates/**' },
  })
  const host = createMockHostServices({
    ...(options.hasNoPicker ? {} : { picker }),
    files: options.isDenied
      ? { ...createFiles(folders), list: () => Promise.reject(new Error('access denied')) }
      : createFiles(folders),
    workspace: {
      files: { open: (path: string) => opened.push(path) },
      messages: {
        showError: (text: string) => errors.push(text),
        getWriter: () => (text: string, severity?: string) => notices.push({ text, severity }),
      },
    } as never,
  })
  activate(context, host, { createNote, listFields })
  return { context, createNote, listFields, picker, opened, errors, notices }
}

const created = (path: string): Result<CreateNoteResponse, ApiError> => ({
  ok: true,
  value: { path, warning: '' },
})

type Picker = ReturnType<typeof createMockPickerHost>
type Trigger = ReturnType<typeof createMockPluginContext>['triggerCommand']

const acceptName = async (picker: Picker, running: Promise<unknown>): Promise<void> => {
  await vi.waitFor(() => expect(picker.getQuestion()).toBeDefined())
  const question = picker.getQuestion()
  picker.answer(question && 'value' in question ? (question.value ?? '') : '')
  await running
}

const makeNote = async (
  context: { triggerCommand: Trigger },
  picker: Picker,
  target: CommandNote,
  template: string,
): Promise<void> => acceptName(picker, context.triggerCommand(COMMAND, target, template))

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

  it('reports a folder that cannot be listed', async () => {
    const { context, errors } = setup(created('notes/Meeting.md'), invented, { isDenied: true })

    const items = await context.getRegisteredCommands()[0]?.listItems?.(targetOf('a.md', false))

    expect(items).toEqual([])
    expect(errors).toEqual(['cannot list the folder "": access denied'])
  })

  it('opens the note the backend made', async () => {
    const { context, picker, opened, errors } = setup(created('notes/Meeting.md'))

    await makeNote(context, picker, targetOf('notes', true), 'Templates/Meeting.md')

    expect(opened).toEqual(['notes/Meeting.md'])
    expect(errors).toEqual([])
  })

  it('shows the error and opens nothing when the backend fails', async () => {
    const { context, picker, opened, errors } = setup({
      ok: false,
      error: { code: 'not_found', message: 'template not found: Templates/Meeting.md' },
    })

    await makeNote(context, picker, targetOf('notes', true), 'Templates/Meeting.md')

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
    const { context, picker, createNote } = setup(created('x.md'))

    await makeNote(context, picker, targetOf('notes/People', true), 'Templates/Meeting.md')
    await makeNote(
      context,
      picker,
      targetOf('journal/2026-11-14.md', false),
      'Templates/Meeting.md',
    )
    await makeNote(context, picker, targetOf('', true), 'Templates/Meeting.md')
    await makeNote(context, picker, targetOf('', false), 'Templates/Meeting.md')

    expect(createNote.mock.calls.map(([request]) => request.folder)).toEqual([
      'notes/People',
      'journal',
      '',
      '',
    ])
  })

  it('asks the name of the note first, with the suggested name typed', async () => {
    const { context, picker, listFields } = setup(created('notes/Meeting.md'))

    const running = context.triggerCommand(COMMAND, targetOf('notes', true), 'Templates/Meeting.md')
    await vi.waitFor(() => expect(picker.getQuestion()).toBeDefined())

    expect(listFields).toHaveBeenCalledWith({ template: 'Templates/Meeting.md', folder: 'notes' })
    expect(picker.getQuestion()).toEqual({ title: 'Meeting · Name', value: 'Meeting' })
    picker.dismiss()
    await running
  })

  it('makes the note with the name typed', async () => {
    const { context, picker, createNote } = setup(created('notes/Kickoff.md'))

    const running = context.triggerCommand(COMMAND, targetOf('notes', true), 'Templates/Meeting.md')
    await vi.waitFor(() => expect(picker.getQuestion()).toBeDefined())
    picker.answer('Kickoff')
    await running

    expect(createNote).toHaveBeenCalledExactlyOnceWith({
      template: 'Templates/Meeting.md',
      folder: 'notes',
      name: 'Kickoff',
      values: {},
      etag: 'etag-none',
    })
  })

  it('makes no note when the person cancels the name', async () => {
    const { context, picker, createNote, opened, errors } = setup(created('notes/Meeting.md'))

    const running = context.triggerCommand(COMMAND, targetOf('notes', true), 'Templates/Meeting.md')
    await vi.waitFor(() => expect(picker.getQuestion()).toBeDefined())
    picker.dismiss()
    await running

    expect(createNote).not.toHaveBeenCalled()
    expect(opened).toEqual([])
    expect(errors).toEqual([])
  })

  it('asks the name again when the backend says it is taken, and keeps the answers', async () => {
    const { context, picker, createNote, opened, errors } = setup(
      created('notes/Retro.md'),
      invented,
      {
        fields: askingFor([CLIENT]),
      },
    )
    createNote.mockResolvedValueOnce({
      ok: false,
      error: { code: 'already_exists', message: 'a note of that name exists: notes/Visit.md' },
    })

    const running = context.triggerCommand(COMMAND, targetOf('notes', true), 'Templates/Visit.md')
    await vi.waitFor(() => expect(picker.getQuestion()).toBeDefined())
    picker.answer('Visit')
    await vi.waitFor(() => expect(picker.getQuestion()).toMatchObject({ title: 'Visit · Client' }))
    picker.answer('Globex')
    await vi.waitFor(() =>
      expect(picker.getQuestion()).toEqual({
        title: 'Visit · Name · a note of that name exists: notes/Visit.md',
        value: 'Visit',
      }),
    )
    picker.answer('Retro')
    await running

    expect(createNote.mock.calls.map(([request]) => [request.name, request.values])).toEqual([
      ['Visit', { Client: 'Globex' }],
      ['Retro', { Client: 'Globex' }],
    ])
    expect(opened).toEqual(['notes/Retro.md'])
    expect(errors).toEqual([])
  })

  it('asks the name again when the backend finds it no name for a note', async () => {
    const { context, picker, createNote } = setup(created('notes/Ok.md'))
    createNote.mockResolvedValueOnce({
      ok: false,
      error: { code: 'invalid_argument', message: 'a note cannot be given this title' },
    })

    const running = context.triggerCommand(COMMAND, targetOf('notes', true), 'Templates/Meeting.md')
    await vi.waitFor(() => expect(picker.getQuestion()).toBeDefined())
    picker.answer('a/b')
    await vi.waitFor(() =>
      expect(picker.getQuestion()).toEqual({
        title: 'Meeting · Name · a note cannot be given this title',
        value: 'a/b',
      }),
    )
    picker.dismiss()
    await running

    expect(createNote).toHaveBeenCalledTimes(1)
  })

  it('ends with the backend message when an answer is refused instead of asking the name again', async () => {
    const { context, picker, createNote, errors } = setup(created('notes/Ok.md'))
    createNote.mockResolvedValueOnce({
      ok: false,
      error: { code: 'failed_precondition', message: '"nap" is not one of the options of "Kind": call, visit' },
    })

    const running = context.triggerCommand(COMMAND, targetOf('notes', true), 'Templates/Meeting.md')
    await vi.waitFor(() => expect(picker.getQuestion()).toBeDefined())
    picker.answer('Meeting')
    await running

    expect(picker.getQuestion()).toBeUndefined()
    expect(createNote).toHaveBeenCalledTimes(1)
    expect(errors).toEqual(['"nap" is not one of the options of "Kind": call, visit'])
  })

  it('asks the fields of the template after the name and makes the note with the answers', async () => {
    const { context, picker, createNote, opened } = setup(created('notes/Visit.md'), invented, {
      fields: askingFor([CLIENT]),
    })

    const running = context.triggerCommand(COMMAND, targetOf('notes', true), 'Templates/Visit.md')
    await vi.waitFor(() =>
      expect(picker.getQuestion()).toEqual({ title: 'Visit · Name', value: 'Visit' }),
    )
    picker.answer('Visit')
    await vi.waitFor(() =>
      expect(picker.getQuestion()).toEqual({ title: 'Visit · Client', value: 'Acme' }),
    )
    picker.answer('Globex')
    await running

    expect(createNote).toHaveBeenCalledWith({
      template: 'Templates/Visit.md',
      folder: 'notes',
      name: 'Visit',
      values: { Client: 'Globex' },
      etag: 'etag-asked',
    })
    expect(opened).toEqual(['notes/Visit.md'])
  })

  it('shows an error when the window cannot ask the name', async () => {
    const { context, createNote, errors } = setup(created('x.md'), invented, { hasNoPicker: true })

    await context.triggerCommand(COMMAND, targetOf('notes', true), 'Templates/Meeting.md')

    expect(errors).toEqual(['the window cannot ask the name of the note'])
    expect(createNote).not.toHaveBeenCalled()
  })

  it('opens the note and warns when it could not be indexed', async () => {
    const { context, picker, opened, errors, notices } = setup({
      ok: true,
      value: { path: 'notes/Meeting.md', warning: 'note is made but not indexed' },
    })

    await makeNote(context, picker, targetOf('notes', true), 'Templates/Meeting.md')

    expect(opened).toEqual(['notes/Meeting.md'])
    expect(notices).toEqual([{ text: 'note is made but not indexed', severity: 'warning' }])
    expect(errors).toEqual([])
  })

  it('says nothing when the note has no warning', async () => {
    const { context, picker, notices } = setup(created('notes/Meeting.md'))

    await makeNote(context, picker, targetOf('notes', true), 'Templates/Meeting.md')

    expect(notices).toEqual([])
  })

  it('makes no note when the person cancels a field', async () => {
    const { context, picker, createNote, opened, errors } = setup(
      created('notes/Visit.md'),
      invented,
      {
        fields: askingFor([CLIENT]),
      },
    )

    const running = context.triggerCommand(COMMAND, targetOf('notes', true), 'Templates/Visit.md')
    await vi.waitFor(() => expect(picker.getQuestion()).toBeDefined())
    picker.answer('Visit')
    await vi.waitFor(() => expect(picker.getQuestion()).toMatchObject({ title: 'Visit · Client' }))
    picker.dismiss()
    await running

    expect(createNote).not.toHaveBeenCalled()
    expect(opened).toEqual([])
    expect(errors).toEqual([])
  })

  it('shows the error and asks nothing when the template cannot be read', async () => {
    const { context, picker, createNote, errors } = setup(created('x.md'), invented, {
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
    expect(picker.getQuestion()).toBeUndefined()
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
