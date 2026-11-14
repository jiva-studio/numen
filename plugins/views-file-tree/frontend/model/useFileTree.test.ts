/**
 * Tests for createFileTree state and tree operations.
 */
import { describe, expect, it } from 'vitest'
import { createLostConnection } from '@numen/plugin-sdk/testing'
import { shallowRef, watch } from 'vue'
import type { Entry } from '../types'
import type { ListingRow } from '../types'
import {
  getFolderPath,
  getParentFolders,
  resolveDropFolder,
  ROOT,
  createFileTree,
  readSortBy,
  type SortBy,
} from './useFileTree'

const file = (path: string, overrides: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  isFolder: false,
  isNote: true,
  type: 'note',
  ...overrides,
})

const folder = (path: string): Entry => file(path, { isFolder: true, isNote: false })

const vault = (
  folders: Record<string, readonly Entry[]> = {
    [ROOT]: [folder('physics'), folder('notes'), file('Cover.png', { isNote: false })],
    physics: [folder('physics/heat'), file('physics/Entropy.md')],
    'physics/heat': [file('physics/heat/Kelvin.md')],
    notes: [file('notes/Today.md')],
  },
) => {
  const request: string[] = []
  return {
    requests: request,
    core: {
      list: async (folderPath: string) => {
        request.push(folderPath)
        return folders[folderPath] ?? []
      },
    },
    puts: (folderPath: string, entry: Entry) => {
      folders[folderPath] = [...(folders[folderPath] ?? []), entry]
    },
  }
}

const paths = (rows: readonly ListingRow[]): readonly string[] =>
  rows.flatMap((one) => [one.entry.path, ...paths(one.rows)])

describe('parent directory resolution', () => {
  it('is the root for a path at the top of the vault', () => {
    expect(getFolderPath('Cover.png')).toBe(ROOT)
  })

  it('is everything above the last segment', () => {
    expect(getFolderPath('physics/heat/Kelvin.md')).toBe('physics/heat')
  })
})

describe('the folders above a path', () => {
  it('are given from the root down, so each opens inside the one before it', () => {
    expect(getParentFolders('physics/heat/Kelvin.md')).toStrictEqual(['physics', 'physics/heat'])
  })

  it('are none for a path at the top of the vault', () => {
    expect(getParentFolders('Cover.png')).toStrictEqual([])
  })

  it('never name the path itself', () => {
    expect(getParentFolders('physics/heat')).not.toContain('physics/heat')
  })
})

describe('where a row release lands', () => {
  it('is the folder it went into', () => {
    expect(resolveDropFolder({ into: 'physics/heat' })).toBe('physics/heat')
  })

  it('is the folder holding the row it came before', () => {
    expect(resolveDropFolder({ before: 'physics/heat/Kelvin.md' })).toBe('physics/heat')
  })

  it('is the root for a row that came before one at the top of the vault', () => {
    expect(resolveDropFolder({ before: 'Cover.png' })).toBe(ROOT)
  })

  it('is the root for a drop into no row at all', () => {
    expect(resolveDropFolder({ into: null })).toBe(ROOT)
  })
})

describe('the tree as it opens', () => {
  it('draws what the root holds, in the order the vault gave it', async () => {
    const { core } = vault()
    const list = createFileTree(core)

    await list.folders.openFolder(ROOT)

    expect(paths(list.queries.rows.value)).toStrictEqual(['physics', 'notes', 'Cover.png'])
  })

  it('draws every file the vault holds, and not only its notes', async () => {
    const { core } = vault()
    const list = createFileTree(core)

    await list.folders.openFolder(ROOT)

    expect(paths(list.queries.rows.value)).toContain('Cover.png')
  })
})

