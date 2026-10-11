/**
 * What the window asks of the application, and how it reads a refusal.
 *
 * Each call hands the client the message the schema names for it, and each
 * refusal comes back as a failure in the window's own words.
 */
import { describe, expect, it, vi } from 'vitest'
import { ErrorCode, VaultsErrorCode } from '@numen/protocol'
import { core, vaults } from './index'
import { WINDOW } from './clients'

const { asked } = vi.hoisted(() => {
  const asked = {
    createNote: vi.fn(),
    renameNote: vi.fn(),
    listHeadings: vi.fn(),
    resolveAddresses: vi.fn(),
    removeFile: vi.fn(),
    listFiles: vi.fn(),
    moveFile: vi.fn(),
    createFolder: vi.fn(),
    createURL: vi.fn(),
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
  return { asked }
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

describe('a note asked for', () => {
  it('is created at the path of its folder, and refused in the words of the error', async () => {
    asked.createNote.mockResolvedValue({ path: 'Notes/Heat.md' })
    const created = await core.create({ title: 'Heat', folder: 'Notes', links: [] })
    expect(created).toEqual({ ok: true, value: { path: 'Notes/Heat.md' } })
    expect(asked.createNote.mock.calls.at(-1)?.[0].path).toBe('Notes')

    asked.createNote.mockResolvedValue({ error: ErrorCode.OCCUPIED })
    const refused = await core.create({ title: 'Heat', folder: 'Notes', links: [] })
    expect(refused).toEqual({ ok: false, error: 'occupied' })
  })

  it('is renamed by the path it has, and a stale answer is a change under the window', async () => {
    asked.renameNote.mockResolvedValue({ path: 'B.md', title: 'B', by: 1 })
    const renamed = await core.rename('A.md', 'B')
    expect(asked.renameNote.mock.calls.at(-1)?.[0]).toEqual({ path: 'A.md', title: 'B' })
    expect(renamed).toEqual({
      ok: true,
      value: { path: 'B.md', title: 'B', hasFrontmatter: true, moved: null },
    })

    asked.renameNote.mockResolvedValue({ error: ErrorCode.STALE })
    expect(await core.rename('A.md', 'B')).toEqual({ ok: false, error: 'changed' })

    asked.renameNote.mockResolvedValue({ error: ErrorCode.OCCUPIED })
    expect(await core.rename('A.md', 'B')).toEqual({ ok: false, error: 'occupied' })
  })

  it('has its headings asked of every path given', async () => {
    asked.listHeadings.mockResolvedValue({ headings: [] })
    await core.headings(['a.md', 'b.md'])
    expect(asked.listHeadings.mock.calls.at(-1)?.[0]).toEqual({ paths: ['a.md', 'b.md'] })
  })

  it('has its addresses resolved from the note that wrote them, less the ones that cross', async () => {
    asked.resolveAddresses.mockResolvedValue({
      resolved: [
        { written: 'B', path: 'b.md', isCrossed: false },
        { written: 'C', path: 'c.md', isCrossed: true },
      ],
    })
    const resolved = await core.resolve('a.md', ['B', 'C'])
    expect(asked.resolveAddresses.mock.calls.at(-1)?.[0]).toEqual({
      from: 'a.md',
      written: ['B', 'C'],
    })
    expect([...resolved]).toEqual([['B', 'b.md']])
  })
})

describe('a file asked for', () => {
  it('is removed to the trash unless it is to be destroyed, and refused in words', async () => {
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

  it('is moved from one path to another, and refused in words', async () => {
    asked.moveFile.mockResolvedValue({})
    await core.move('a.md', 'b.md')
    expect(asked.moveFile.mock.calls.at(-1)?.[0]).toEqual({ from: 'a.md', to: 'b.md' })

    asked.moveFile.mockResolvedValue({ error: ErrorCode.OCCUPIED })
    expect(await core.move('a.md', 'b.md')).toEqual({ ok: false, error: 'occupied' })
  })

  it('has a folder created at its path', async () => {
    asked.createFolder.mockResolvedValue({})
    await core.createFolder('Notes/New')
    expect(asked.createFolder.mock.calls.at(-1)?.[0]).toEqual({ path: 'Notes/New' })
  })

  it('has a url created in a folder, and refused in words', async () => {
    asked.createURL.mockResolvedValue({ path: 'Notes/page.md' })
    await core.createUrl('https://example.com', 'Notes')
    expect(asked.createURL.mock.calls.at(-1)?.[0]).toEqual({
      url: 'https://example.com',
      path: 'Notes',
    })

    asked.createURL.mockResolvedValue({ error: ErrorCode.UNNAMEABLE })
    expect(await core.createUrl('https://example.com', 'Notes')).toEqual({
      ok: false,
      error: 'unnameable',
    })
  })

  it('has its kinds asked of every path given', async () => {
    asked.listFileKinds.mockResolvedValue({ kinds: [] })
    await core.fileKinds(['a.epub', 'b.pdf'])
    expect(asked.listFileKinds.mock.calls.at(-1)?.[0]).toEqual({ paths: ['a.epub', 'b.pdf'] })
  })
})

describe('a vault asked for', () => {
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

  it('is refused in the words of the error', async () => {
    asked.addVault.mockResolvedValue({ error: VaultsErrorCode.NAME_TAKEN })
    expect(await vaults.add('/v', 'Name')).toEqual({ ok: false, error: 'nameTaken' })
  })
})
