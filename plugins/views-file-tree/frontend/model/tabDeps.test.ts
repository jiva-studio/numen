/**
 * What a files tab requires of the window, as satisfied by the host's workspace
 * and the vault's files.
 */
import { describe, expect, it } from 'vitest'
import { SLOT } from '@numen/plugin-sdk'
import { createHost, ROW_MENU } from '../fixtures/host'
import { createFilesTabDeps } from './tabDeps'
import type { Entry, FilesTabState } from '../types'

const entry = (overrides: Partial<Entry> = {}): Entry => ({
  path: 'a.md',
  name: 'a.md',
  isFolder: false,
  isNote: true,
  type: 'note',
  ...overrides,
})

const createTabState = (one: Entry | null) =>
  ({ list: { queries: { getEntryAt: () => one } } }) as unknown as FilesTabState

describe('a file a files tab opens', () => {
  it('goes to the tabs linked to it where it is a note one of them takes', () => {
    const { host, recorded } = createHost({ linked: { files: true } })

    createFilesTabDeps(host, 'files').window.openEntry(entry())

    expect(recorded.synced).toStrictEqual([['files', 'a.md', 'a']])
    expect(recorded.opened).toStrictEqual([])
  })

  it('opens in the pane the person is in where no linked tab takes it, or it is no note', () => {
    const { host, recorded } = createHost({ linked: { files: true } })
    const deps = createFilesTabDeps(host, 'files')

    deps.window.openEntry(entry({ path: 'Heat.pdf', name: 'Heat.pdf', isNote: false }))
    deps.window.openEntry(entry({ path: 'Untitled', name: '' }))

    expect(recorded.opened).toStrictEqual([
      ['Heat.pdf', { title: 'Heat.pdf', placement: 'current' }],
    ])
    expect(recorded.synced).toStrictEqual([['files', 'Untitled', '']])
  })

  it('opens under its path where it has no name and no linked tab takes it', () => {
    const { host, recorded } = createHost()

    createFilesTabDeps(host, 'files').window.openEntry(entry({ path: 'Untitled', name: '' }))

    expect(recorded.opened).toStrictEqual([
      ['Untitled', { title: 'Untitled', placement: 'current' }],
    ])
  })
})

describe('a path a files tab selects', () => {
  it('moves the tabs linked to it, a note by its kind or by its path', () => {
    const { host, recorded } = createHost()
    const deps = createFilesTabDeps(host, 'files')

    deps.window.onSelectPath?.('a.md', createTabState(entry()))
    deps.window.onSelectPath?.(
      'b.md',
      createTabState(entry({ path: 'b.md', name: 'b', isNote: false })),
    )
    deps.window.onSelectPath?.(
      'c.txt',
      createTabState(entry({ path: 'c.txt', name: 'c', isNote: true })),
    )

    expect(recorded.synced).toStrictEqual([
      ['files', 'a.md', 'a'],
      ['files', 'b.md', 'b'],
      ['files', 'c.txt', 'c'],
    ])
  })

  it('moves nothing for a folder, a file that is no note, or a path the tree does not hold', () => {
    const { host, recorded } = createHost()
    const deps = createFilesTabDeps(host, 'files')

    deps.window.onSelectPath?.('notes', createTabState(entry({ path: 'notes', isFolder: true })))
    deps.window.onSelectPath?.(
      'Cover.png',
      createTabState(entry({ path: 'Cover.png', isNote: false })),
    )
    deps.window.onSelectPath?.('gone.md', createTabState(null))

    expect(recorded.synced).toStrictEqual([])
  })
})

describe('the menu of a row', () => {
  const rowTarget = { path: 'a.md', title: 'a.md', isFolder: false, others: ['b.md'], tab: '' }

  it('is the entries of the slot of the file tree', () => {
    const { host } = createHost()

    expect(createFilesTabDeps(host, 'files').menu.getMenuEntries(rowTarget)).toStrictEqual(ROW_MENU)
  })

  it('runs the command of an entry through the commands of the window', async () => {
    const { host, commands } = createHost()

    await createFilesTabDeps(host, 'files').menu.runCommand('app:file.remove', rowTarget)

    expect(commands.getRuns()).toStrictEqual([
      { command: 'app:file.remove', target: rowTarget, itemId: undefined },
    ])
  })

  it('runs the command of an entry with the values of the entry', async () => {
    const { host, commands } = createHost()

    await createFilesTabDeps(host, 'files').menu.runCommand('app:file.open', rowTarget, undefined, {
      placement: 'split',
    })

    expect(commands.getRuns()).toStrictEqual([
      {
        command: 'app:file.open',
        target: rowTarget,
        itemId: undefined,
        values: { placement: 'split' },
      },
    ])
  })

  it('is empty, and runs nothing, in a window that offers no slots and no commands', async () => {
    const { host } = createHost()
    const { slots: _slots, commands: _commands, ...withoutSlots } = host
    const deps = createFilesTabDeps(withoutSlots, 'files')

    expect(deps.menu.getMenuEntries(rowTarget)).toStrictEqual([])
    await expect(deps.menu.runCommand('app:file.remove', rowTarget)).resolves.toBe('')
  })

  it('reads the entries of no slot but the menu of the file tree', () => {
    const { host, slots } = createHost()
    slots.setEntries(SLOT.tabStripContext, [])

    expect(createFilesTabDeps(host, 'files').menu.getMenuEntries(rowTarget)).toHaveLength(
      ROW_MENU.length,
    )
  })
})

describe('what else a files tab asks of the window', () => {
  it('moves through the files of the vault', async () => {
    const { host, recorded } = createHost()
    const deps = createFilesTabDeps(host, 'files')

    await deps.transfer.movePath('a.md', 'dir/a.md')

    expect(recorded.made).toStrictEqual(['move a.md dir/a.md'])
  })

  it('shares what is dragged, says what failed and draws through the window', () => {
    const { host, recorded } = createHost()
    const deps = createFilesTabDeps(host, 'files')

    deps.window.setDraggedPaths(['a.md'])
    deps.window.showError('taken')

    expect(recorded.dragged).toStrictEqual([['a.md']])
    expect(recorded.errors).toStrictEqual(['taken'])
    expect(deps.window.getEntryIcon(null, false)).toBeDefined()
  })
})

describe('data dropped from outside the window', () => {
  it('calls the drop host with the folder and data', async () => {
    const { host } = createHost()
    const dropped: string[] = []
    const deps = createFilesTabDeps(
      {
        ...host,
        drops: {
          canDrop: (types) => types.includes('text/uri-list'),
          drop: async (folder, data) => {
            dropped.push(`${folder} ${data['text/uri-list']}`)
            return [`${folder}/clip.md`]
          },
        },
      },
      'files',
    )

    expect(deps.transfer.canDropData(['text/uri-list'])).toBe(true)
    expect(
      await deps.transfer.dropData('talks', { 'text/uri-list': 'https://video.test/clip' }),
    ).toEqual(['talks/clip.md'])
    expect(dropped).toEqual(['talks https://video.test/clip'])
  })

  it('returns no paths without a drop host', async () => {
    const deps = createFilesTabDeps(createHost().host, 'files')
    expect(deps.transfer.canDropData(['text/uri-list'])).toBe(false)
    expect(
      await deps.transfer.dropData('', { 'text/uri-list': 'https://video.test/clip' }),
    ).toEqual([])
  })
})
