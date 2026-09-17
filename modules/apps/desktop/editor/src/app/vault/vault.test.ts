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

const asked = {
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
  createURL: vi.fn(),
  listFileKinds: vi.fn(),
  getSettings: vi.fn(),
  writeSettings: vi.fn(),
  readSettingsFile: vi.fn(),
  writeSettingsFile: vi.fn(),
  getVaultState: vi.fn(),
  getAgentState: vi.fn(),
  writeOpenTabs: vi.fn(),
  reportFlush: vi.fn(),
  listVaults: vi.fn(),
  getShownVault: vi.fn(),
  chooseFolder: vi.fn(),
  addVault: vi.fn(),
  renameVault: vi.fn(),
  removeVault: vi.fn(),
  openVault: vi.fn(),
}

const { core, vaults } = await import('./index')

describe('the file a save presents', () => {
  it('comes back as the file it was, through a path that holds spaces', async () => {
    const at = { path: 'mahabharata/The aggressor.md', size: 4096n, mtime: 1_700_000_000_123n }
    asked.readNote.mockResolvedValue({ body: 'prose', at })
    asked.writeNote.mockResolvedValue({})

    const read = await core.read(at.path)
    await core.write(at.path, 'prose and more', { prose: read.body, at: read.at ?? '' })

    expect(asked.writeNote.mock.calls[0]?.[0]).toEqual({
      path: at.path,
      body: 'prose and more',
      seen: { prose: 'prose', at },
    })
  })

  it('is left out of a save that presents nothing', async () => {
    asked.writeNote.mockResolvedValue({})

    await core.write('Heat.md', 'mine', null)

    expect(asked.writeNote.mock.calls.at(-1)?.[0]).toEqual({ path: 'Heat.md', body: 'mine' })
  })
})