describe('a folder opened', () => {
  it('is read, and its contents are drawn inside it', async () => {
    const { core } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    await list.folders.openFolder('physics')

    expect(paths(list.queries.rows.value)).toStrictEqual([
      'physics',
      'physics/heat',
      'physics/Entropy.md',
      'notes',
      'Cover.png',
    ])
  })

  it('is read again every time it opens', async () => {
    const { core, requests } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    await list.folders.openFolder('physics')
    list.folders.closeFolder('physics')

    await list.folders.openFolder('physics')

    expect(requests.filter((one) => one === 'physics')).toHaveLength(2)
  })
})

describe('a folder closed', () => {
  it('draws nothing under it', async () => {
    const { core } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    await list.folders.openFolder('physics')

    list.folders.closeFolder('physics')

    expect(paths(list.queries.rows.value)).toStrictEqual(['physics', 'notes', 'Cover.png'])
  })

  it('keeps the folders open inside it, so they are drawn again where it opens', async () => {
    const { core } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    await list.folders.openFolder('physics')
    await list.folders.openFolder('physics/heat')

    list.folders.closeFolder('physics')
    await list.folders.openFolder('physics')

    expect(paths(list.queries.rows.value)).toContain('physics/heat/Kelvin.md')
  })

  it('is never the root, which is the tree itself', async () => {
    const { core } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)

    list.folders.closeFolder(ROOT)

    expect(list.queries.rows.value).toHaveLength(3)
  })
})

describe('a change the vault reports', () => {
  it('reads parent directory of target path', async () => {
    const { core, requests } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    await list.folders.openFolder('physics')
    requests.length = 0

    await list.sync.refreshChanged(['physics/Entropy.md'])

    expect(requests).toStrictEqual(['physics'])
  })

  it('leaves every other open folder alone', async () => {
    const { core, requests } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    await list.folders.openFolder('physics')
    await list.folders.openFolder('notes')
    requests.length = 0

    await list.sync.refreshChanged(['physics/Entropy.md'])

    expect(requests).not.toContain('notes')
  })

  it('reads nothing for a path inside a folder that is closed', async () => {
    const { core, requests } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    requests.length = 0

    await list.sync.refreshChanged(['physics/heat/Kelvin.md'])

    expect(requests).toStrictEqual([])
  })

  it('reads the open folder above a folder it draws no row for', async () => {
    const { core, requests, puts } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    puts(ROOT, folder('trips'))
    requests.length = 0

    await list.sync.refreshChanged(['trips/Kyoto.md'])

    expect(requests).toStrictEqual([ROOT])
    expect(paths(list.queries.rows.value)).toContain('trips')
  })

  it('reads every open folder when it is empty at all', async () => {
    const { core, requests } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    await list.folders.openFolder('physics')
    requests.length = 0

    await list.sync.refreshChanged([])

    expect([...requests].sort()).toStrictEqual([ROOT, 'physics'])
  })

  it('reads the folder a file left and the folder it arrived in', async () => {
    const { core, requests } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    await list.folders.openFolder('physics')
    await list.folders.openFolder('notes')
    requests.length = 0

    await list.sync.refreshChanged([], [{ from: 'physics/Entropy.md', to: 'notes/Entropy.md' }])

    expect([...requests].sort()).toStrictEqual(['notes', 'physics'])
  })

  it('leaves each chosen row chosen at where it went', async () => {
    const { core } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    await list.folders.openFolder('physics')
    list.selectPaths(['physics/Entropy.md', 'Cover.png'])

    await list.sync.refreshChanged([], [{ from: 'physics/Entropy.md', to: 'notes/Entropy.md' }])

    expect(list.state.selectedPaths.value).toStrictEqual(['notes/Entropy.md', 'Cover.png'])
  })
})

describe('the window returning to the front', () => {
  it('reads every open folder, and turns up what nothing reported', async () => {
    const { core, puts } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    await list.folders.openFolder('physics')
    puts('physics', file('physics/Diagram.png', { isNote: false }))

    await list.sync.refresh()

    expect(paths(list.queries.rows.value)).toContain('physics/Diagram.png')
  })

  it('reads no folder that is closed', async () => {
    const { core, requests } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    requests.length = 0

    await list.sync.refresh()

    expect(requests).toStrictEqual([ROOT])
  })
})

