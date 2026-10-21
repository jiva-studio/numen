/**
 * Tests for useFileTree state and tree operations.
 */
import { describe, expect, it } from 'vitest'
import { watch } from 'vue'
import type { Entry } from '../types'
import type { ListingRow } from '../types'
import {
  generateUniqueName,
  getFolderPath,
  getParentFolders,
  resolveDropFolder,
  ROOT,
  useFileTree,
} from './useFileTree'

/** One row of a listing, under the folder it sits in. */
const file = (path: string, over: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  isFolder: false,
  kind: 'note',
  type: 'note',
  ...over,
})

const folder = (path: string): Entry => file(path, { isFolder: true, kind: 'other' })

/**
 * A vault of two folders and a picture nothing holds a source for, and a count
 * of what every folder was asked.
 */
const vault = (
  folders: Record<string, readonly Entry[]> = {
    [ROOT]: [folder('physics'), folder('notes'), file('Cover.png', { kind: 'other' })],
    physics: [folder('physics/heat'), file('physics/Entropy.md')],
    'physics/heat': [file('physics/heat/Kelvin.md')],
    notes: [file('notes/Today.md')],
  },
) => {
  const request: string[] = []
  return {
    requests: request,
    core: {
      list: async (at: string) => {
        request.push(at)
        return folders[at] ?? []
      },
    },
    puts: (at: string, entry: Entry) => {
      folders[at] = [...(folders[at] ?? []), entry]
    },
  }
}

/** The paths of the rows drawn, in the order they are drawn, one line each. */
const paths = (rows: readonly ListingRow[]): readonly string[] =>
  rows.flatMap((one) => [one.entry.path, ...paths(one.rows)])

