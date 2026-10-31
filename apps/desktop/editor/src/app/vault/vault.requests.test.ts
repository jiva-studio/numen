import { describe, expect, it, vi } from 'vitest'
import { ErrorCode, Marker, Role, VaultsErrorCode } from '@numen/protocol'
import { core, vaults } from './index'
import { WINDOW } from './clients'

const { requestLog } = vi.hoisted(() => {
  const request = {
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
    listFiles: vi.fn(),
    moveFile: vi.fn(),
    createFolder: vi.fn(),
    listFileKinds: vi.fn(),
    listVaults: vi.fn(),
    getCurrentVault: vi.fn(),
    chooseFolder: vi.fn(),
    addVault: vi.fn(),
    renameVault: vi.fn(),
    removeVault: vi.fn(),
    openVault: vi.fn(),
  }
  vi.stubGlobal('window', { location: { origin: 'http://numen.invalid' } })
  return { requestLog: request }
})

vi.mock('@/shared/clients', () => ({
  vault: requestLog,
  agentService: requestLog,
  workspace: requestLog,
  files: requestLog,
  notes: requestLog,
  search: requestLog,
  vaultsService: requestLog,
  settingsService: requestLog,
  windowService: requestLog,
}))

describe('a note requested for', () => {
  it('is created at the path of its folder, and rejected with error of the error', async () => {
    requestLog.createNote.mockResolvedValue({ path: 'Notes/Heat.md' })
    const created = await core.create({ title: 'Heat', folder: 'Notes' })
    expect(created).toEqual({ ok: true, value: { path: 'Notes/Heat.md' } })
    expect(requestLog.createNote.mock.calls.at(-1)?.[0].path).toBe('Notes')

    requestLog.createNote.mockResolvedValue({ error: ErrorCode.ALREADY_EXISTS })
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
    expect(requestLog.renameNote.mock.calls.at(-1)?.[0]).toEqual({ path: 'A.md', title: 'B' })
    expect(renamedItems).toEqual({
      ok: true,
      value: { path: 'B.md', title: 'B', hasFrontmatter: true, moved: null },
    })

    requestLog.renameNote.mockResolvedValue({ error: ErrorCode.STALE })
    expect(await core.rename('A.md', 'B')).toEqual({ ok: false, error: 'changed' })

    requestLog.renameNote.mockResolvedValue({ error: ErrorCode.ALREADY_EXISTS })
    expect(await core.rename('A.md', 'B')).toEqual({ ok: false, error: 'alreadyExists' })
  })

  it('has its headings requested of every path given', async () => {
    requestLog.listHeadings.mockResolvedValue({ headings: [] })
    await core.headings(['a.md', 'b.md'])
    expect(requestLog.listHeadings.mock.calls.at(-1)?.[0]).toEqual({ paths: ['a.md', 'b.md'] })
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
    expect(requestLog.resolveAddresses.mock.calls.at(-1)?.[0]).toEqual({
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
  it('is removed to the trash unless it is to be destroyed, and rejected with error', async () => {
    requestLog.removeFile.mockResolvedValue({ trashed: true, dangling: [] })
    await core.remove('a.md')
    expect(requestLog.removeFile.mock.calls.at(-1)?.[0]).toEqual({ path: 'a.md', destroy: false })
    await core.remove('a.md', false)
    expect(requestLog.removeFile.mock.calls.at(-1)?.[0]).toEqual({ path: 'a.md', destroy: false })
    await core.remove('a.md', true)
    expect(requestLog.removeFile.mock.calls.at(-1)?.[0]).toEqual({ path: 'a.md', destroy: true })

    requestLog.removeFile.mockResolvedValue({ error: ErrorCode.MISSING })
    expect(await core.remove('a.md')).toEqual({ ok: false, error: 'missing' })
  })

  it('is listed under its folder', async () => {
    requestLog.listFiles.mockResolvedValue({ entries: [] })
    await core.list('Notes')
    expect(requestLog.listFiles.mock.calls.at(-1)?.[0]).toEqual({ path: 'Notes' })
  })

  it('is moved from one path to another, and rejected with error', async () => {
    requestLog.moveFile.mockResolvedValue({})
    await core.move('a.md', 'b.md')
    expect(requestLog.moveFile.mock.calls.at(-1)?.[0]).toEqual({ from: 'a.md', to: 'b.md' })

    requestLog.moveFile.mockResolvedValue({ error: ErrorCode.ALREADY_EXISTS })
    expect(await core.move('a.md', 'b.md')).toEqual({ ok: false, error: 'alreadyExists' })
  })

  it('has a folder created at its path', async () => {
    requestLog.createFolder.mockResolvedValue({})
    await core.createFolder('Notes/New')
    expect(requestLog.createFolder.mock.calls.at(-1)?.[0]).toEqual({ path: 'Notes/New' })
  })

  it('has its kinds requested of every path given', async () => {
    requestLog.listFileKinds.mockResolvedValue({ kinds: [] })
    await core.fileKinds(['a.epub', 'b.pdf'])
    expect(requestLog.listFileKinds.mock.calls.at(-1)?.[0]).toEqual({ paths: ['a.epub', 'b.pdf'] })
  })
})

describe('a vault requested for', () => {
  it('is listed with the vault this window shows', async () => {
    requestLog.listVaults.mockResolvedValue({ vaults: [] })
    requestLog.getCurrentVault.mockResolvedValue({ vault: 'v1' })
    await vaults.list()
    expect(requestLog.listVaults.mock.calls.at(-1)?.[0]).toEqual({})
    expect(requestLog.getCurrentVault.mock.calls.at(-1)?.[0]).toEqual({ window: WINDOW })
  })

  it('is chosen from a folder dialog that starts nowhere in particular', async () => {
    requestLog.chooseFolder.mockResolvedValue({ isChosen: false, path: '/ignored' })
    expect(await vaults.choose('Pick')).toBe('')
    expect(requestLog.chooseFolder.mock.calls.at(-1)?.[0]).toEqual({
      title: 'Pick',
      startingAt: '',
    })
  })

  it('is added, renamed, removed and opened by what the schema names', async () => {
    requestLog.addVault.mockResolvedValue({})
    await vaults.add('/v', 'Name')
    expect(requestLog.addVault.mock.calls.at(-1)?.[0]).toEqual({ path: '/v', name: 'Name' })

    requestLog.renameVault.mockResolvedValue({})
    await vaults.rename('v1', 'Other')
    expect(requestLog.renameVault.mock.calls.at(-1)?.[0]).toEqual({ id: 'v1', name: 'Other' })

    requestLog.removeVault.mockResolvedValue({})
    await vaults.remove('v1', true)
    expect(requestLog.removeVault.mock.calls.at(-1)?.[0]).toEqual({ id: 'v1', trash: true })

    requestLog.openVault.mockResolvedValue({})
    await vaults.open('v1')
    expect(requestLog.openVault.mock.calls.at(-1)?.[0]).toEqual({ id: 'v1' })
  })

  it('fails in the words of the error', async () => {
    requestLog.addVault.mockResolvedValue({ error: VaultsErrorCode.NAME_TAKEN })
    expect(await vaults.add('/v', 'Name')).toEqual({ ok: false, error: 'nameTaken' })
  })
})

describe('a note followed', () => {
  it('has its neighbourhood requested of its path', async () => {
    requestLog.getNeighbourhood.mockResolvedValue({
      focus: { path: 'a.md', title: 'A' },
      related: [],
    })
    await core.neighbourhood('a.md')
    expect(requestLog.getNeighbourhood.mock.calls.at(-1)?.[0]).toEqual({ path: 'a.md' })
  })

  it('has the opening note requested of nobody in particular', async () => {
    requestLog.getOpeningNote.mockResolvedValue({})
    expect(await core.getInitialOpenPath()).toBeNull()
    expect(requestLog.getOpeningNote.mock.calls.at(-1)?.[0]).toEqual({})
  })

  it('has a connection read with the role the core derived, and the markers of its records', async () => {
    requestLog.getConnection.mockResolvedValue({
      connection: {
        first: 'a.md',
        second: 'b.md',
        role: Role.PARENT,
        firstSide: { label: 'under', note: 'why', marker: Marker.ARROW },
        secondSide: { label: '', note: '', marker: Marker.UNSPECIFIED },
      },
    })

    const read = await core.relations.getConnection('a.md', 'b.md', 'preset')

    expect(requestLog.getConnection.mock.calls.at(-1)?.[0]).toEqual({
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
    expect(requestLog.saveConnection.mock.calls.at(-1)?.[0]).toEqual({
      connection: {
        first: 'a.md',
        second: 'b.md',
        role: Role.CHILD,
        firstSide: { type: '', note: '', label: '', marker: Marker.ARROW },
        secondSide: undefined,
      },
    })
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
    requestLog.saveConnection.mockResolvedValueOnce({ error: ErrorCode.MISSING })
    expect(await core.relations.saveConnection(connection)).toEqual({ ok: false, error: 'missing' })
  })

  it('has one side, or both, removed under the type of the records', async () => {
    requestLog.removeSide.mockResolvedValue({})
    requestLog.removeConnection.mockResolvedValue({})

    await core.relations.removeSide('a.md', 'b.md', 'preset')
    await core.relations.removeConnection('a.md', 'b.md')

    expect(requestLog.removeSide.mock.calls.at(-1)?.[0]).toEqual({
      path: 'a.md',
      to: 'b.md',
      type: 'preset',
    })
    expect(requestLog.removeConnection.mock.calls.at(-1)?.[0]).toEqual({
      first: 'a.md',
      second: 'b.md',
      type: '',
    })
  })

  it('has one record written under the wire role and marker', async () => {
    requestLog.joinNotes.mockResolvedValue({})

    await core.relations.join('a.md', 'b.md', { role: 'jump', label: 'see', marker: 'arrow' })

    expect(requestLog.joinNotes.mock.calls.at(-1)?.[0]).toEqual({
      from: 'a.md',
      to: 'b.md',
      side: { role: Role.JUMP, type: '', note: '', label: 'see', marker: Marker.ARROW },
    })
  })

  it('has a related note made by the core, which answers with its path', async () => {
    requestLog.createRelatedNote.mockResolvedValue({ path: 'physics/Untitled note.md' })

    const made = await core.relations.createRelated({ from: 'physics/a.md', role: 'child' })

    expect(made).toEqual({ ok: true, value: 'physics/Untitled note.md' })
    expect(requestLog.createRelatedNote.mock.calls.at(-1)?.[0]).toEqual({
      from: 'physics/a.md',
      role: Role.CHILD,
      title: '',
      type: '',
      marker: Marker.UNSPECIFIED,
    })

    requestLog.createRelatedNote.mockResolvedValue({ isNameExhausted: true })
    expect(await core.relations.createRelated({ from: 'a.md', role: 'jump' })).toEqual({
      ok: false,
      error: 'nameExhausted',
    })
  })

  it('has the backlinks of a note listed with both roles', async () => {
    requestLog.listBacklinks.mockResolvedValue({
      backlinks: [
        {
          from: { path: 'b.md', title: 'B' },
          role: Role.PARENT,
          roleFromHere: Role.CHILD,
          type: '',
          note: '',
          label: 'under',
          marker: Marker.ARROW,
        },
      ],
    })

    expect(await core.relations.listBacklinks('a.md')).toEqual([
      {
        from: { path: 'b.md', title: 'B' },
        role: 'parent',
        roleFromHere: 'child',
        type: '',
        reason: '',
        label: 'under',
        marker: 'arrow',
      },
    ])
  })

  it('has its edits read under the signal of the window, and a missing span read as the start', async () => {
    const edits = [
      { change: 1, path: 'a.md', range: { start: 2, end: 5 }, text: 'abc', isFinal: false },
      { change: 2, path: 'a.md', text: 'x', isFinal: true },
    ]
    requestLog.watchEdits.mockImplementation(async function* () {
      yield* edits
    })
    const signal = new AbortController().signal

    const observed = []
    for await (const edit of core.watchEdits(signal)) observed.push(edit)

    expect(requestLog.watchEdits.mock.calls.at(-1)).toEqual([{}, { signal }])
    expect(observed).toEqual([
      { change: 1, path: 'a.md', span: { start: 2, end: 5 }, text: 'abc', isComplete: false },
      { change: 2, path: 'a.md', span: { start: 0, end: 0 }, text: 'x', isComplete: true },
    ])
  })
})

describe('a note read', () => {
  it('is requested for by its path', async () => {
    requestLog.readNote.mockResolvedValue({ body: 'prose' })
    await core.read('Notes/Heat.md')
    expect(requestLog.readNote.mock.calls.at(-1)?.[0]).toEqual({ path: 'Notes/Heat.md' })
  })
})