describe('the tree walked down to a path', () => {
  it('opens each folder above it, from the root down', async () => {
    const { core, requests } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    requests.length = 0

    await list.revealPath('physics/heat/Kelvin.md')

    expect(requests).toStrictEqual(['physics', 'physics/heat'])
  })

  it('draws the path it was walked down to', async () => {
    const { core } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)

    await list.revealPath('physics/heat/Kelvin.md')

    expect(paths(list.queries.rows.value)).toContain('physics/heat/Kelvin.md')
  })

  it('leaves the path the whole of what is chosen', async () => {
    const { core } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    list.selectPaths(['Cover.png'])

    await list.revealPath('physics/heat/Kelvin.md')

    expect(list.state.selectedPaths.value).toStrictEqual(['physics/heat/Kelvin.md'])
  })
})

describe('a folder that could not be read', () => {
  it('is returned in the tab, and the tree remains as it was', async () => {
    const list = createFileTree({
      list: async () => {
        throw createLostConnection('the vault is not there')
      },
    })

    await list.folders.openFolder(ROOT)

    expect(list.state.errorMessage.value).toContain('Numen is not responding')
    expect(list.state.errorMessage.value).not.toContain('the vault is not there')
    expect(list.queries.rows.value).toStrictEqual([])
  })
})

describe('a tab that has closed', () => {
  it('draws nothing of what it was given after it closed', async () => {
    const { core } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)

    list.close()

    expect(list.queries.rows.value).toStrictEqual([])
  })

  it('keeps nothing a question answers with after it closed', async () => {
    const { core } = vault()
    const list = createFileTree(core)
    const pendingOpen = list.folders.openFolder(ROOT)
    list.close()

    await pendingOpen

    expect(list.queries.rows.value).toStrictEqual([])
  })
})

describe('the refresh of every open folder', () => {
  /** A tree with the root, physics and notes open, and a count of the times its rows were drawn again. */
  const openTree = async (folders?: Record<string, readonly Entry[]>) => {
    const one = vault(folders)
    const list = createFileTree(one.core)
    await list.folders.openFolder(ROOT)
    await list.folders.openFolder('physics')
    await list.folders.openFolder('notes')
    let redrawn = 0
    watch(list.queries.rows, () => redrawn++, { flush: 'sync' })
    return { ...one, list, redrawn: () => redrawn }
  }

  it('draws the rows once for all the folders it listed', async () => {
    const { list, redrawn, requests } = await openTree()
    requests.length = 0
    const rowsBefore = list.queries.rows.value

    await list.sync.refresh()

    expect(requests).toHaveLength(3)
    expect(redrawn()).toBe(0)
    expect(list.queries.rows.value).toBe(rowsBefore)
  })

  it('draws the rows once when several folders changed', async () => {
    const { list, redrawn, puts } = await openTree()
    puts(ROOT, file('Extra.md'))
    puts('notes', file('notes/Later.md'))

    await list.sync.refresh()

    expect(redrawn()).toBe(1)
    expect(paths(list.queries.rows.value)).toContain('Extra.md')
    expect(paths(list.queries.rows.value)).toContain('notes/Later.md')
  })

  it('replaces the entries of the folder that changed and keeps the others', async () => {
    const { list, puts } = await openTree()
    const physics = list.queries.getEntriesInFolder('physics')
    const root = list.queries.getEntriesInFolder(ROOT)
    puts('notes', file('notes/Later.md'))

    await list.sync.refresh()

    expect(list.queries.getEntriesInFolder('notes')).toHaveLength(2)
    expect(list.queries.getEntriesInFolder('physics')).toBe(physics)
    expect(list.queries.getEntriesInFolder(ROOT)).toBe(root)
  })

  it('keeps the entries of a folder that failed and takes the others', async () => {
    const one = vault()
    let isFailing = false
    const list = createFileTree({
      list: async (folderPath: string) => {
        if (isFailing && folderPath === 'physics') throw new Error('gone')
        return one.core.list(folderPath)
      },
    })
    await list.folders.openFolder(ROOT)
    await list.folders.openFolder('physics')
    await list.folders.openFolder('notes')
    const cachedEntries = list.queries.getEntriesInFolder('physics')
    one.puts('notes', file('notes/Later.md'))
    isFailing = true

    await list.sync.refresh()

    expect(list.queries.getEntriesInFolder('physics')).toBe(cachedEntries)
    expect(list.queries.getEntriesInFolder('notes')).toHaveLength(2)
    expect(list.state.errorMessage.value).not.toBe('')
  })
})