describe('the folder a path sits in', () => {
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

describe('a name nothing in a folder carries', () => {
  it('is the word itself where nothing is called that', () => {
    expect(generateUniqueName(['physics'], 'New folder')).toBe('New folder')
  })

  it('is the word and a count where it is taken', () => {
    expect(generateUniqueName(['New folder'], 'New folder')).toBe('New folder 2')
  })

  it('counts past every one that is taken', () => {
    expect(generateUniqueName(['New folder', 'New folder 2'], 'New folder')).toBe('New folder 3')
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
    const list = useFileTree(core)

    await list.openFolder(ROOT)

    expect(paths(list.rows.value)).toStrictEqual(['physics', 'notes', 'Cover.png'])
  })

  it('draws every file the vault holds, and not only its notes', async () => {
    const { core } = vault()
    const list = useFileTree(core)

    await list.openFolder(ROOT)

    expect(paths(list.rows.value)).toContain('Cover.png')
  })
})

describe('a folder opened', () => {
  it('is read, and what it holds is drawn inside it', async () => {
    const { core } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')

    expect(paths(list.rows.value)).toStrictEqual([
      'physics',
      'physics/heat',
      'physics/Entropy.md',
      'notes',
      'Cover.png',
    ])
  })

  it('is read again every time it opens', async () => {
    const { core, requests } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    list.closeFolder('physics')

    await list.openFolder('physics')

    expect(requests.filter((one) => one === 'physics')).toHaveLength(2)
  })
})

describe('a folder closed', () => {
  it('draws nothing under it', async () => {
    const { core } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')

    list.closeFolder('physics')

    expect(paths(list.rows.value)).toStrictEqual(['physics', 'notes', 'Cover.png'])
  })

  it('keeps the folders open inside it, so they are drawn again where it opens', async () => {
    const { core } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    await list.openFolder('physics/heat')

    list.closeFolder('physics')
    await list.openFolder('physics')

    expect(paths(list.rows.value)).toContain('physics/heat/Kelvin.md')
  })

  it('is never the root, which is the tree itself', async () => {
    const { core } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)

    list.closeFolder(ROOT)

    expect(list.rows.value).toHaveLength(3)
  })
})

describe('a change the vault reports', () => {
  it('reads the folder the path it names sits in', async () => {
    const { core, requests } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    requests.length = 0

    await list.refreshChanged(['physics/Entropy.md'])

    expect(requests).toStrictEqual(['physics'])
  })

  it('leaves every other open folder alone', async () => {
    const { core, requests } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    await list.openFolder('notes')
    requests.length = 0

    await list.refreshChanged(['physics/Entropy.md'])

    expect(requests).not.toContain('notes')
  })

  it('reads nothing for a path inside a folder that is closed', async () => {
    const { core, requests } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    requests.length = 0

    await list.refreshChanged(['physics/heat/Kelvin.md'])

    expect(requests).toStrictEqual([])
  })

  it('reads the open folder above a folder it draws no row for', async () => {
    const { core, requests, puts } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    puts(ROOT, folder('trips'))
    requests.length = 0

    await list.refreshChanged(['trips/Kyoto.md'])

    expect(requests).toStrictEqual([ROOT])
    expect(paths(list.rows.value)).toContain('trips')
  })

  it('reads every open folder when it is empty at all', async () => {
    const { core, requests } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    requests.length = 0

    await list.refreshChanged([])

    expect([...requests].sort()).toStrictEqual([ROOT, 'physics'])
  })

  it('reads the folder a file left and the folder it arrived in', async () => {
    const { core, requests } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    await list.openFolder('notes')
    requests.length = 0

    await list.refreshChanged([], [{ from: 'physics/Entropy.md', to: 'notes/Entropy.md' }])

    expect([...requests].sort()).toStrictEqual(['notes', 'physics'])
  })

  it('leaves each chosen row chosen at where it went', async () => {
    const { core } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    list.selectPaths(['physics/Entropy.md', 'Cover.png'])

    await list.refreshChanged([], [{ from: 'physics/Entropy.md', to: 'notes/Entropy.md' }])

    expect(list.selectedPaths.value).toStrictEqual(['notes/Entropy.md', 'Cover.png'])
  })
})

describe('the window coming back to the front', () => {
  it('reads every open folder, and turns up what nothing reported', async () => {
    const { core, puts } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    puts('physics', file('physics/Diagram.png', { kind: 'other' }))

    await list.refresh()

    expect(paths(list.rows.value)).toContain('physics/Diagram.png')
  })

  it('reads no folder that is closed', async () => {
    const { core, requests } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    requests.length = 0

    await list.refresh()

    expect(requests).toStrictEqual([ROOT])
  })
})

describe('the tree walked down to a path', () => {
  it('opens each folder above it, from the root down', async () => {
    const { core, requests } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    requests.length = 0

    await list.revealPath('physics/heat/Kelvin.md')

    expect(requests).toStrictEqual(['physics', 'physics/heat'])
  })

  it('draws the path it was walked down to', async () => {
    const { core } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)

    await list.revealPath('physics/heat/Kelvin.md')

    expect(paths(list.rows.value)).toContain('physics/heat/Kelvin.md')
  })

  it('leaves the path the whole of what is chosen', async () => {
    const { core } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    list.selectPaths(['Cover.png'])

    await list.revealPath('physics/heat/Kelvin.md')

    expect(list.selectedPaths.value).toStrictEqual(['physics/heat/Kelvin.md'])
  })
})

describe('a folder that could not be read', () => {
  it('is returned in the tab, and the tree remains as it was', async () => {
    const list = useFileTree({
      list: async () => {
        throw new Error('the vault is not there')
      },
    })

    await list.openFolder(ROOT)

    expect(list.errorMessage.value).toContain('numen did not answer')
    expect(list.errorMessage.value).not.toContain('the vault is not there')
    expect(list.rows.value).toStrictEqual([])
  })
})

