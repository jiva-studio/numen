/**
 * The one thing this module decides on its own: how a file the window carries
 * about is written down and read back.
 *
 * Everything else here hands a message across unchanged, and the schema says
 * what those look like.
 */
import { describe, expect, it, vi } from 'vitest'
import { Role } from '@numen/protocol'
import type { Role as WindowRole } from '@/entities/note'

vi.stubGlobal('window', { location: { origin: 'http://numen.invalid' } })

vi.mock('@/shared/clients', () => ({
  vault: request,
  agentService: request,
  workspace: request,
  files: request,
  notes: request,
  search: request,
  vaultsService: request,
  settingsService: request,
  windowService: request,
}))

const request = {
  readNote: vi.fn(),
  writeNote: vi.fn(),
  createNote: vi.fn(),
  writeLink: vi.fn(),
  getNeighbourhood: vi.fn(),
  getOpeningNote: vi.fn(),
  renameNote: vi.fn(),
  listHeadings: vi.fn(),
  resolveAddresses: vi.fn(),
  removeFile: vi.fn(),
  listFiles: vi.fn(),
  moveFile: vi.fn(),
  createFolder: vi.fn(),
  listFileKinds: vi.fn(),
  getSettings: vi.fn(),
  writeSettings: vi.fn(),
  getVaultState: vi.fn(),
  getAgentState: vi.fn(),
  writeOpenTabs: vi.fn(),
  reportFlush: vi.fn(),
  listVaults: vi.fn(),
  getCurrentVault: vi.fn(),
  chooseFolder: vi.fn(),
  addVault: vi.fn(),
  renameVault: vi.fn(),
  removeVault: vi.fn(),
  openVault: vi.fn(),
}

const { core, vaults } = await import('./index')

describe('the file a save presents', () => {
  it('arrives back as the file it was, through a path that holds spaces', async () => {
    const noteFile = {
      path: 'mahabharata/The aggressor.md',
      size: 4096n,
      mtime: 1_700_000_000_123n,
    }
    request.readNote.mockResolvedValue({ body: 'prose', at: noteFile })
    request.writeNote.mockResolvedValue({})

    const readItem = await core.read(noteFile.path)
    if (!readItem.ok) throw new Error('the read failed')
    await core.write(noteFile.path, 'prose and more', {
      prose: readItem.value.body,
      stat: readItem.value.at ?? '',
    })

    expect(request.writeNote.mock.calls[0]?.[0]).toEqual({
      path: noteFile.path,
      body: 'prose and more',
      baseline: { prose: 'prose', at: noteFile },
    })
  })

  it('is left out of a save that presents nothing', async () => {
    request.writeNote.mockResolvedValue({})

    await core.write('Heat.md', 'mine', null)

    expect(request.writeNote.mock.calls.at(-1)?.[0]).toEqual({ path: 'Heat.md', body: 'mine' })
  })
})

describe('the role a link carries', () => {
  it('goes to the schema under the word the vault uses for it', async () => {
    request.createNote.mockResolvedValue({ path: 'Entropy.md' })

    await core.create({
      title: 'Entropy',
      folder: '',
      links: [{ to: 'Ontology.md', role: 'jump', label: 'see also' }],
    })

    expect(request.createNote.mock.calls[0]?.[0].links).toEqual([
      { to: 'Ontology.md', role: Role.JUMP, label: 'see also' },
    ])
  })

  // The roles are the schema's, so a role added there is one this asks for.
  it('is every role the schema names, on a link written into a note that is there', async () => {
    const carried = Object.keys(Role).filter(
      (name) => Number.isNaN(Number(name)) && name !== 'UNSPECIFIED',
    )
    expect(carried).toContain('ATTACHMENT')

    for (const name of carried) {
      request.writeLink.mockResolvedValue({})

      await core.join('Ontology.md', {
        to: 'Entropy.md',
        role: name.toLowerCase() as WindowRole,
      })

      expect(request.writeLink.mock.calls.at(-1)?.[0].link).toEqual({
        to: 'Entropy.md',
        role: Role[name as keyof typeof Role],
        label: '',
      })
    }
  })
})

describe('files domain', () => {
  it('handles remove, list, move, createFolder, and fileKinds', async () => {
    request.removeFile.mockResolvedValue({ trashed: true, dangling: [] })
    const removed = await core.remove('test.md')
    expect(removed.ok && removed.value.trashed).toBe(true)

    request.listFiles.mockResolvedValue({
      entries: [{ path: 'test.md', name: 'test.md', isFolder: false, kind: 1, type: 'note' }],
    })
    const listing = await core.list('')
    expect(listing.length).toBe(1)

    request.moveFile.mockResolvedValue({ moved: { from: 'a.md', to: 'b.md', repaired: [] } })
    const moved = await core.move('a.md', 'b.md')
    expect(moved.ok ? moved.value?.to : null).toBe('b.md')

    request.createFolder.mockResolvedValue({})
    expect(await core.createFolder('folder')).toBeNull()

    request.listFileKinds.mockResolvedValue({
      kinds: [
        { path: 'a.bin', isNote: false, type: '' },
        { path: 'a.md', isNote: true, type: 'deck' },
      ],
    })
    const kinds = await core.fileKinds(['a.bin', 'a.md'])
    expect(kinds.get('a.bin')).toStrictEqual({ isNote: false, type: 'note' })
    expect(kinds.get('a.md')).toStrictEqual({ isNote: true, type: 'deck' })
  })
})