describe('a tree before anything is requested of it', () => {
  it('holds nothing chosen, nothing returned, and the root open', () => {
    const { core } = vault()
    const list = createFileTree(core)

    expect(list.state.selectedPaths.value).toStrictEqual([])
    expect(list.state.errorMessage.value).toBe('')
    expect(list.queries.openRows.value).toStrictEqual([ROOT])
    expect(list.queries.isFolderOpen(ROOT)).toBe(true)
    expect(list.queries.getEntriesInFolder('unloaded')).toStrictEqual([])
  })

  it('reads the root when every open folder is refreshed', async () => {
    const { core, requests } = vault()
    const list = createFileTree(core)

    await list.sync.refresh()

    expect(requests).toStrictEqual([ROOT])
  })
})

describe('a tab that has closed, requested for more', () => {
  it('leaves only the root open', async () => {
    const { core } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder('physics')

    list.close()

    expect(list.queries.openRows.value).toStrictEqual([ROOT])
    expect(list.queries.isFolderOpen(ROOT)).toBe(true)
  })

  it('opens no folder and reads none', async () => {
    const { core, requests } = vault()
    const list = createFileTree(core)
    list.close()

    await list.folders.openFolder('physics')

    expect(list.queries.isFolderOpen('physics')).toBe(false)
    expect(requests).toStrictEqual([])
  })

  it('chooses no path when it closed while the folders above it were opening', async () => {
    const folders: Record<string, readonly Entry[]> = {
      [ROOT]: [folder('physics')],
      physics: [file('physics/Entropy.md')],
    }
    const holder: { list?: ReturnType<typeof createFileTree> } = {}
    const list = createFileTree({
      list: async (folderPath: string) => {
        holder.list?.close()
        return folders[folderPath] ?? []
      },
    })
    holder.list = list

    await list.revealPath('physics/Entropy.md')

    expect(list.state.selectedPaths.value).toStrictEqual([])
  })
})

describe('the root folder closed', () => {
  it('stays open', async () => {
    const { core } = vault()
    const list = createFileTree(core)

    list.folders.closeFolder(ROOT)

    expect(list.queries.isFolderOpen(ROOT)).toBe(true)
    expect(list.queries.openRows.value).toStrictEqual([ROOT])
  })
})

describe('a folder toggled', () => {
  it('opens where it was shut, and reads it', async () => {
    const { core, requests } = vault()
    const list = createFileTree(core)

    await list.folders.toggleFolder('physics')

    expect(list.queries.isFolderOpen('physics')).toBe(true)
    expect(requests).toContain('physics')
  })

  it('shuts where it was open', async () => {
    const { core } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder('physics')

    await list.folders.toggleFolder('physics')

    expect(list.queries.isFolderOpen('physics')).toBe(false)
  })
})

describe('a folder that could not be read, then read', () => {
  it('stops being returned in the tab', async () => {
    let fails = true
    const list = createFileTree({
      list: async () => {
        if (fails) throw new Error('down')
        return []
      },
    })
    await list.folders.openFolder(ROOT)
    expect(list.state.errorMessage.value).not.toBe('')

    fails = false
    await list.sync.refresh()

    expect(list.state.errorMessage.value).toBe('')
  })
})

