/**
 * The calls of the note service, over the real client and a router in place of the core.
 *
 * The one thing this module decides on its own: how a file the window references is serialized
 * and parsed. Everything else passes a message through unchanged, and the schema defines its shape.
 */
import { describe, expect, it, vi } from 'vitest'
import {
  NOTE_ERROR_CODE,
  NOTE_MARKER,
  NOTE_RELATION,
  NOTE_ROLE,
  NAME_SOURCE,
} from '@numen/core-client'
import { createHandledClients } from '@/testing/coreAnswers'
import { createLinkApi, createNoteApi } from './notes'

const request = {
  readNote: vi.fn(),
  writeNote: vi.fn(),
  createNote: vi.fn(),
  listRelatedNotes: vi.fn(),
  getOpeningNote: vi.fn(),
  renameNote: vi.fn(),
  listHeadings: vi.fn(),
  resolveAddresses: vi.fn(),
  watchEdits: vi.fn(),
}

const clients = createHandledClients(request)
const api = createNoteApi(clients.notes)
const createLinks = () => createLinkApi(clients.notes)

describe('the file a save presents', () => {
  it('arrives back as the file it was, through a path that holds spaces', async () => {
    const noteFile = {
      path: 'mahabharata/The aggressor.md',
      size: 4096n,
      mtime: 1_700_000_000_123n,
    }
    request.readNote.mockResolvedValue({ body: 'prose', fileStat: noteFile })
    request.writeNote.mockResolvedValue({})

    const read = await api.readNote(noteFile.path)
    if (!read.ok) throw new Error('the read failed')
    await api.writeNote(noteFile.path, 'prose and more', {
      prose: read.value.body,
      stat: read.value.at ?? '',
    })

    expect(request.writeNote.mock.calls[0]?.[0]).toMatchObject({
      path: noteFile.path,
      body: 'prose and more',
      baseline: { prose: 'prose', fileStat: noteFile },
    })
  })

  it('is left out of a save that presents nothing', async () => {
    request.writeNote.mockResolvedValue({})

    await api.writeNote('Heat.md', 'mine', null)

    expect(request.writeNote.mock.calls.at(-1)?.[0]).toMatchObject({
      path: 'Heat.md',
      body: 'mine',
    })
    expect(request.writeNote.mock.calls.at(-1)?.[0].baseline).toBeUndefined()
  })

  it('is a change under the window where the file moved past the save', async () => {
    request.writeNote.mockResolvedValue({ error: NOTE_ERROR_CODE.CONFLICT })

    expect(
      await api.writeNote('Heat.md', 'mine', { prose: 'old', stat: '1:0: 1 Heat.md' }),
    ).toEqual({
      ok: false,
      error: 'changed',
    })
  })
})

describe('a note read', () => {
  it('is requested for by its path', async () => {
    request.readNote.mockResolvedValue({ body: 'prose' })

    expect(await api.readNote('Notes/Heat.md')).toEqual({ ok: true, value: { body: 'prose' } })
    expect(request.readNote.mock.calls.at(-1)?.[0]).toMatchObject({ path: 'Notes/Heat.md' })
  })

  it('fails in the words of the error', async () => {
    request.readNote.mockResolvedValue({ error: NOTE_ERROR_CODE.NOT_TEXT })

    expect(await api.readNote('photo.png')).toEqual({ ok: false, error: 'notText' })
  })
})

describe('a note created', () => {
  it('is created at the path of its folder, and rejected with error of the error', async () => {
    request.createNote.mockResolvedValue({ path: 'Notes/Heat.md' })
    const created = await api.createNote({ title: 'Heat', folder: 'Notes' })
    expect(created).toEqual({ ok: true, value: { path: 'Notes/Heat.md' } })
    expect(request.createNote.mock.calls.at(-1)?.[0].path).toBe('Notes')

    request.createNote.mockResolvedValue({ error: NOTE_ERROR_CODE.ALREADY_EXISTS })
    expect(await api.createNote({ title: 'Heat', folder: 'Notes' })).toEqual({
      ok: false,
      error: 'alreadyExists',
    })

    request.createNote.mockResolvedValue({ isNameExhausted: true })
    expect(await api.createNote({ title: '', folder: '' })).toEqual({
      ok: false,
      error: 'alreadyExists',
    })
  })
})