describe('settings domain', () => {
  it('reads and writes the sync and settings', async () => {
    const writtenSettings = JSON.stringify({ naming: { sync_title_and_filename: true } })
    request.getSettings.mockResolvedValue({
      written: writtenSettings,
      models: [],
      path: 'settings.json',
    })
    request.writeSettings.mockResolvedValue({})

    expect(await core.getSyncEnabled()).toBe(true)
    expect(await core.setSyncEnabled(false)).toBeNull()

    const config = await core.getSettings()
    expect(config.path).toBe('settings.json')
    await core.updateSettings([])
  })
})

describe('notes domain', () => {
  it('handles neighbourhood, the opening path, rename, headings, and resolve', async () => {
    request.getNeighbourhood.mockResolvedValue({
      focus: { path: 'a.md', title: 'A' },
      related: [],
    })
    const neigh = await core.neighbourhood('a.md')
    expect(neigh.focus.path).toBe('a.md')

    request.getOpeningNote.mockResolvedValue({ note: { path: 'opening.md' } })
    expect((await core.getInitialOpenPath())?.path).toBe('opening.md')

    request.renameNote.mockResolvedValue({
      path: 'b.md',
      title: 'B',
      nameSource: 1,
      moved: { path: 'b.md' },
    })
    const renamedItems = await core.rename('a.md', 'B')
    expect(renamedItems.ok && renamedItems.value.path).toBe('b.md')

    request.listHeadings.mockResolvedValue({
      headings: [{ path: 'a.md', headings: [{ text: 'H1', level: 1, line: 1 }] }],
    })
    const heads = await core.headings(['a.md'])
    expect(heads.get('a.md')?.[0]?.text).toBe('H1')

    request.resolveAddresses.mockResolvedValue({
      found: [{ raw: '[[B]]', address: 'name://B', text: 'B' }],
      resolved: [{ written: 'name://B', path: 'b.md', isCrossed: false }],
    })
    const res = await core.resolveLinks('a.md', '[[B]]')
    expect(res[0]?.path).toBe('b.md')
  })
})

describe('session domain', () => {
  it('handles state, agentUnreachable, writeOpenTabs, and reportFlush', async () => {
    request.getVaultState.mockResolvedValue({
      id: 'v1',
      name: 'V1',
      path: '/vault',
      scan: { isReady: true, error: '', unwatched: '' },
      coverage: { chunkCount: 10n, embeddedCount: 10n, isEmbedding: false },
    })
    const st = await core.state()
    expect(st.name).toBe('V1')

    request.getAgentState.mockResolvedValue({ unreachable: 'offline' })
    expect(await core.agentUnreachable()).toBe('offline')

    request.writeOpenTabs.mockResolvedValue({})
    await core.writeOpenTabs({ tabs: [], activeId: '' })
    expect(request.writeOpenTabs).toHaveBeenCalled()

    request.reportFlush.mockResolvedValue({})
    await core.reportFlush('tok', 'written')
    expect(request.reportFlush).toHaveBeenCalled()
  })
})

describe('vaults domain', () => {
  it('handles list, choose, add, rename, remove, open, and core.vaults', async () => {
    request.listVaults.mockResolvedValue({ vaults: [{ id: 'v1', name: 'Vault 1' }] })
    request.getCurrentVault.mockResolvedValue({ vault: 'v1' })

    const list = await vaults.list()
    expect(list.vaults.length).toBe(1)
    expect(list.showing).toBe('v1')

    const fromCore = await core.vaults()
    expect(fromCore.showing).toBe('v1')

    request.chooseFolder.mockResolvedValue({ isChosen: true, path: '/v2' })
    expect(await vaults.choose('Pick')).toBe('/v2')

    request.addVault.mockResolvedValue({ vault: { id: 'v2', name: 'V2' } })
    const added = await vaults.add('/v2', 'V2')
    expect(added.ok ? added.value?.id : null).toBe('v2')

    request.renameVault.mockResolvedValue({ vault: { id: 'v2', name: 'V2 renamed' } })
    const renamedItems = await vaults.rename('v2', 'V2 renamed')
    expect(renamedItems.ok ? renamedItems.value?.name : null).toBe('V2 renamed')

    request.removeVault.mockResolvedValue({})
    expect(await vaults.remove('v2', false)).toBeNull()

    request.openVault.mockResolvedValue({})
    expect(await vaults.open('v1')).toBeNull()
  })
})
