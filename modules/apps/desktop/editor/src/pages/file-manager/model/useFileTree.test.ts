/**
 * Tests for useFileTree state and tree operations.
 */
import { describe, expect, it } from 'vitest'
import type { Entry } from '@/entities/file'
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
  folder: false,
  kind: 'note',
  type: 'note',
  ...over,
})

const folder = (path: string): Entry => file(path, { folder: true, kind: 'other' })

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
  const asked: string[] = []
  return {
    asked,
    core: {
      list: async (at: string) => {
        asked.push(at)
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

describe('where a row let go of lands', () => {
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
    const { core, asked } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    list.closeFolder('physics')

    await list.openFolder('physics')

    expect(asked.filter((one) => one === 'physics')).toHaveLength(2)
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
    const { core, asked } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    asked.length = 0

    await list.refreshChanged(['physics/Entropy.md'])

    expect(asked).toStrictEqual(['physics'])
  })

  it('leaves every other open folder alone', async () => {
    const { core, asked } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    await list.openFolder('notes')
    asked.length = 0

    await list.refreshChanged(['physics/Entropy.md'])

    expect(asked).not.toContain('notes')
  })

  it('reads nothing for a path inside a folder that is closed', async () => {
    const { core, asked } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    asked.length = 0

    await list.refreshChanged(['physics/heat/Kelvin.md'])

    expect(asked).toStrictEqual([])
  })

  it('reads the open folder above a folder it draws no row for', async () => {
    const { core, asked, puts } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    puts(ROOT, folder('trips'))
    asked.length = 0

    await list.refreshChanged(['trips/Kyoto.md'])

    expect(asked).toStrictEqual([ROOT])
    expect(paths(list.rows.value)).toContain('trips')
  })

  it('reads every open folder when it names nothing at all', async () => {
    const { core, asked } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    asked.length = 0

    await list.refreshChanged([])

    expect([...asked].sort()).toStrictEqual([ROOT, 'physics'])
  })

  it('reads the folder a file left and the folder it arrived in', async () => {
    const { core, asked } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    await list.openFolder('physics')
    await list.openFolder('notes')
    asked.length = 0

    await list.refreshChanged([], [{ from: 'physics/Entropy.md', to: 'notes/Entropy.md' }])

    expect([...asked].sort()).toStrictEqual(['notes', 'physics'])
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
    const { core, asked } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    asked.length = 0

    await list.refresh()

    expect(asked).toStrictEqual([ROOT])
  })
})

describe('the tree walked down to a path', () => {
  it('opens each folder above it, from the root down', async () => {
    const { core, asked } = vault()
    const list = useFileTree(core)
    await list.openFolder(ROOT)
    asked.length = 0

    await list.revealPath('physics/heat/Kelvin.md')

    expect(asked).toStrictEqual(['physics', 'physics/heat'])
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
  it('is said in the tab, and the tree stands as it was', async () => {
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
    const asking = list.openFolder(ROOT)
    list.close()

    await asking

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
