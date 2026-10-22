/**
 * What a files tab asks of the window, as the host's workspace and the files of
 * the vault answer it.
 */
import { describe, expect, it } from 'vitest'
import { createHost } from '../fixtures/host'
import { createFilesTabDeps } from './tabDeps'
import type { Entry, FilesTabState } from '../types'

const entry = (overrides: Partial<Entry> = {}): Entry => ({
  path: 'a.md',
  name: 'a.md',
  isFolder: false,
  kind: 'note',
  type: 'note',
  ...overrides,
})

const createTabState = (one: Entry | null) =>
  ({ list: { getEntryAt: () => one } }) as unknown as FilesTabState

describe('a file a files tab opens', () => {
  it('goes to the tabs linked to it where it is a note one of them takes', () => {
    const { host, recorded } = createHost({ linked: { files: true } })

    createFilesTabDeps(host, 'files').openEntry(entry())

    expect(recorded.synced).toStrictEqual([['files', 'a.md', 'a.md']])
    expect(recorded.opened).toStrictEqual([])
  })

  it('opens in the pane the person is in where no linked tab takes it, or it is no note', () => {
    const { host, recorded } = createHost({ linked: { files: true } })
    const deps = createFilesTabDeps(host, 'files')

    deps.openEntry(entry({ path: 'Heat.pdf', name: 'Heat.pdf', kind: 'book' }))
    deps.openEntry(entry({ path: 'Untitled', name: '' }))

    expect(recorded.opened).toStrictEqual([['Heat.pdf', { title: 'Heat.pdf', placement: 'here' }]])
    expect(recorded.synced).toStrictEqual([['files', 'Untitled', '']])
  })

  it('opens under its path where it has no name and no linked tab takes it', () => {
    const { host, recorded } = createHost()

    createFilesTabDeps(host, 'files').openEntry(entry({ path: 'Untitled', name: '' }))

    expect(recorded.opened).toStrictEqual([['Untitled', { title: 'Untitled', placement: 'here' }]])
  })
})

describe('a path a files tab selects', () => {
  it('moves the tabs linked to it, a note by its kind or by its path', () => {
    const { host, recorded } = createHost()
    const deps = createFilesTabDeps(host, 'files')

    deps.onSelectPath?.('a.md', createTabState(entry()))
    deps.onSelectPath?.('b.md', createTabState(entry({ path: 'b.md', name: 'b', kind: 'other' })))
    deps.onSelectPath?.('c.txt', createTabState(entry({ path: 'c.txt', name: 'c', kind: 'note' })))

    expect(recorded.synced).toStrictEqual([
      ['files', 'a.md', 'a.md'],
      ['files', 'b.md', 'b'],
      ['files', 'c.txt', 'c'],
    ])
  })

  it('moves nothing for a folder, a file that is no note, or a path the tree does not hold', () => {
    const { host, recorded } = createHost()
    const deps = createFilesTabDeps(host, 'files')

    deps.onSelectPath?.('notes', createTabState(entry({ path: 'notes', isFolder: true })))
    deps.onSelectPath?.('Cover.png', createTabState(entry({ path: 'Cover.png', kind: 'other' })))
    deps.onSelectPath?.('gone.md', createTabState(null))

    expect(recorded.synced).toStrictEqual([])
  })
})

describe('what else a files tab asks of the window', () => {
  it('runs a command over the paths and makes, moves and imports through the files of the vault', async () => {
    const { host, recorded } = createHost()
    const deps = createFilesTabDeps(host, 'files')

    deps.runCommand('remove', ['a.md', 'b.md'], { name: 'a', source: 'note' })
    await deps.movePath('a.md', 'dir/a.md')
    await deps.createFolder('dir')
    expect(await deps.createNote('dir/')).toBe('dir/note')
    expect(await deps.createDeck('dir/', 'D')).toBe('dir/D')
    expect(await deps.createStencil('dir/', 'S')).toBe('dir/S')
    expect(await deps.createPreset('dir/', 'P')).toBe('dir/P')
    expect(await deps.importUrl('dir/', 'https://a.test')).toBe('dir/https://a.test')
    expect(await deps.importUrls('dir/', ['https://b.test'])).toStrictEqual(['https://b.test'])

    expect(recorded.commands).toStrictEqual([
      ['remove', { paths: ['a.md', 'b.md'], name: 'a', source: 'note' }],
    ])
    expect(recorded.made).toStrictEqual([
      'move a.md dir/a.md',
      'folder dir',
      'note dir/',
      'deck dir/ D',
      'stencil dir/ S',
      'preset dir/ P',
      'url dir/ https://a.test',
      'urls dir/ https://b.test',
    ])
  })

  it('shares what is dragged, says what failed and draws through the window', () => {
    const { host, recorded } = createHost()
    const deps = createFilesTabDeps(host, 'files')

    deps.setDraggedPaths(['a.md'])
    deps.showError('taken')

    expect(recorded.dragged).toStrictEqual([['a.md']])
    expect(recorded.errors).toStrictEqual(['taken'])
    expect(
      deps.getPathCommands({ entry: entry(), isSeveral: true }).map((one) => one.id),
    ).toStrictEqual(['remove'])
    expect(deps.getCommandIcon('unknown')).toBeNull()
    expect(deps.getEntryIcon(null, false)).toBeDefined()
  })
})