describe('a listing that changes in one field of one entry', () => {
  const first = file('a.md')
  const second = file('b.md')
  const changes: readonly [string, Partial<Entry>][] = [
    ['path', { path: 'c.md' }],
    ['name', { name: 'B' }],
    ['isFolder', { isFolder: true }],
    ['isNote', { isNote: false }],
    ['type', { type: 'deck' }],
  ]

  it.each(changes)('is drawn again where the %s changed', async (_field, overrides) => {
    let listing: readonly Entry[] = [first, second]
    const list = createFileTree({ list: async () => listing })
    await list.folders.openFolder(ROOT)
    const changed = { ...second, ...overrides }
    listing = [first, changed]

    await list.sync.refresh()

    expect(list.queries.getEntriesInFolder(ROOT)).toStrictEqual([first, changed])
  })
})

describe('the folder a change is drawn in', () => {
  it('is an open folder the listing above does not name', async () => {
    const { core, requests } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    await list.folders.openFolder('ghost')
    requests.length = 0

    await list.sync.refreshChanged(['ghost/Note.md'])

    expect(requests).toStrictEqual(['ghost'])
  })

  it('leaves the chosen rows as they were where nothing was renamed', async () => {
    const { core } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    list.selectPaths(['Cover.png'])
    const selected = list.state.selectedPaths.value

    await list.sync.refreshChanged(['Cover.png'])

    expect(list.state.selectedPaths.value).toBe(selected)
  })
})

describe('a change the vault reports without naming anything', () => {
  it('reads every open folder when called with no arguments', async () => {
    const { core, requests } = vault()
    const list = createFileTree(core)
    await list.folders.openFolder(ROOT)
    await list.folders.openFolder('physics')
    requests.length = 0

    await list.sync.refreshChanged()

    expect(requests.sort()).toStrictEqual([ROOT, 'physics'])
  })
})

