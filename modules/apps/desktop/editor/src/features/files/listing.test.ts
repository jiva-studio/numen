/**
 * What a files tab knows about the vault, asked without a screen.
 *
 * The rules that matter are which folders are read again and which are left
 * alone: a tab that reads the whole vault at every keystroke looks exactly like
 * one that reads the right folder, and only the count of questions tells them
 * apart.
 */
import { describe, expect, it } from 'vitest'
import type { Entry } from '../../shared/core'
import { above, folderOf, freeName, landedIn, listing, ROOT, type ListingRow } from './listing'

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
  held: Record<string, readonly Entry[]> = {
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
        return held[at] ?? []
      },
    },
    /** A file put in a folder the way another program would put one there. */
    puts: (at: string, entry: Entry) => {
      held[at] = [...(held[at] ?? []), entry]
    },
  }
}

/** The paths of the rows drawn, in the order they are drawn, one line each. */
const paths = (rows: readonly ListingRow[]): readonly string[] =>
  rows.flatMap((one) => [one.entry.path, ...paths(one.rows)])

describe('the folder a path sits in', () => {
  it('is the root for a path at the top of the vault', () => {
    expect(folderOf('Cover.png')).toBe(ROOT)
  })

  it('is everything above the last segment', () => {
    expect(folderOf('physics/heat/Kelvin.md')).toBe('physics/heat')
  })
})

describe('the folders above a path', () => {
  it('are given from the root down, so each opens inside the one before it', () => {
    expect(above('physics/heat/Kelvin.md')).toStrictEqual(['physics', 'physics/heat'])
  })

  it('are none for a path at the top of the vault', () => {
    expect(above('Cover.png')).toStrictEqual([])
  })

  it('never name the path itself', () => {
    expect(above('physics/heat')).not.toContain('physics/heat')
  })
})

describe('a name nothing in a folder carries', () => {
  it('is the word itself where nothing is called that', () => {
    expect(freeName(['physics'], 'New folder')).toBe('New folder')
  })

  it('is the word and a count where it is taken', () => {
    expect(freeName(['New folder'], 'New folder')).toBe('New folder 2')
  })

  it('counts past every one that is taken', () => {
    expect(freeName(['New folder', 'New folder 2'], 'New folder')).toBe('New folder 3')
  })
})

describe('where a row let go of lands', () => {
  it('is the folder it went into', () => {
    expect(landedIn({ into: 'physics/heat' })).toBe('physics/heat')
  })

  it('is the folder holding the row it came before', () => {
    expect(landedIn({ before: 'physics/heat/Kelvin.md' })).toBe('physics/heat')
  })

  it('is the root for a row that came before one at the top of the vault', () => {
    expect(landedIn({ before: 'Cover.png' })).toBe(ROOT)
  })

  // A folder deep in the tree is brought out by letting it go on the tree's
  // own area, which names no row.
  it('is the root for a drop into no row at all', () => {
    expect(landedIn({ into: null })).toBe(ROOT)
  })
})

describe('the tree as it opens', () => {
  it('draws what the root holds, in the order the vault gave it', async () => {
    const { core } = vault()
    const list = listing(core)

    await list.opens(ROOT)

    expect(paths(list.rows.value)).toStrictEqual(['physics', 'notes', 'Cover.png'])
  })

  it('draws every file the vault holds, and not only its notes', async () => {
    const { core } = vault()
    const list = listing(core)

    await list.opens(ROOT)

    expect(paths(list.rows.value)).toContain('Cover.png')
  })
})

describe('a folder opened', () => {
  it('is read, and what it holds is drawn inside it', async () => {
    const { core } = vault()
    const list = listing(core)
    await list.opens(ROOT)

    await list.opens('physics')

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
    const list = listing(core)
    await list.opens(ROOT)
    await list.opens('physics')
    list.closes('physics')

    await list.opens('physics')

    expect(asked.filter((one) => one === 'physics')).toHaveLength(2)
  })
})

describe('a folder closed', () => {
  it('draws nothing under it', async () => {
    const { core } = vault()
    const list = listing(core)
    await list.opens(ROOT)
    await list.opens('physics')

    list.closes('physics')

    expect(paths(list.rows.value)).toStrictEqual(['physics', 'notes', 'Cover.png'])
  })

  it('keeps the folders open inside it, so they are drawn again where it opens', async () => {
    const { core } = vault()
    const list = listing(core)
    await list.opens(ROOT)
    await list.opens('physics')
    await list.opens('physics/heat')

    list.closes('physics')
    await list.opens('physics')

    expect(paths(list.rows.value)).toContain('physics/heat/Kelvin.md')
  })

  it('is never the root, which is the tree itself', async () => {
    const { core } = vault()
    const list = listing(core)
    await list.opens(ROOT)

    list.closes(ROOT)

    expect(list.rows.value).toHaveLength(3)
  })
})