describe('a note renamed', () => {
  it('is renamed by the path it has, and a stale answer is a change under the window', async () => {
    request.renameNote.mockResolvedValue({
      path: 'B.md',
      title: 'B',
      nameSource: NAME_SOURCE.FRONTMATTER,
    })
    expect(await api.renameNote('A.md', 'B')).toEqual({
      ok: true,
      value: { path: 'B.md', title: 'B', hasFrontmatter: true, moved: null },
    })
    expect(request.renameNote.mock.calls.at(-1)?.[0]).toMatchObject({ path: 'A.md', title: 'B' })

    request.renameNote.mockResolvedValue({ error: NOTE_ERROR_CODE.CONFLICT })
    expect(await api.renameNote('A.md', 'B')).toEqual({ ok: false, error: 'changed' })

    request.renameNote.mockResolvedValue({ error: NOTE_ERROR_CODE.ALREADY_EXISTS })
    expect(await api.renameNote('A.md', 'B')).toEqual({ ok: false, error: 'alreadyExists' })
  })

  it('carries the file that moved with it, and the links that were repaired', async () => {
    request.renameNote.mockResolvedValue({
      path: 'b.md',
      title: 'B',
      nameSource: NAME_SOURCE.FILENAME,
      moveResult: { from: 'a.md', to: 'b.md', repairedNotes: ['c.md'] },
    })

    expect(await api.renameNote('a.md', 'B')).toEqual({
      ok: true,
      value: {
        path: 'b.md',
        title: 'B',
        hasFrontmatter: false,
        moved: { from: 'a.md', to: 'b.md', repaired: ['c.md'] },
      },
    })
  })
})

describe('the headings of notes', () => {
  it('are requested of every path given, and keyed by path', async () => {
    request.listHeadings.mockResolvedValue({
      headings: [{ path: 'a.md', headings: [{ text: 'H1', level: 1, line: 1 }] }],
    })

    const headings = await api.listHeadings(['a.md', 'b.md'])

    expect(request.listHeadings.mock.calls.at(-1)?.[0]).toMatchObject({ paths: ['a.md', 'b.md'] })
    expect(headings.get('a.md')).toEqual([{ text: 'H1', level: 1, line: 1 }])
    expect(headings.has('b.md')).toBe(false)
  })
})

describe('the wikilinks of a text', () => {
  it('resolve relative to the source note, and omit crossed links', async () => {
    const links = createLinks()
    request.resolveAddresses.mockResolvedValue({
      links: [
        { raw: '[[B]]', address: 'name://B', text: 'B' },
        { raw: '[[C|see C]]', address: 'name://C', text: 'see C' },
        { raw: '[[D]]', address: 'name://D', text: 'D' },
      ],
      resolvedAddresses: [
        { address: 'name://B', path: 'b.md', isCrossed: false },
        { address: 'name://C', path: 'c.md', isCrossed: true },
      ],
    })

    const resolved = await links.resolveLinks('a.md', '[[B]] [[C|see C]] [[D]]')

    expect(request.resolveAddresses.mock.calls.at(-1)?.[0]).toMatchObject({
      from: 'a.md',
      text: '[[B]] [[C|see C]] [[D]]',
      vaultId: '',
    })
    expect(resolved).toEqual([
      { raw: '[[B]]', address: 'name://B', text: 'B', path: 'b.md' },
      { raw: '[[C|see C]]', address: 'name://C', text: 'see C', path: '' },
      { raw: '[[D]]', address: 'name://D', text: 'D', path: '' },
    ])
  })

  it('resolves a name:// address to the path of the note, or empty when unresolved', async () => {
    const links = createLinks()
    request.resolveAddresses.mockResolvedValueOnce({
      links: [{ raw: '[[B]]', address: 'name://B', text: 'B' }],
      resolvedAddresses: [{ address: 'name://B', path: 'b.md', isCrossed: false }],
    })
    expect(await links.resolveNameLink('a.md', 'name://B')).toBe('b.md')

    request.resolveAddresses.mockResolvedValueOnce({ links: [], resolvedAddresses: [] })
    expect(await links.resolveNameLink('a.md', 'name://Missing')).toBe('')
  })
})

