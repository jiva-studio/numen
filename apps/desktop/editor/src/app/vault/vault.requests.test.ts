import { describe, expect, it, vi } from 'vitest'
import { ErrorCode, Role, VaultsErrorCode } from '@numen/protocol'
import { core, vaults } from './index'
import { WINDOW } from './clients'

const { asked } = vi.hoisted(() => {
  const request = {
    createNote: vi.fn(),
    getNeighbourhood: vi.fn(),
    getOpeningNote: vi.fn(),
    readNote: vi.fn(),
    watchEdits: vi.fn(),
    removeLink: vi.fn(),
    replaceLinks: vi.fn(),
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
  return { asked: request }
})

vi.mock('@/shared/clients', () => ({
  vault: asked,
  agentService: asked,
  workspace: asked,
  files: asked,
  notes: asked,
  search: asked,
  vaultsService: asked,
  settingsService: asked,
  windowService: asked,
}))

describe('a note requested for', () => {
  it('is created at the path of its folder, and rejected with error of the error', async () => {
    asked.createNote.mockResolvedValue({ path: 'Notes/Heat.md' })
    const created = await core.create({ title: 'Heat', folder: 'Notes', links: [] })
    expect(created).toEqual({ ok: true, value: { path: 'Notes/Heat.md' } })
    expect(asked.createNote.mock.calls.at(-1)?.[0].path).toBe('Notes')

    asked.createNote.mockResolvedValue({ error: ErrorCode.ALREADY_EXISTS })
    const outcome = await core.create({ title: 'Heat', folder: 'Notes', links: [] })
    expect(outcome).toEqual({ ok: false, error: 'alreadyExists' })
  })

  it('is renamed by the path it has, and a stale answer is a change under the window', async () => {
    asked.renameNote.mockResolvedValue({ path: 'B.md', title: 'B', nameSource: 1 })
    const renamedItems = await core.rename('A.md', 'B')
    expect(asked.renameNote.mock.calls.at(-1)?.[0]).toEqual({ path: 'A.md', title: 'B' })
    expect(renamedItems).toEqual({
      ok: true,
      value: { path: 'B.md', title: 'B', hasFrontmatter: true, moved: null },
    })

    asked.renameNote.mockResolvedValue({ error: ErrorCode.STALE })
    expect(await core.rename('A.md', 'B')).toEqual({ ok: false, error: 'changed' })

    asked.renameNote.mockResolvedValue({ error: ErrorCode.ALREADY_EXISTS })
    expect(await core.rename('A.md', 'B')).toEqual({ ok: false, error: 'alreadyExists' })
  })

  it('has its headings requested of every path given', async () => {
    asked.listHeadings.mockResolvedValue({ headings: [] })
    await core.headings(['a.md', 'b.md'])
    expect(asked.listHeadings.mock.calls.at(-1)?.[0]).toEqual({ paths: ['a.md', 'b.md'] })
  })

  it('has the wikilinks of a text resolved from the note that wrote them, less the ones that cross', async () => {
    asked.resolveAddresses.mockResolvedValue({
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
    expect(asked.resolveAddresses.mock.calls.at(-1)?.[0]).toEqual({
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
    asked.removeFile.mockResolvedValue({ trashed: true, dangling: [] })
    await core.remove('a.md')
    expect(asked.removeFile.mock.calls.at(-1)?.[0]).toEqual({ path: 'a.md', destroy: false })
    await core.remove('a.md', false)
    expect(asked.removeFile.mock.calls.at(-1)?.[0]).toEqual({ path: 'a.md', destroy: false })
    await core.remove('a.md', true)
    expect(asked.removeFile.mock.calls.at(-1)?.[0]).toEqual({ path: 'a.md', destroy: true })

    asked.removeFile.mockResolvedValue({ error: ErrorCode.MISSING })
    expect(await core.remove('a.md')).toEqual({ ok: false, error: 'missing' })
  })

  it('is listed under its folder', async () => {
    asked.listFiles.mockResolvedValue({ entries: [] })
    await core.list('Notes')
    expect(asked.listFiles.mock.calls.at(-1)?.[0]).toEqual({ path: 'Notes' })
  })

  it('is moved from one path to another, and rejected with error', async () => {
    asked.moveFile.mockResolvedValue({})
    await core.move('a.md', 'b.md')
    expect(asked.moveFile.mock.calls.at(-1)?.[0]).toEqual({ from: 'a.md', to: 'b.md' })

    asked.moveFile.mockResolvedValue({ error: ErrorCode.ALREADY_EXISTS })
    expect(await core.move('a.md', 'b.md')).toEqual({ ok: false, error: 'alreadyExists' })
  })

  it('has a folder created at its path', async () => {
    asked.createFolder.mockResolvedValue({})
    await core.createFolder('Notes/New')
    expect(asked.createFolder.mock.calls.at(-1)?.[0]).toEqual({ path: 'Notes/New' })
  })

  it('has its kinds requested of every path given', async () => {
    asked.listFileKinds.mockResolvedValue({ kinds: [] })
    await core.fileKinds(['a.epub', 'b.pdf'])
    expect(asked.listFileKinds.mock.calls.at(-1)?.[0]).toEqual({ paths: ['a.epub', 'b.pdf'] })
  })
})

describe('a vault requested for', () => {
  it('is listed with the vault this window shows', async () => {
    asked.listVaults.mockResolvedValue({ vaults: [] })
    asked.getCurrentVault.mockResolvedValue({ vault: 'v1' })
    await vaults.list()
    expect(asked.listVaults.mock.calls.at(-1)?.[0]).toEqual({})
    expect(asked.getCurrentVault.mock.calls.at(-1)?.[0]).toEqual({ window: WINDOW })
  })

  it('is chosen from a folder dialog that starts nowhere in particular', async () => {
    asked.chooseFolder.mockResolvedValue({ isChosen: false, path: '/ignored' })
    expect(await vaults.choose('Pick')).toBe('')
    expect(asked.chooseFolder.mock.calls.at(-1)?.[0]).toEqual({ title: 'Pick', startingAt: '' })
  })

  it('is added, renamed, removed and opened by what the schema names', async () => {
    asked.addVault.mockResolvedValue({})
    await vaults.add('/v', 'Name')
    expect(asked.addVault.mock.calls.at(-1)?.[0]).toEqual({ path: '/v', name: 'Name' })

    asked.renameVault.mockResolvedValue({})
    await vaults.rename('v1', 'Other')
    expect(asked.renameVault.mock.calls.at(-1)?.[0]).toEqual({ id: 'v1', name: 'Other' })

    asked.removeVault.mockResolvedValue({})
    await vaults.remove('v1', true)
    expect(asked.removeVault.mock.calls.at(-1)?.[0]).toEqual({ id: 'v1', trash: true })

    asked.openVault.mockResolvedValue({})
    await vaults.open('v1')
    expect(asked.openVault.mock.calls.at(-1)?.[0]).toEqual({ id: 'v1' })
  })

  it('fails in the words of the error', async () => {
    asked.addVault.mockResolvedValue({ error: VaultsErrorCode.NAME_TAKEN })
    expect(await vaults.add('/v', 'Name')).toEqual({ ok: false, error: 'nameTaken' })
  })
})

describe('a note followed', () => {
  it('has its neighbourhood requested of its path', async () => {
    asked.getNeighbourhood.mockResolvedValue({ focus: { path: 'a.md', title: 'A' }, related: [] })
    await core.neighbourhood('a.md')
    expect(asked.getNeighbourhood.mock.calls.at(-1)?.[0]).toEqual({ path: 'a.md' })
  })

  it('has the opening note requested of nobody in particular', async () => {
    asked.getOpeningNote.mockResolvedValue({})
    expect(await core.getInitialOpenPath()).toBeNull()
    expect(asked.getOpeningNote.mock.calls.at(-1)?.[0]).toEqual({})
  })

  it('has a link removed under the role it was written with, or under none', async () => {
    asked.removeLink.mockResolvedValue({})
    await core.removeLink('a.md', 'b.md', 'jump')
    expect(asked.removeLink.mock.calls.at(-1)?.[0]).toEqual({
      path: 'a.md',
      to: 'b.md',
      role: Role.JUMP,
    })
    await core.removeLink('a.md', 'b.md')
    expect(asked.removeLink.mock.calls.at(-1)?.[0]).toEqual({ path: 'a.md', to: 'b.md' })
  })

  it('has the links of a pair replaced in one request, each under the wire role', async () => {
    asked.replaceLinks.mockResolvedValue({})
    await core.replaceLinks('a.md', 'b.md', [
      { from: 'a.md', to: 'b.md', role: 'jump', label: 'see' },
    ])
    expect(asked.replaceLinks.mock.calls.at(-1)?.[0]).toEqual({
      first: 'a.md',
      second: 'b.md',
      links: [{ from: 'a.md', to: 'b.md', role: Role.JUMP, label: 'see' }],
    })
  })

  it('has its edits read under the signal of the window, and a missing span read as the start', async () => {
    const edits = [
      { change: 1, path: 'a.md', range: { start: 2, end: 5 }, text: 'abc', isFinal: false },
      { change: 2, path: 'a.md', text: 'x', isFinal: true },
    ]
    asked.watchEdits.mockImplementation(async function* () {
      yield* edits
    })
    const signal = new AbortController().signal

    const observed = []
    for await (const edit of core.watchEdits(signal)) observed.push(edit)

    expect(asked.watchEdits.mock.calls.at(-1)).toEqual([{}, { signal }])
    expect(observed).toEqual([
      { change: 1, path: 'a.md', span: { start: 2, end: 5 }, text: 'abc', isComplete: false },
      { change: 2, path: 'a.md', span: { start: 0, end: 0 }, text: 'x', isComplete: true },
    ])
  })
})

describe('a note read', () => {
  it('is requested for by its path', async () => {
    asked.readNote.mockResolvedValue({ body: 'prose' })
    await core.read('Notes/Heat.md')
    expect(asked.readNote.mock.calls.at(-1)?.[0]).toEqual({ path: 'Notes/Heat.md' })
  })
})
