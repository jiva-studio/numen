/**
 * The calls of the note service, over the real client and a router in place of the core.
 *
 * The one thing this module decides on its own: how a file the window references is serialized
 * and parsed. Everything else passes a message through unchanged, and the schema defines its shape.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  NOTE_ERROR_CODE,
  NOTE_MARKER,
  NOTE_RELATION,
  NOTE_ROLE,
  NAME_SOURCE,
} from '@numen/core-client'
import { installCoreHandlers } from '@numen/core-client/testing'
import {
  createNote,
  getInitialOpenPath,
  getNeighborhood,
  listHeadings,
  readNote,
  renameNote,
  resolveLinks,
  watchEdits,
  writeNote,
} from './notes'

const request = {
  readNote: vi.fn(),
  writeNote: vi.fn(),
  createNote: vi.fn(),
  getNeighborhood: vi.fn(),
  getOpeningNote: vi.fn(),
  renameNote: vi.fn(),
  listHeadings: vi.fn(),
  resolveAddresses: vi.fn(),
  watchEdits: vi.fn(),
}

let restore: () => void

beforeEach(() => {
  restore = installCoreHandlers(request)
})

afterEach(() => {
  restore()
})

describe('the file a save presents', () => {
  it('arrives back as the file it was, through a path that holds spaces', async () => {
    const noteFile = {
      path: 'mahabharata/The aggressor.md',
      size: 4096n,
      mtime: 1_700_000_000_123n,
    }
    request.readNote.mockResolvedValue({ body: 'prose', at: noteFile })
    request.writeNote.mockResolvedValue({})

    const read = await readNote(noteFile.path)
    if (!read.ok) throw new Error('the read failed')
    await writeNote(noteFile.path, 'prose and more', {
      prose: read.value.body,
      stat: read.value.at ?? '',
    })

    expect(request.writeNote.mock.calls[0]?.[0]).toMatchObject({
      path: noteFile.path,
      body: 'prose and more',
      baseline: { prose: 'prose', at: noteFile },
    })
  })

  it('is left out of a save that presents nothing', async () => {
    request.writeNote.mockResolvedValue({})

    await writeNote('Heat.md', 'mine', null)

    expect(request.writeNote.mock.calls.at(-1)?.[0]).toMatchObject({
      path: 'Heat.md',
      body: 'mine',
    })
    expect(request.writeNote.mock.calls.at(-1)?.[0].baseline).toBeUndefined()
  })

  it('is a change under the window where the file moved past the save', async () => {
    request.writeNote.mockResolvedValue({ error: NOTE_ERROR_CODE.CONFLICT })

    expect(await writeNote('Heat.md', 'mine', { prose: 'old', stat: '1 1 Heat.md' })).toEqual({
      ok: false,
      error: 'changed',
    })
  })
})

describe('a note read', () => {
  it('is requested for by its path', async () => {
    request.readNote.mockResolvedValue({ body: 'prose' })

    expect(await readNote('Notes/Heat.md')).toEqual({ ok: true, value: { body: 'prose' } })
    expect(request.readNote.mock.calls.at(-1)?.[0]).toMatchObject({ path: 'Notes/Heat.md' })
  })

  it('fails in the words of the error', async () => {
    request.readNote.mockResolvedValue({ error: NOTE_ERROR_CODE.NOT_TEXT })

    expect(await readNote('photo.png')).toEqual({ ok: false, error: 'notText' })
  })
})

describe('a note created', () => {
  it('is created at the path of its folder, and rejected with error of the error', async () => {
    request.createNote.mockResolvedValue({ path: 'Notes/Heat.md' })
    const created = await createNote({ title: 'Heat', folder: 'Notes' })
    expect(created).toEqual({ ok: true, value: { path: 'Notes/Heat.md' } })
    expect(request.createNote.mock.calls.at(-1)?.[0].path).toBe('Notes')

    request.createNote.mockResolvedValue({ error: NOTE_ERROR_CODE.ALREADY_EXISTS })
    expect(await createNote({ title: 'Heat', folder: 'Notes' })).toEqual({
      ok: false,
      error: 'alreadyExists',
    })

    request.createNote.mockResolvedValue({ isNameExhausted: true })
    expect(await createNote({ title: '', folder: '' })).toEqual({
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
    expect(await renameNote('A.md', 'B')).toEqual({
      ok: true,
      value: { path: 'B.md', title: 'B', hasFrontmatter: true, moved: null },
    })
    expect(request.renameNote.mock.calls.at(-1)?.[0]).toMatchObject({ path: 'A.md', title: 'B' })

    request.renameNote.mockResolvedValue({ error: NOTE_ERROR_CODE.CONFLICT })
    expect(await renameNote('A.md', 'B')).toEqual({ ok: false, error: 'changed' })

    request.renameNote.mockResolvedValue({ error: NOTE_ERROR_CODE.ALREADY_EXISTS })
    expect(await renameNote('A.md', 'B')).toEqual({ ok: false, error: 'alreadyExists' })
  })

  it('carries the file that moved with it, and the links that were repaired', async () => {
    request.renameNote.mockResolvedValue({
      path: 'b.md',
      title: 'B',
      nameSource: NAME_SOURCE.FILENAME,
      moved: { from: 'a.md', to: 'b.md', repaired: ['c.md'] },
    })

    expect(await renameNote('a.md', 'B')).toEqual({
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

    const headings = await listHeadings(['a.md', 'b.md'])

    expect(request.listHeadings.mock.calls.at(-1)?.[0]).toMatchObject({ paths: ['a.md', 'b.md'] })
    expect(headings.get('a.md')).toEqual([{ text: 'H1', level: 1, line: 1 }])
    expect(headings.has('b.md')).toBe(false)
  })
})

describe('the wikilinks of a text', () => {
  it('resolve relative to the source note, and omit crossed links', async () => {
    request.resolveAddresses.mockResolvedValue({
      found: [
        { raw: '[[B]]', address: 'name://B', text: 'B' },
        { raw: '[[C|see C]]', address: 'name://C', text: 'see C' },
        { raw: '[[D]]', address: 'name://D', text: 'D' },
      ],
      resolved: [
        { written: 'name://B', path: 'b.md', isCrossed: false },
        { written: 'name://C', path: 'c.md', isCrossed: true },
      ],
    })

    const resolved = await resolveLinks('a.md', '[[B]] [[C|see C]] [[D]]')

    expect(request.resolveAddresses.mock.calls.at(-1)?.[0]).toMatchObject({
      from: 'a.md',
      text: '[[B]] [[C|see C]] [[D]]',
      vault: '',
    })
    expect(resolved).toEqual([
      { raw: '[[B]]', address: 'name://B', text: 'B', path: 'b.md' },
      { raw: '[[C|see C]]', address: 'name://C', text: 'see C', path: '' },
      { raw: '[[D]]', address: 'name://D', text: 'D', path: '' },
    ])
  })
})

describe('the note the vault opens first', () => {
  it('is requested of no particular caller, and is none where the answer names none', async () => {
    request.getOpeningNote.mockResolvedValue({})

    expect(await getInitialOpenPath()).toBeNull()
    expect(request.getOpeningNote.mock.calls.at(-1)?.[0]).toMatchObject({})
  })

  it('is the path the answer names', async () => {
    request.getOpeningNote.mockResolvedValue({ note: { path: 'opening.md', title: 'Opening' } })

    expect(await getInitialOpenPath()).toEqual({ path: 'opening.md' })
  })
})

describe('the edits made to notes', () => {
  it('are read, and a missing span is read as the start', async () => {
    const edits = [
      { change: '1', path: 'a.md', range: { start: 2, end: 5 }, text: 'abc', isFinal: false },
      { change: '2', path: 'a.md', text: 'x', isFinal: true },
    ]
    request.watchEdits.mockImplementation(async function* () {
      yield* edits
    })

    const observed = []
    for await (const edit of watchEdits(new AbortController().signal)) observed.push(edit)

    expect(observed).toEqual([
      { change: '1', path: 'a.md', span: { start: 2, end: 5 }, text: 'abc', isComplete: false },
      { change: '2', path: 'a.md', span: { start: 0, end: 0 }, text: 'x', isComplete: true },
    ])
  })
})

describe('a neighborhood', () => {
  it('has its neighborhood requested of its path', async () => {
    request.getNeighborhood.mockResolvedValue({
      focus: { path: 'a.md', title: 'A' },
      related: [],
      connections: [],
    })

    const neighborhood = await getNeighborhood('a.md')

    expect(neighborhood.focus.path).toBe('a.md')
    expect(request.getNeighborhood.mock.calls.at(-1)?.[0]).toMatchObject({ path: 'a.md' })
  })

  it('is drawn around the note in focus, with each note in the relation it holds', async () => {
    request.getNeighborhood.mockResolvedValue({
      focus: { path: 'Rota.md', title: 'The rota' },
      focusType: 'sample',
      related: [
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
          firstSide: { role: NOTE_ROLE.CHILD, label: 'contains rota', marker: NOTE_MARKER.ARROW },
          secondSide: { role: NOTE_ROLE.PARENT, label: 'part of' },
        },
      ],
    })

    const neighborhood = await getNeighborhood('Rota.md')

    expect(neighborhood.focus).toEqual({ path: 'Rota.md', title: 'The rota' })
    expect(neighborhood.focusType).toBe('sample')
    expect(neighborhood.related).toEqual([
      { path: 'Allotments.md', title: 'Allotments', type: 'note', relation: 'parent' },
    ])
    expect(neighborhood.connections).toEqual([
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
    request.getNeighborhood.mockResolvedValue({
      focus: { path: 'Rota.md', title: 'The rota' },
      related: [
        { note: { path: 'Odd.md', title: 'Odd' }, relation: NOTE_RELATION.UNSPECIFIED },
        { relation: NOTE_RELATION.CHILD },
      ],
    })

    expect((await getNeighborhood('Rota.md')).related).toEqual([])
  })

  it('relies on nothing where the vault no longer holds the note', async () => {
    request.getNeighborhood.mockResolvedValue({ related: [] })

    expect((await getNeighborhood('Gone.md')).focus.path).toBe('')
  })
})