describe('the note the vault opens first', () => {
  it('is requested of no particular caller, and is none where the answer names none', async () => {
    request.getOpeningNote.mockResolvedValue({})

    expect(await api.getInitialOpenPath()).toBeNull()
    expect(request.getOpeningNote.mock.calls.at(-1)?.[0]).toMatchObject({})
  })

  it('is the path the answer names', async () => {
    request.getOpeningNote.mockResolvedValue({ note: { path: 'opening.md', title: 'Opening' } })

    expect(await api.getInitialOpenPath()).toEqual({ path: 'opening.md' })
  })
})

describe('the edits made to notes', () => {
  it('are read, and a missing span is read as the start', async () => {
    const apiInstance = createNoteApi(clients.notes)
    const edits = [
      { change: '1', path: 'a.md', range: { start: 2, end: 5 }, text: 'abc', isFinal: false },
      { change: '2', path: 'a.md', text: 'x', isFinal: true },
    ]
    request.watchEdits.mockImplementation(async function* () {
      yield* edits
    })

    const observed = []
    for await (const edit of apiInstance.watchEdits(new AbortController().signal)) {
      observed.push(edit)
    }

    expect(observed).toEqual([
      { change: '1', path: 'a.md', span: { start: 2, end: 5 }, text: 'abc', isComplete: false },
      { change: '2', path: 'a.md', span: { start: 0, end: 0 }, text: 'x', isComplete: true },
    ])
  })

  it('stops watching edits when the signal aborts', async () => {
    const apiInstance = createNoteApi(clients.notes)
    request.watchEdits.mockImplementation(async function* (_req, context) {
      yield { change: '1', path: 'a.md', text: 'a', isFinal: false }
      if (!context.signal.aborted) {
        yield { change: '2', path: 'a.md', text: 'b', isFinal: true }
      }
    })
    const controller = new AbortController()
    const observed: object[] = []
    await (async () => {
      for await (const edit of apiInstance.watchEdits(controller.signal)) {
        observed.push(edit)
        controller.abort()
      }
    })().catch(() => undefined)

    expect(observed).toHaveLength(1)
  })
})

const readGraph = async (paths: readonly string[], depth?: 1 | 2) => {
  const graphs = await api.listRelatedNotes(paths, depth)
  expect(graphs).toHaveLength(paths.length)
  return graphs[0]!
}

