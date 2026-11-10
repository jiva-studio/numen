import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { NOTE_ERROR_CODE, NOTE_MARKER, NOTE_ROLE } from '@numen/core-client'
import { installCoreHandlers } from '@numen/core-client/testing'
import { core } from './index'

const requestLog = {
  createNote: vi.fn(),
  getNeighbourhood: vi.fn(),
  getOpeningNote: vi.fn(),
  readNote: vi.fn(),
  watchEdits: vi.fn(),
  getConnection: vi.fn(),
  saveConnection: vi.fn(),
  removeSide: vi.fn(),
  removeConnection: vi.fn(),
  joinNotes: vi.fn(),
  createRelatedNote: vi.fn(),
  listBacklinks: vi.fn(),
  renameNote: vi.fn(),
  listHeadings: vi.fn(),
  resolveAddresses: vi.fn(),
  removeFile: vi.fn(),
  emptyTrash: vi.fn(),
  listFiles: vi.fn(),
  moveFile: vi.fn(),
  createFolder: vi.fn(),
  listFileKinds: vi.fn(),
}

let restore: () => void

beforeEach(() => {
  restore = installCoreHandlers(requestLog)
})

afterEach(() => {
  restore()
})

describe('a note requested for', () => {
  it('is created at the path of its folder, and rejected with error of the error', async () => {
    requestLog.createNote.mockResolvedValue({ path: 'Notes/Heat.md' })
    const created = await core.create({ title: 'Heat', folder: 'Notes' })
    expect(created).toEqual({ ok: true, value: { path: 'Notes/Heat.md' } })
    expect(requestLog.createNote.mock.calls.at(-1)?.[0].path).toBe('Notes')

    requestLog.createNote.mockResolvedValue({ error: NOTE_ERROR_CODE.ALREADY_EXISTS })
    const outcome = await core.create({ title: 'Heat', folder: 'Notes' })
    expect(outcome).toEqual({ ok: false, error: 'alreadyExists' })

    requestLog.createNote.mockResolvedValue({ isNameExhausted: true })
    expect(await core.create({ title: '', folder: '' })).toEqual({
      ok: false,
      error: 'alreadyExists',
    })
  })

  it('is renamed by the path it has, and a stale answer is a change under the window', async () => {
    requestLog.renameNote.mockResolvedValue({ path: 'B.md', title: 'B', nameSource: 1 })
    const renamedItems = await core.rename('A.md', 'B')
    expect(requestLog.renameNote.mock.calls.at(-1)?.[0]).toMatchObject({ path: 'A.md', title: 'B' })
    expect(renamedItems).toEqual({
      ok: true,
      value: { path: 'B.md', title: 'B', hasFrontmatter: true, moved: null },
    })

    requestLog.renameNote.mockResolvedValue({ error: NOTE_ERROR_CODE.CONFLICT })
    expect(await core.rename('A.md', 'B')).toEqual({ ok: false, error: 'changed' })

    requestLog.renameNote.mockResolvedValue({ error: NOTE_ERROR_CODE.ALREADY_EXISTS })
    expect(await core.rename('A.md', 'B')).toEqual({ ok: false, error: 'alreadyExists' })
  })

  it('has its headings requested of every path given', async () => {
    requestLog.listHeadings.mockResolvedValue({ headings: [] })
    await core.headings(['a.md', 'b.md'])
    expect(requestLog.listHeadings.mock.calls.at(-1)?.[0]).toMatchObject({
      paths: ['a.md', 'b.md'],
    })
  })

  it('resolves wikilinks of a text relative to the source note and omits crossed links', async () => {
    requestLog.resolveAddresses.mockResolvedValue({
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
    const resolved = await core.resolveLinks('a.md', '[[B]] [[C|see C]] [[D]]')
    expect(requestLog.resolveAddresses.mock.calls.at(-1)?.[0]).toMatchObject({
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

describe('a file requested for', () => {
  it('is removed to the trash, and rejected with error', async () => {
    requestLog.removeFile.mockResolvedValue({ trashed: true, dangling: [] })
    await core.remove('a.md')
    expect(requestLog.removeFile.mock.calls.at(-1)?.[0]).toMatchObject({ path: 'a.md' })

    requestLog.removeFile.mockResolvedValue({ error: NOTE_ERROR_CODE.NOT_FOUND })
    expect(await core.remove('a.md')).toEqual({ ok: false, error: 'notFound' })
  })

  it('is the count of the paths the trash could not delete, when it is emptied', async () => {
    requestLog.emptyTrash.mockResolvedValue({ undeleted: ['.trash/a.md', '.trash/b/c.md'] })

    expect(await core.emptyTrash()).toBe(2)
    expect(requestLog.emptyTrash.mock.lastCall?.[0]).toMatchObject({})
  })

  it('is listed under its folder', async () => {
    requestLog.listFiles.mockResolvedValue({ entries: [] })
    await core.list('Notes')
    expect(requestLog.listFiles.mock.calls.at(-1)?.[0]).toMatchObject({ path: 'Notes' })
  })

  it('is moved from one path to another, and rejected with error', async () => {
    requestLog.moveFile.mockResolvedValue({})
    await core.move('a.md', 'b.md')
    expect(requestLog.moveFile.mock.calls.at(-1)?.[0]).toMatchObject({ from: 'a.md', to: 'b.md' })

    requestLog.moveFile.mockResolvedValue({ error: NOTE_ERROR_CODE.ALREADY_EXISTS })
    expect(await core.move('a.md', 'b.md')).toEqual({ ok: false, error: 'alreadyExists' })
  })

  it('has a folder created at its path', async () => {
    requestLog.createFolder.mockResolvedValue({})
    await core.createFolder('Notes/New')
    expect(requestLog.createFolder.mock.calls.at(-1)?.[0]).toMatchObject({ path: 'Notes/New' })
  })

  it('has its kinds requested of every path given', async () => {
    requestLog.listFileKinds.mockResolvedValue({ kinds: [] })
    await core.fileKinds(['a.epub', 'b.pdf'])
    expect(requestLog.listFileKinds.mock.calls.at(-1)?.[0]).toMatchObject({
      paths: ['a.epub', 'b.pdf'],
    })
  })
})

describe('a note followed', () => {
  it('has its neighbourhood requested of its path', async () => {
    requestLog.getNeighbourhood.mockResolvedValue({
      focus: { path: 'a.md', title: 'A' },
      related: [],
      connections: [],
    })
    await core.neighbourhood('a.md')
    expect(requestLog.getNeighbourhood.mock.calls.at(-1)?.[0]).toMatchObject({ path: 'a.md' })
  })

  it('has the opening note requested of no particular caller', async () => {
    requestLog.getOpeningNote.mockResolvedValue({})
    expect(await core.getInitialOpenPath()).toBeNull()
    expect(requestLog.getOpeningNote.mock.calls.at(-1)?.[0]).toMatchObject({})
  })

  it('has a connection read with the role the core derived, and the markers of its records', async () => {
    requestLog.getConnection.mockResolvedValue({
      connection: {
        first: 'a.md',
        second: 'b.md',
        role: NOTE_ROLE.PARENT,
        firstSide: { label: 'under', note: 'why', marker: NOTE_MARKER.ARROW },
        secondSide: { label: '', note: '', marker: NOTE_MARKER.UNSPECIFIED },
      },
    })

    const read = await core.relations.getConnection('a.md', 'b.md', 'preset')

    expect(requestLog.getConnection.mock.calls.at(-1)?.[0]).toMatchObject({
      first: 'a.md',
      second: 'b.md',
      type: 'preset',
    })
    expect(read).toEqual({
      ok: true,
      value: {
        first: 'a.md',
        second: 'b.md',
        role: 'parent',
        type: 'preset',
        firstSide: { label: 'under', marker: 'arrow', reason: 'why' },
        secondSide: { label: '', marker: null, reason: '' },
      },
    })

    requestLog.getConnection.mockResolvedValueOnce({ error: NOTE_ERROR_CODE.NOT_FOUND })
    expect(await core.relations.getConnection('a.md', 'b.md')).toEqual({
      ok: false,
      error: 'notFound',
    })
    expect(requestLog.getConnection.mock.calls.at(-1)?.[0]).toMatchObject({
      first: 'a.md',
      second: 'b.md',
      type: '',
    })

    requestLog.getConnection.mockResolvedValueOnce({})
    expect(await core.relations.getConnection('a.md', 'b.md')).toEqual({
      ok: false,
      error: 'unreadable',
    })
  })

  it('has a connection saved in one request, with no role on its records', async () => {
    requestLog.saveConnection.mockResolvedValue({})

    const saved = await core.relations.saveConnection({
      first: 'a.md',
      second: 'b.md',
      role: 'child',
      type: '',
      firstSide: { label: '', marker: 'arrow', reason: '' },
      secondSide: null,
    })

    expect(saved).toEqual({ ok: true, value: undefined })
    expect(requestLog.saveConnection.mock.calls.at(-1)?.[0]).toMatchObject({
      connection: {
        first: 'a.md',
        second: 'b.md',
        role: NOTE_ROLE.CHILD,
        firstSide: { type: '', note: '', label: '', marker: NOTE_MARKER.ARROW },
      },
    })
    expect(requestLog.saveConnection.mock.calls.at(-1)?.[0].connection.secondSide).toBeUndefined()
  })

  it('names a connection the core refused by why it was refused', async () => {
    const connection = {
      first: 'a.md',
      second: 'b.md',
      role: null,
      type: '',
      firstSide: null,
      secondSide: null,
    } as const
    requestLog.saveConnection.mockResolvedValueOnce({ isCycle: true })
    expect(await core.relations.saveConnection(connection)).toEqual({ ok: false, error: 'cycle' })
    requestLog.saveConnection.mockResolvedValueOnce({ isInvalid: true })
    expect(await core.relations.saveConnection(connection)).toEqual({ ok: false, error: 'invalid' })
    requestLog.saveConnection.mockResolvedValueOnce({ error: NOTE_ERROR_CODE.NOT_FOUND })
    expect(await core.relations.saveConnection(connection)).toEqual({
      ok: false,
      error: 'notFound',
    })
  })

  it('has one side, or both, removed under the type of the records', async () => {
    requestLog.removeSide.mockResolvedValue({})
    requestLog.removeConnection.mockResolvedValue({})

    await core.relations.removeSide('a.md', 'b.md', 'preset')
    await core.relations.removeConnection('a.md', 'b.md')

    expect(requestLog.removeSide.mock.calls.at(-1)?.[0]).toMatchObject({
      path: 'a.md',
      to: 'b.md',
      type: 'preset',
    })
    expect(requestLog.removeConnection.mock.calls.at(-1)?.[0]).toMatchObject({
      first: 'a.md',
      second: 'b.md',
      type: '',
    })
  })

  it('has one record written under the wire role and marker', async () => {
    requestLog.joinNotes.mockResolvedValue({})

    await core.relations.join('a.md', 'b.md', { role: 'jump', label: 'see', marker: 'arrow' })

    expect(requestLog.joinNotes.mock.calls.at(-1)?.[0]).toMatchObject({
      from: 'a.md',
      to: 'b.md',
      side: { role: NOTE_ROLE.JUMP, type: '', note: '', label: 'see', marker: NOTE_MARKER.ARROW },
    })
  })

  it('has a related note made by the core, which answers with its path', async () => {
    requestLog.createRelatedNote.mockResolvedValue({ path: 'physics/Untitled note.md' })

    const made = await core.relations.createRelated({ from: 'physics/a.md', role: 'child' })

    expect(made).toEqual({ ok: true, value: 'physics/Untitled note.md' })
    expect(requestLog.createRelatedNote.mock.calls.at(-1)?.[0]).toMatchObject({
      from: 'physics/a.md',
      role: NOTE_ROLE.CHILD,
      title: '',
      type: '',
      marker: NOTE_MARKER.UNSPECIFIED,
    })

    requestLog.createRelatedNote.mockResolvedValue({ isNameExhausted: true })
    expect(await core.relations.createRelated({ from: 'a.md', role: 'jump' })).toEqual({
      ok: false,
      error: 'nameExhausted',
    })

    requestLog.createRelatedNote.mockResolvedValue({ error: NOTE_ERROR_CODE.NOT_FOUND })
    expect(await core.relations.createRelated({ from: 'a.md', role: 'jump' })).toEqual({
      ok: false,
      error: 'notFound',
    })
  })

  it('has the backlinks of a note listed with both roles', async () => {
    requestLog.listBacklinks.mockResolvedValue({
      backlinks: [
        {
          from: { path: 'b.md', title: 'B' },
          role: NOTE_ROLE.PARENT,
          sourceRole: NOTE_ROLE.CHILD,
          type: '',
          note: '',
          label: 'under',
          marker: NOTE_MARKER.ARROW,
        },
      ],
    })

    expect(await core.relations.listBacklinks('a.md')).toEqual([
      {
        from: { path: 'b.md', title: 'B' },
        role: 'parent',
        sourceRole: 'child',
        type: '',
        reason: '',
        label: 'under',
        marker: 'arrow',
      },
    ])
    expect(requestLog.listBacklinks.mock.lastCall?.[0]).toMatchObject({ path: 'a.md' })
  })

  it('has its edits read, and a missing span read as the start', async () => {
    const edits = [
      { change: '1', path: 'a.md', range: { start: 2, end: 5 }, text: 'abc', isFinal: false },
      { change: '2', path: 'a.md', text: 'x', isFinal: true },
    ]
    requestLog.watchEdits.mockImplementation(async function* () {
      yield* edits
    })
    const signal = new AbortController().signal

    const observed = []
    for await (const edit of core.watchEdits(signal)) observed.push(edit)

    expect(observed).toEqual([
      { change: '1', path: 'a.md', span: { start: 2, end: 5 }, text: 'abc', isComplete: false },
      { change: '2', path: 'a.md', span: { start: 0, end: 0 }, text: 'x', isComplete: true },
    ])
  })
})

describe('a note read', () => {
  it('is requested for by its path', async () => {
    requestLog.readNote.mockResolvedValue({ body: 'prose' })
    await core.read('Notes/Heat.md')
    expect(requestLog.readNote.mock.calls.at(-1)?.[0]).toMatchObject({ path: 'Notes/Heat.md' })
  })
})