describe('the order of the rows', () => {
  const dated = (path: string, modifiedAt?: number): Entry =>
    file(path, modifiedAt === undefined ? {} : { modifiedAt })
  const datedFolder = (path: string, modifiedAt?: number): Entry => ({
    ...folder(path),
    ...(modifiedAt === undefined ? {} : { modifiedAt }),
  })

  const createTree = (folders: Record<string, readonly Entry[]>, initial: SortBy = 'modified') => {
    const requests: string[] = []
    const sortBy = shallowRef<SortBy>(initial)
    const core = {
      list: async (folderPath: string) => {
        requests.push(folderPath)
        return folders[folderPath] ?? []
      },
    }
    return { list: createFileTree(core, { sortBy }), sortBy, requests, folders }
  }

  const root = (...entries: Entry[]) => ({ [ROOT]: entries })

  it('keeps the order of the listing when no sort is given', async () => {
    const { core } = vault(root(dated('b.md', 1), dated('a.md', 9), dated('c.md', 5)))
    const list = createFileTree(core)

    await list.folders.openFolder(ROOT)

    expect(paths(list.queries.rows.value)).toStrictEqual(['b.md', 'a.md', 'c.md'])
  })

  it('lists folders first and then files in the order of the listing when sorting by name', async () => {
    const { list } = createTree(
      root(
        datedFolder('alpha', 1),
        datedFolder('beta', 9),
        dated('a.md', 1),
        dated('b.md', 9),
        dated('c.md', 5),
      ),
      'name',
    )

    await list.folders.openFolder(ROOT)

    expect(paths(list.queries.rows.value)).toStrictEqual(['alpha', 'beta', 'a.md', 'b.md', 'c.md'])
  })

  it('lists the most recently modified file first when sorting by date', async () => {
    const { list } = createTree(root(dated('a.md', 100), dated('b.md', 300), dated('c.md', 200)))

    await list.folders.openFolder(ROOT)

    expect(paths(list.queries.rows.value)).toStrictEqual(['b.md', 'c.md', 'a.md'])
  })

  it('keeps folders before files when sorting by date', async () => {
    const { list } = createTree(
      root(
        datedFolder('old', 1),
        dated('new.md', 900),
        datedFolder('fresh', 500),
        dated('mid.md', 50),
      ),
    )

    await list.folders.openFolder(ROOT)

    expect(paths(list.queries.rows.value)).toStrictEqual(['fresh', 'old', 'new.md', 'mid.md'])
  })

  it('puts entries with no date after the dated ones, each in name order', async () => {
    const { list } = createTree(
      root(
        dated('a.md'),
        dated('b.md', 10),
        dated('c.md'),
        dated('d.md', 20),
        datedFolder('x-folder'),
        datedFolder('y-folder', 5),
      ),
    )

    await list.folders.openFolder(ROOT)

    expect(paths(list.queries.rows.value)).toStrictEqual([
      'y-folder',
      'x-folder',
      'd.md',
      'b.md',
      'a.md',
      'c.md',
    ])
  })

  it('keeps name order between entries with the same date', async () => {
    const { list } = createTree(
      root(dated('a.md', 7), dated('b.md', 9), dated('c.md', 7), dated('d.md', 9)),
    )

    await list.folders.openFolder(ROOT)

    expect(paths(list.queries.rows.value)).toStrictEqual(['b.md', 'd.md', 'a.md', 'c.md'])
  })

  it('sorts the children of an open folder by date as well', async () => {
    const { list } = createTree({
      [ROOT]: [datedFolder('docs', 1)],
      docs: [dated('docs/a.md', 1), datedFolder('docs/sub', 1), dated('docs/b.md', 2)],
    })
    await list.folders.openFolder(ROOT)

    await list.folders.openFolder('docs')

    expect(paths(list.queries.rows.value)).toStrictEqual([
      'docs',
      'docs/sub',
      'docs/b.md',
      'docs/a.md',
    ])
  })

  it('re-orders the rows when the sort changes, with its open folders and selection kept', async () => {
    const { list, sortBy, requests } = createTree(
      {
        [ROOT]: [datedFolder('docs', 1), dated('a.md', 1), dated('b.md', 2)],
        docs: [dated('docs/a.md', 1), dated('docs/b.md', 2)],
      },
      'name',
    )
    await list.folders.openFolder(ROOT)
    await list.folders.openFolder('docs')
    list.selectPaths(['docs/a.md'])
    const calls = requests.length
    const byName = ['docs', 'docs/a.md', 'docs/b.md', 'a.md', 'b.md']
    expect(paths(list.queries.rows.value)).toStrictEqual(byName)

    sortBy.value = 'modified'

    expect(paths(list.queries.rows.value)).toStrictEqual([
      'docs',
      'docs/b.md',
      'docs/a.md',
      'b.md',
      'a.md',
    ])
    expect(list.queries.isFolderOpen('docs')).toBe(true)
    expect(list.state.selectedPaths.value).toStrictEqual(['docs/a.md'])
    expect(requests).toHaveLength(calls)

    sortBy.value = 'name'

    expect(paths(list.queries.rows.value)).toStrictEqual(byName)
  })

  it('takes the new date of a changed file into the rows on the next listing', async () => {
    const { list, folders } = createTree(root(dated('a.md', 100), dated('b.md', 200)))
    await list.folders.openFolder(ROOT)
    expect(paths(list.queries.rows.value)).toStrictEqual(['b.md', 'a.md'])

    folders[ROOT] = [dated('a.md', 300), dated('b.md', 200)]
    await list.sync.refreshChanged(['a.md'])

    expect(paths(list.queries.rows.value)).toStrictEqual(['a.md', 'b.md'])
  })

  it('reads an unknown or missing sort setting as name order', () => {
    expect(readSortBy('modified')).toBe('modified')
    expect(readSortBy('name')).toBe('name')
    expect(readSortBy(undefined)).toBe('name')
    expect(readSortBy('created')).toBe('name')
    expect(readSortBy(3)).toBe('name')
  })
})