describe('a noteGraph', () => {
  it('asks for all paths in one request and maps each graph', async () => {
    request.listRelatedNotes.mockClear()
    request.listRelatedNotes.mockResolvedValue({
      graphs: [
        { focus: { path: 'b.md', title: 'B' }, relatedNotes: [] },
        { focus: { path: 'a.md', title: 'A' }, relatedNotes: [] },
      ],
    })

    const graphs = await api.listRelatedNotes(['b.md', 'a.md'])

    expect(request.listRelatedNotes).toHaveBeenCalledTimes(1)
    expect(request.listRelatedNotes.mock.calls[0]?.[0]).toMatchObject({ paths: ['b.md', 'a.md'] })
    expect(graphs.map((graph) => graph.focus.path)).toEqual(['b.md', 'a.md'])
  })

  it('maps a graph whose focus is unset to an empty path and leaves the others whole', async () => {
    request.listRelatedNotes.mockResolvedValue({
      graphs: [
        { focus: { path: 'a.md', title: 'A' }, relatedNotes: [] },
        { relatedNotes: [] },
        { focus: { path: 'c.md', title: 'C' }, relatedNotes: [] },
      ],
    })

    const graphs = await api.listRelatedNotes(['a.md', 'gone.md', 'c.md'])

    expect(graphs.map((graph) => graph.focus.path)).toEqual(['a.md', '', 'c.md'])
  })

  it('has its noteGraph requested of its path', async () => {
    request.listRelatedNotes.mockResolvedValue({
      graphs: [
        {
          focus: { path: 'a.md', title: 'A' },
          relatedNotes: [],
          connections: [],
        },
      ],
    })

    const noteGraph = await readGraph(['a.md'])

    expect(noteGraph.focus.path).toBe('a.md')
    expect(request.listRelatedNotes.mock.calls.at(-1)?.[0]).toMatchObject({ paths: ['a.md'] })
  })

  it('asks for depth 1 unless a depth is given, and for depth 2 when it is given', async () => {
    request.listRelatedNotes.mockResolvedValue({
      graphs: [{ focus: { path: 'a.md', title: 'A' } }],
    })

    await api.listRelatedNotes(['a.md'])
    expect(request.listRelatedNotes.mock.calls.at(-1)?.[0]).toMatchObject({
      paths: ['a.md'],
      depth: 1,
    })
    await api.listRelatedNotes(['a.md'], 2)
    expect(request.listRelatedNotes.mock.calls.at(-1)?.[0]).toMatchObject({
      paths: ['a.md'],
      depth: 2,
    })
  })

  it('names the note an item of depth 2 belongs to, and omits it for a note related to the focus', async () => {
    request.listRelatedNotes.mockResolvedValue({
      graphs: [
        {
          focus: { path: 'Rota.md', title: 'The rota' },
          relatedNotes: [
            {
              note: { path: 'Plots.md', title: 'Plots' },
              relation: NOTE_RELATION.CHILD,
              relatedTo: '',
            },
            {
              note: { path: 'Beans.md', title: 'Beans' },
              relation: NOTE_RELATION.CHILD,
              relatedTo: 'Plots.md',
            },
          ],
        },
      ],
    })

    const noteGraph = await readGraph(['Rota.md'], 2)

    expect(noteGraph.related.map((one) => one.relatedTo)).toEqual([undefined, 'Plots.md'])
    expect('relatedTo' in noteGraph.related[0]!).toBe(false)
  })

  it('is drawn around the note in focus, with each note in the relation it holds', async () => {
    request.listRelatedNotes.mockResolvedValue({
      graphs: [
        {
          focus: { path: 'Rota.md', title: 'The rota' },
          focusType: 'sample',
          relatedNotes: [
            {
              note: { path: 'Allotments.md', title: 'Allotments' },
              relation: NOTE_RELATION.PARENT,
            },
          ],
          connections: [
            {
              first: 'Allotments.md',
              second: 'Rota.md',
              role: NOTE_ROLE.CHILD,
              firstSide: {
                role: NOTE_ROLE.CHILD,
                label: 'contains rota',
                marker: NOTE_MARKER.ARROW,
              },
              secondSide: { role: NOTE_ROLE.PARENT, label: 'part of' },
            },
          ],
        },
      ],
    })

    const noteGraph = await readGraph(['Rota.md'])

    expect(noteGraph.focus).toEqual({ path: 'Rota.md', title: 'The rota' })
    expect(noteGraph.focusType).toBe('sample')
    expect(noteGraph.related).toEqual([
      { path: 'Allotments.md', title: 'Allotments', type: 'note', relation: 'parent' },
    ])
    expect(noteGraph.connections).toEqual([
      {
        first: 'Allotments.md',
        second: 'Rota.md',
        role: 'child',
        type: '',
        firstSide: { label: 'contains rota', marker: 'arrow', reason: '' },
        secondSide: { label: 'part of', marker: null, reason: '' },
      },
    ])
  })

  it('leaves out a note it has no relation for, and one the answer names no note at', async () => {
    request.listRelatedNotes.mockResolvedValue({
      graphs: [
        {
          focus: { path: 'Rota.md', title: 'The rota' },
          relatedNotes: [
            { note: { path: 'Odd.md', title: 'Odd' }, relation: NOTE_RELATION.UNSPECIFIED },
            { relation: NOTE_RELATION.CHILD },
          ],
        },
      ],
    })

    expect((await api.listRelatedNotes(['Rota.md']))[0]!.related).toEqual([])
  })

  it('relies on nothing where the vault no longer holds the note', async () => {
    request.listRelatedNotes.mockResolvedValue({ graphs: [{ relatedNotes: [] }] })

    expect((await api.listRelatedNotes(['Gone.md']))[0]!.focus.path).toBe('')
  })
})