describe('the role a link carries', () => {
  it('goes to the schema under the word the vault uses for it', async () => {
    asked.createNote.mockResolvedValue({ path: 'Entropy.md' })

    await core.create({
      title: 'Entropy',
      folder: '',
      links: [{ to: 'Ontology.md', role: 'jump', label: 'see also' }],
    })

    expect(asked.createNote.mock.calls[0]?.[0].links).toEqual([
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
      asked.writeLink.mockResolvedValue({})

      await core.join('Ontology.md', {
        to: 'Entropy.md',
        role: name.toLowerCase() as WindowRole,
      })

      expect(asked.writeLink.mock.calls.at(-1)?.[0].link).toEqual({
        to: 'Entropy.md',
        role: Role[name as keyof typeof Role],
        label: '',
      })
    }
  })
})

describe('files domain', () => {
  it('handles remove, list, move, createFolder, createUrl, and fileKinds', async () => {
    asked.removeFile.mockResolvedValue({ trashed: true, dangling: [] })
    const removed = await core.remove('test.md')
    expect(removed.trashed).toBe(true)

    asked.listFiles.mockResolvedValue({
      entries: [{ path: 'test.md', name: 'test.md', isFolder: false, kind: 1, type: 1 }],
    })
    const listing = await core.list('')
    expect(listing.length).toBe(1)

    asked.moveFile.mockResolvedValue({ moved: { from: 'a.md', to: 'b.md', repaired: [] } })
    const moved = await core.move('a.md', 'b.md')
    expect(moved.moved?.to).toBe('b.md')

    asked.createFolder.mockResolvedValue({})
    expect(await core.createFolder('folder')).toBeNull()

    asked.createURL.mockResolvedValue({ path: 'url.md' })
    const url = await core.createUrl('https://example.com', '')
    expect(url.path).toBe('url.md')

    asked.listFileKinds.mockResolvedValue({
      kinds: [{ path: 'book.epub', kind: 2, type: 1, format: 2 }],
    })
    const kinds = await core.fileKinds(['book.epub'])
    expect(kinds.get('book.epub')?.format).toBe('epub')
  })
})

describe('settings domain', () => {
  it('handles the sync, hanging, settings, settings-file and review settings', async () => {
    const writtenSettings = JSON.stringify({
      naming: { sync_title_and_filename: true },
      appearance: { hang_parts_under_a_node: true, parts_under_a_node: 3 },
      review: { day_starts: '04:00' },
    })
    asked.getSettings.mockResolvedValue({
      written: writtenSettings,
      models: [],
      partsUnderANodeBounds: { least: 1, most: 5 },
      latestDayStarts: '06:00',
      day: '2026-09-10',
      path: 'settings.json',
    })
    asked.writeSettings.mockResolvedValue({})

    expect(await core.getSyncEnabled()).toBe(true)
    expect(await core.setSyncEnabled(false)).toBeNull()

    const hung = await core.getHangingSettings()
    expect(hung.isHanging).toBe(true)
    expect(hung.parts).toBe(3)
    expect(await core.setHangingSettings(false, 2)).toBeNull()

    const config = await core.getSettings()
    expect(config.path).toBe('settings.json')
    await core.updateSettings([])

    asked.readSettingsFile.mockResolvedValue({ written: 'raw', path: 'settings.json' })
    asked.writeSettingsFile.mockResolvedValue({})
    const file = await core.getSettingsFile()
    expect(file.written).toBe('raw')
    const saved = await core.saveSettingsFile('raw2', null)
    expect(saved.changed).toBe(false)

    const rev = await core.getReviewSettings()
    expect(rev.starts).toBe('04:00')
    expect(await core.setReviewSettings('05:00')).toBeNull()
  })
})

describe('notes domain', () => {
  it('handles neighbourhood, the opening path, rename, headings, and resolve', async () => {
    asked.getNeighbourhood.mockResolvedValue({
      focus: { path: 'a.md', title: 'A' },
      related: [],
    })
    const neigh = await core.neighbourhood('a.md')
    expect(neigh.focus.path).toBe('a.md')

    asked.getOpeningNote.mockResolvedValue({ note: { path: 'opening.md' } })
    expect((await core.getInitialOpenPath())?.path).toBe('opening.md')

    asked.renameNote.mockResolvedValue({
      path: 'b.md',
      title: 'B',
      by: 1,
      moved: { path: 'b.md' },
    })
    const renamed = await core.rename('a.md', 'B')
    expect(renamed.path).toBe('b.md')

    asked.listHeadings.mockResolvedValue({
      headings: [{ path: 'a.md', headings: [{ text: 'H1', level: 1, line: 1 }] }],
    })
    const heads = await core.headings(['a.md'])
    expect(heads.get('a.md')?.[0]?.text).toBe('H1')

    asked.resolveAddresses.mockResolvedValue({
      resolved: [{ written: 'B', path: 'b.md', crossed: false }],
    })
    const res = await core.resolve('a.md', ['B'])
    expect(res.get('B')).toBe('b.md')
  })
})

describe('session domain', () => {
  it('handles state, agentUnreachable, writeOpenTabs, and reportFlush', async () => {
    asked.getVaultState.mockResolvedValue({
      id: 'v1',
      name: 'V1',
      path: '/vault',
      scan: { ready: true, error: '', unwatched: '' },
      coverage: { chunkCount: 10n, embeddedCount: 10n, embedding: false },
    })
    const st = await core.state()
    expect(st.name).toBe('V1')

    asked.getAgentState.mockResolvedValue({ unreachable: 'offline' })
    expect(await core.agentUnreachable()).toBe('offline')

    asked.writeOpenTabs.mockResolvedValue({})
    await core.writeOpenTabs({ tabs: [], front: '' })
    expect(asked.writeOpenTabs).toHaveBeenCalled()

    asked.reportFlush.mockResolvedValue({})
    await core.reportFlush('tok', 'written')
    expect(asked.reportFlush).toHaveBeenCalled()
  })
})

describe('vaults domain', () => {
  it('handles list, choose, add, rename, remove, open, and core.vaults', async () => {
    asked.listVaults.mockResolvedValue({ vaults: [{ id: 'v1', name: 'Vault 1' }] })
    asked.getShownVault.mockResolvedValue({ vault: 'v1' })

    const list = await vaults.list()
    expect(list.vaults.length).toBe(1)
    expect(list.showing).toBe('v1')

    const fromCore = await core.vaults()
    expect(fromCore.showing).toBe('v1')

    asked.chooseFolder.mockResolvedValue({ isChosen: true, path: '/v2' })
    expect(await vaults.choose('Pick')).toBe('/v2')

    asked.addVault.mockResolvedValue({ vault: { id: 'v2', name: 'V2' } })
    const added = await vaults.add('/v2', 'V2')
    expect(added.vault?.id).toBe('v2')

    asked.renameVault.mockResolvedValue({ vault: { id: 'v2', name: 'V2 renamed' } })
    const renamed = await vaults.rename('v2', 'V2 renamed')
    expect(renamed.vault?.name).toBe('V2 renamed')

    asked.removeVault.mockResolvedValue({})
    expect(await vaults.remove('v2', false)).toBeNull()

    asked.openVault.mockResolvedValue({})
    expect(await vaults.open('v1')).toBeNull()
  })
})