describe('a change the vault reports', () => {
  it('reads the folder the path it names sits in', async () => {
    const { core, asked } = vault()
    const list = listing(core)
    await list.opens(ROOT)
    await list.opens('physics')
    asked.length = 0

    await list.changed(['physics/Entropy.md'])

    expect(asked).toStrictEqual(['physics'])
  })

  it('leaves every other open folder alone', async () => {
    const { core, asked } = vault()
    const list = listing(core)
    await list.opens(ROOT)
    await list.opens('physics')
    await list.opens('notes')
    asked.length = 0

    await list.changed(['physics/Entropy.md'])

    expect(asked).not.toContain('notes')
  })

  it('reads nothing for a path inside a folder that is closed', async () => {
    const { core, asked } = vault()
    const list = listing(core)
    await list.opens(ROOT)
    asked.length = 0

    await list.changed(['physics/heat/Kelvin.md'])

    expect(asked).toStrictEqual([])
  })

  it('reads the open folder above a folder it draws no row for', async () => {
    const { core, asked, puts } = vault()
    const list = listing(core)
    await list.opens(ROOT)
    puts(ROOT, folder('trips'))
    asked.length = 0

    await list.changed(['trips/Kyoto.md'])

    expect(asked).toStrictEqual([ROOT])
    expect(paths(list.rows.value)).toContain('trips')
  })

  it('reads every open folder when it names nothing at all', async () => {
    const { core, asked } = vault()
    const list = listing(core)
    await list.opens(ROOT)
    await list.opens('physics')
    asked.length = 0

    await list.changed([])

    expect([...asked].sort()).toStrictEqual([ROOT, 'physics'])
  })

  it('reads the folder a file left and the folder it arrived in', async () => {
    const { core, asked } = vault()
    const list = listing(core)
    await list.opens(ROOT)
    await list.opens('physics')
    await list.opens('notes')
    asked.length = 0

    await list.changed([], [{ from: 'physics/Entropy.md', to: 'notes/Entropy.md' }])

    expect([...asked].sort()).toStrictEqual(['notes', 'physics'])
  })

  it('leaves each chosen row chosen at where it went', async () => {
    const { core } = vault()
    const list = listing(core)
    await list.opens(ROOT)
    await list.opens('physics')
    list.chooses(['physics/Entropy.md', 'Cover.png'])

    await list.changed([], [{ from: 'physics/Entropy.md', to: 'notes/Entropy.md' }])

    expect(list.chosen.value).toStrictEqual(['notes/Entropy.md', 'Cover.png'])
  })
})

describe('the window coming back to the front', () => {
  /**
   * A picture dropped in by another program is not reported by the watcher, so
   * this is the pass that turns it up.
   */
  it('reads every open folder, and turns up what nothing reported', async () => {
    const { core, puts } = vault()
    const list = listing(core)
    await list.opens(ROOT)
    await list.opens('physics')
    puts('physics', file('physics/Diagram.png', { kind: 'other' }))

    await list.again()

    expect(paths(list.rows.value)).toContain('physics/Diagram.png')
  })

  it('reads no folder that is closed', async () => {
    const { core, asked } = vault()
    const list = listing(core)
    await list.opens(ROOT)
    asked.length = 0

    await list.again()

    expect(asked).toStrictEqual([ROOT])
  })
})

describe('the tree walked down to a path', () => {
  it('opens each folder above it, from the root down', async () => {
    const { core, asked } = vault()
    const list = listing(core)
    await list.opens(ROOT)
    asked.length = 0

    await list.reveals('physics/heat/Kelvin.md')

    expect(asked).toStrictEqual(['physics', 'physics/heat'])
  })

  it('draws the path it was walked down to', async () => {
    const { core } = vault()
    const list = listing(core)
    await list.opens(ROOT)

    await list.reveals('physics/heat/Kelvin.md')

    expect(paths(list.rows.value)).toContain('physics/heat/Kelvin.md')
  })

  it('leaves the path the whole of what is chosen', async () => {
    const { core } = vault()
    const list = listing(core)
    await list.opens(ROOT)
    list.chooses(['Cover.png'])

    await list.reveals('physics/heat/Kelvin.md')

    expect(list.chosen.value).toStrictEqual(['physics/heat/Kelvin.md'])
  })
})

describe('a folder that could not be read', () => {
  it('is said in the tab, and the tree stands as it was', async () => {
    const list = listing({
      list: async () => {
        throw new Error('the vault is not there')
      },
    })

    await list.opens(ROOT)

    expect(list.trouble.value).toContain('numen did not answer')
    expect(list.trouble.value).not.toContain('the vault is not there')
    expect(list.rows.value).toStrictEqual([])
  })
})

describe('a tab that has closed', () => {
  it('draws nothing of what it was given after it closed', async () => {
    const { core } = vault()
    const list = listing(core)
    await list.opens(ROOT)

    list.close()

    expect(list.rows.value).toStrictEqual([])
  })

  it('keeps nothing a question answers with after it closed', async () => {
    const { core } = vault()
    const list = listing(core)
    const asking = list.opens(ROOT)
    list.close()

    await asking

    expect(list.rows.value).toStrictEqual([])
  })
})

describe('a name nothing in a folder is filed under', () => {
  it('counts past what the folder already holds', async () => {
    const { core } = vault({ [ROOT]: [folder('New folder')] })
    const list = listing(core)
    await list.opens(ROOT)

    expect(list.freeIn(ROOT, 'New folder')).toBe('New folder 2')
  })
})