describe('a tab that has closed', () => {
  it('draws nothing of what it was given after it closed', async () => {
    const { core } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)

    list.close()

    expect(list.rows.value).toStrictEqual([])
  })

  it('keeps nothing a question answers with after it closed', async () => {
    const { core } = vault()
    const list = useFileTree(core)
    const pendingOpen = list.openFolder(ROOT)
    list.close()

    await pendingOpen

    expect(list.rows.value).toStrictEqual([])
  })
})

describe('a name nothing in a folder is filed under', () => {
  it('counts past what the folder already holds', async () => {
    const { core } = vault({ [ROOT]: [folder('New folder')] })
    const list = useFileTree(core)
    await list.openFolder(ROOT)

    expect(list.getUniqueNameInFolder(ROOT, 'New folder')).toBe('New folder 2')
  })
})

describe('the refresh of every open folder', () => {
  /** A tree with the root, physics and notes open, and a count of the times its rows were drawn again. */
  const openTree = async (folders?: Record<string, readonly Entry[]>) => {
    const one = vault(folders)
    const list = useFileTree(one.core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    await list.openFolder('notes')
    let redrawn = 0
    watch(list.rows, () => redrawn++, { flush: 'sync' })
    return { ...one, list, redrawn: () => redrawn }
  }

  it('draws the rows once for all the folders it listed', async () => {
    const { list, redrawn, requests } = await openTree()
    requests.length = 0
    const rowsBefore = list.rows.value

    await list.refresh()

    expect(requests).toHaveLength(3)
    expect(redrawn()).toBe(0)
    expect(list.rows.value).toBe(rowsBefore)
  })

  it('draws the rows once when several folders changed', async () => {
    const { list, redrawn, puts } = await openTree()
    puts(ROOT, file('Extra.md'))
    puts('notes', file('notes/Later.md'))

    await list.refresh()

    expect(redrawn()).toBe(1)
    expect(paths(list.rows.value)).toContain('Extra.md')
    expect(paths(list.rows.value)).toContain('notes/Later.md')
  })

  it('replaces the entries of the folder that changed and keeps the others', async () => {
    const { list, puts } = await openTree()
    const physics = list.getEntriesInFolder('physics')
    const root = list.getEntriesInFolder(ROOT)
    puts('notes', file('notes/Later.md'))

    await list.refresh()

    expect(list.getEntriesInFolder('notes')).toHaveLength(2)
    expect(list.getEntriesInFolder('physics')).toBe(physics)
    expect(list.getEntriesInFolder(ROOT)).toBe(root)
  })

  it('keeps the entries of a folder that failed and takes the others', async () => {
    const one = vault()
    let isFailing = false
    const list = useFileTree({
      list: async (at: string) => {
        if (isFailing && at === 'physics') throw new Error('gone')
        return one.core.list(at)
      },
    })
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    await list.openFolder('notes')
    const cachedEntries = list.getEntriesInFolder('physics')
    one.puts('notes', file('notes/Later.md'))
    isFailing = true

    await list.refresh()

    expect(list.getEntriesInFolder('physics')).toBe(cachedEntries)
    expect(list.getEntriesInFolder('notes')).toHaveLength(2)
    expect(list.errorMessage.value).not.toBe('')
  })
})

describe('a tree before anything is requested of it', () => {
  it('holds nothing chosen, nothing returned, and the root open', () => {
    const { core } = vault()
    const list = useFileTree(core)

    expect(list.selectedPaths.value).toStrictEqual([])
    expect(list.errorMessage.value).toBe('')
    expect(list.openRows.value).toStrictEqual([ROOT])
    expect(list.isFolderOpen(ROOT)).toBe(true)
    expect(list.getEntriesInFolder('unloaded')).toStrictEqual([])
  })

  it('reads the root when every open folder is refreshed', async () => {
    const { core, requests } = vault()
    const list = useFileTree(core)

    await list.refresh()

    expect(requests).toStrictEqual([ROOT])
  })
})

describe('a tab that has closed, requested for more', () => {
  it('leaves only the root open', async () => {
    const { core } = vault()
    const list = useFileTree(core)
    await list.openFolder('physics')

    list.close()

    expect(list.openRows.value).toStrictEqual([ROOT])
    expect(list.isFolderOpen(ROOT)).toBe(true)
  })

  it('opens no folder and reads none', async () => {
    const { core, requests } = vault()
    const list = useFileTree(core)
    list.close()

    await list.openFolder('physics')

    expect(list.isFolderOpen('physics')).toBe(false)
    expect(requests).toStrictEqual([])
  })

  it('chooses no path when it closed while the folders above it were opening', async () => {
    const folders: Record<string, readonly Entry[]> = {
      [ROOT]: [folder('physics')],
      physics: [file('physics/Entropy.md')],
    }
    const holder: { list?: ReturnType<typeof useFileTree> } = {}
    const list = useFileTree({
      list: async (at: string) => {
        holder.list?.close()
        return folders[at] ?? []
      },
    })
    holder.list = list

    await list.revealPath('physics/Entropy.md')

    expect(list.selectedPaths.value).toStrictEqual([])
  })
})

describe('the root folder closed', () => {
  it('stays open', async () => {
    const { core } = vault()
    const list = useFileTree(core)

    list.closeFolder(ROOT)

    expect(list.isFolderOpen(ROOT)).toBe(true)
    expect(list.openRows.value).toStrictEqual([ROOT])
  })
})

describe('a folder toggled', () => {
  it('opens where it was shut, and reads it', async () => {
    const { core, requests } = vault()
    const list = useFileTree(core)

    list.toggleFolder('physics')
    await Promise.resolve()

    expect(list.isFolderOpen('physics')).toBe(true)
    expect(requests).toContain('physics')
  })

  it('shuts where it was open', async () => {
    const { core } = vault()
    const list = useFileTree(core)
    await list.openFolder('physics')

    list.toggleFolder('physics')

    expect(list.isFolderOpen('physics')).toBe(false)
  })
})

describe('a folder that could not be read, then read', () => {
  it('stops being returned in the tab', async () => {
    let fails = true
    const list = useFileTree({
      list: async () => {
        if (fails) throw new Error('down')
        return []
      },
    })
    await list.openFolder(ROOT)
    expect(list.errorMessage.value).not.toBe('')

    fails = false
    await list.refresh()

    expect(list.errorMessage.value).toBe('')
  })
})

describe('a listing that changes in one field of one entry', () => {
  const first = file('a.md')
  const second = file('b.md')
  const changes: readonly [string, Partial<Entry>][] = [
    ['path', { path: 'c.md' }],
    ['name', { name: 'B' }],
    ['isFolder', { isFolder: true }],
    ['kind', { kind: 'book' }],
    ['type', { type: 'deck' }],
  ]

  it.each(changes)('is drawn again where the %s changed', async (_field, over) => {
    let listing: readonly Entry[] = [first, second]
    const list = useFileTree({ list: async () => listing })
    await list.openFolder(ROOT)
    const changed = { ...second, ...over }
    listing = [first, changed]

    await list.refresh()

    expect(list.getEntriesInFolder(ROOT)).toStrictEqual([first, changed])
  })
})

describe('the folder a change is drawn in', () => {
  it('is an open folder the listing above does not name', async () => {
    const { core, requests } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('ghost')
    requests.length = 0

    await list.refreshChanged(['ghost/Note.md'])

    expect(requests).toStrictEqual(['ghost'])
  })

  it('leaves the chosen rows as they were where nothing was renamed', async () => {
    const { core } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    list.selectPaths(['Cover.png'])
    const selected = list.selectedPaths.value

    await list.refreshChanged(['Cover.png'])

    expect(list.selectedPaths.value).toBe(selected)
  })
})

describe('a change the vault reports without naming anything', () => {
  it('reads every open folder when called with no arguments', async () => {
    const { core, requests } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    requests.length = 0

    await list.refreshChanged()

    expect(requests.sort()).toStrictEqual([ROOT, 'physics'])
  })
})
