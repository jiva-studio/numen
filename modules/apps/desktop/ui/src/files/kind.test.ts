/**
 * What a gesture on a row comes to, asked without a screen.
 *
 * A row stands for a file, and where activating one takes the person depends on
 * what the vault holds there and on nothing else. The negatives are the ones
 * worth having: a picture opens nothing, and a name that moves nothing asks the
 * vault for nothing.
 */
import { describe, expect, it } from 'vitest'
import type { Entry } from '../core'
import { filing, landingOf, renamedTo } from './kind'
import { folderOf, listing, ROOT } from './listing'
import { NEW_FOLDER, NEW_NOTE, RENAME } from './menu'
import { WORDS as words } from './words'

const file = (path: string, over: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  folder: false,
  kind: 'note',
  ...over,
})

const folder = (path: string): Entry => file(path, { folder: true, kind: 'other' })

/** A vault of three folders, a note, a book and a picture. */
const held: Record<string, readonly Entry[]> = {
  [ROOT]: [
    folder('physics'),
    folder('notes'),
    folder('heat'),
    file('Entropy.md'),
    file('Heat.pdf', { kind: 'book' }),
    file('Cover.png', { kind: 'other' }),
  ],
  physics: [file('physics/Kelvin.md')],
  notes: [],
  heat: [file('heat/Entropy.md')],
}

/** A moment for whatever a gesture asked the vault for to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

/** A tab of that vault, writing down everything it asked of the window. */
const tab = (refuses = false) => {
  const done: string[] = []
  // Its own copy, so a folder one test makes is not there for the next.
  const vault: Record<string, readonly Entry[]> = { ...held }
  const list = listing({ list: async (at: string) => vault[at] ?? [] })
  const gestures = filing(list, {
    lands: (landing) => void done.push(`lands ${landing ? `${landing.at} ${landing.path}` : '—'}`),
    runs: (id, paths, name) => void done.push(`runs ${id} ${paths.join(' ')} ${name}`),
    moves: async (from, to) => void done.push(`moves ${from} ${to}`),
    carries: (paths) => void done.push(`carries ${paths.join(' ') || '—'}`),
    makes: async (path) => {
      done.push(`makes ${path}`)
      if (refuses) return
      const into = folderOf(path)
      vault[into] = [...(vault[into] ?? []), folder(path)]
      vault[path] = []
    },
    writes: async (folder) => {
      const made = folder === ROOT ? 'Untitled note.md' : `${folder}/Untitled note.md`
      done.push(`writes ${made}`)
      return made
    },
    says: (text) => void done.push(`says ${text}`),
  })
  return { done, list, one: gestures }
}

describe('where a row activated takes the person', () => {
  it('is the note it stands for, in a tab of its own', () => {
    expect(landingOf(file('Entropy.md'))).toStrictEqual({
      at: 'note',
      path: 'Entropy.md',
      title: 'Entropy.md',
    })
  })

  it('is the book it stands for, opened where it begins', () => {
    expect(landingOf(file('Heat.pdf', { kind: 'book' }))).toStrictEqual({
      at: 'document',
      path: 'Heat.pdf',
      title: 'Heat.pdf',
      start: 0,
      length: 0,
    })
  })

  it('is nowhere for a file the vault holds no source for', () => {
    expect(landingOf(file('Cover.png', { kind: 'other' }))).toBeNull()
  })

  it('is nowhere for a folder, which opens where it stands', () => {
    expect(landingOf(folder('physics'))).toBeNull()
  })
})

describe('a name typed over a row', () => {
  it('files it under that name, in the folder it is already in', () => {
    expect(renamedTo('physics/Kelvin.md', 'Celsius.md')).toBe('physics/Celsius.md')
  })

  it('files it at the top of the vault where that is where it is', () => {
    expect(renamedTo('Entropy.md', 'Order.md')).toBe('Order.md')
  })

  it('moves nothing where the name is the one it carries', () => {
    expect(renamedTo('physics/Kelvin.md', 'Kelvin.md')).toBe('')
  })

  it('moves nothing where nothing was typed', () => {
    expect(renamedTo('physics/Kelvin.md', '   ')).toBe('')
  })

  /** A name is a name and not a path: the field renames, and dragging moves. */
  it('moves nothing where the name names a folder of its own', () => {
    expect(renamedTo('physics/Kelvin.md', 'heat/Kelvin.md')).toBe('')
  })

  it('keeps the ending the file carries, where the name carries none', () => {
    expect(renamedTo('physics/Kelvin.md', 'Celsius')).toBe('physics/Celsius.md')
    expect(renamedTo('Cover.png', 'Jacket')).toBe('Jacket.png')
  })

  it('takes the ending the name carries, where it carries one', () => {
    expect(renamedTo('Cover.png', 'Jacket.jpg')).toBe('Jacket.jpg')
  })

  it('moves nothing where the name is the one it carries, ending and all', () => {
    expect(renamedTo('physics/Kelvin.md', 'Kelvin')).toBe('')
  })

  /** A name beginning with a dot is a name, and the whole of it. */
  it('keeps nothing for a file whose name is an ending', () => {
    expect(renamedTo('.keep', 'ignored')).toBe('ignored')
  })

  /**
   * A dot is punctuation more often than it is an ending. A note called after a
   * chapter, a figure or a person keeps the ending its file carries, or it
   * stops being a note the moment it is named.
   */
  it('reads a dot inside a sentence as punctuation, not as an ending', () => {
    expect(renamedTo('physics/Kelvin.md', 'Ch. 2 heat')).toBe('physics/Ch. 2 heat.md')
    expect(renamedTo('Cover.png', 'Fig. 3')).toBe('Fig. 3.png')
    expect(renamedTo('Kelvin.md', 'Mr. Smith')).toBe('Mr. Smith.md')
  })

  it('reads a dot with no space after it as an ending, in any script', () => {
    expect(renamedTo('Kelvin.md', 'Notes.txt')).toBe('Notes.txt')
    expect(renamedTo('Cover.png', 'Jacket.jpeg')).toBe('Jacket.jpeg')
    expect(renamedTo('Kelvin.md', 'Записка.текст')).toBe('Записка.текст')
    expect(renamedTo('Kelvin.md', 'Заметка')).toBe('Заметка.md')
  })

  /**
   * What the rule gives up: a dot between digits reads as an ending, so a note
   * called after a version keeps the name it was given and takes no other.
   */
  it('reads a version as an ending, and leaves it alone', () => {
    expect(renamedTo('Kelvin.md', 'v1.2')).toBe('v1.2')
  })

  /** A name that is a dot and an ending is the whole name, and takes no other. */
  it('adds nothing to a name that is an ending', () => {
    expect(renamedTo('physics/Kelvin.md', '.gitignore')).toBe('physics/.gitignore')
  })

  it('takes what was typed for a folder, which carries no ending at all', () => {
    expect(renamedTo('physics/v1.0', 'v2', true)).toBe('physics/v2')
  })
})

describe('a row activated', () => {
  it('takes the person to the note it stands for', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    one.activate('Entropy.md')

    expect(done).toStrictEqual(['lands note Entropy.md'])
  })

  it('takes the person to the book it stands for', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    one.activate('Heat.pdf')

    expect(done).toStrictEqual(['lands document Heat.pdf'])
  })

  it('takes the person nowhere for a file the vault holds no source for', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    one.activate('Cover.png')

    expect(done).toStrictEqual(['lands —'])
  })

  it('leaves it the whole of what is chosen', async () => {
    const { list, one } = tab()
    await list.opens(ROOT)

    one.activate('Entropy.md')

    expect(list.chosen.value).toStrictEqual(['Entropy.md'])
  })

  it('does nothing at all for a row the tree does not draw', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    one.activate('physics/Kelvin.md')

    expect(done).toStrictEqual([])
  })
})

describe('rows carried out of the tree', () => {
  it('are the note the row stands for', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    one.carry(['Entropy.md'])

    expect(done).toStrictEqual(['carries Entropy.md'])
  })

  it('are every note of a selection, in the order the rows were carried', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)
    await list.opens('physics')

    one.carry(['Entropy.md', 'physics/Kelvin.md'])

    expect(done).toStrictEqual(['carries Entropy.md physics/Kelvin.md'])
  })

  it('are the notes of a mixed selection, and the rest stay where they are', async () => {
    // The vault's links are between notes.
    const { done, list, one } = tab()
    await list.opens(ROOT)

    one.carry(['physics', 'Entropy.md', 'Heat.pdf', 'Cover.png'])

    expect(done).toStrictEqual(['carries Entropy.md'])
  })

  it('are nothing where the rows hold no note at all', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    one.carry(['physics', 'Heat.pdf', 'Cover.png'])

    expect(done).toStrictEqual(['carries —'])
  })

  it('are nothing for a row the tree is no longer drawing', async () => {
    const { done, one } = tab()

    one.carry(['Entropy.md'])

    expect(done).toStrictEqual(['carries —'])
  })

  it('are nothing once they have been let go of', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    one.carry(['Entropy.md'])
    one.drop()

    expect(done).toStrictEqual(['carries Entropy.md', 'carries —'])
  })
})

describe('rows let go of', () => {
  it('are filed in the folder they went into, under the names they carry', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    await one.move(['Entropy.md'], { into: 'physics' })

    expect(done).toStrictEqual(['moves Entropy.md physics/Entropy.md'])
  })

  it('are filed in the folder holding the row they came before', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)
    await list.opens('physics')

    await one.move(['Entropy.md'], { before: 'physics/Kelvin.md' })

    expect(done).toStrictEqual(['moves Entropy.md physics/Entropy.md'])
  })

  it('are a folder carried whole, under the name it carries', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    await one.move(['physics'], { into: 'notes' })

    expect(done).toStrictEqual(['moves physics notes/physics'])
  })

  it('all land in the one folder', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    await one.move(['Entropy.md', 'Heat.pdf', 'Cover.png'], { into: 'notes' })

    expect(done).toStrictEqual([
      'moves Entropy.md notes/Entropy.md',
      'moves Heat.pdf notes/Heat.pdf',
      'moves Cover.png notes/Cover.png',
    ])
  })

  it('read the folders again once, when all of them are done', async () => {
    const asked: string[] = []
    const list = listing({
      list: async (at: string) => {
        asked.push(at)
        return held[at] ?? []
      },
    })
    const one = filing(list, {
      lands: () => {},
      runs: () => {},
      moves: async () => {},
      carries: () => {},
      makes: async () => {},
      writes: async () => '',
      says: () => {},
    })
    await list.opens(ROOT)
    asked.length = 0

    await one.move(['Entropy.md', 'Heat.pdf'], { into: 'notes' })

    expect(asked.filter((at) => at === ROOT)).toStrictEqual([ROOT])
  })

  it('leave behind the one whose name that folder holds, and say which', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    await one.move(['Entropy.md', 'Heat.pdf'], { into: 'heat' })

    expect(done).toStrictEqual(['moves Heat.pdf heat/Heat.pdf', `says ${words.taken} Entropy.md`])
  })

  it('ask nothing where they landed where they already were', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)
    await list.opens('physics')

    await one.move(['physics/Kelvin.md'], { into: 'physics' })

    expect(done).toStrictEqual([])
  })

  it('ask nothing at all where nothing was carried', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    await one.move([], { into: 'physics' })

    expect(done).toStrictEqual([])
  })
})

describe('rows asked to go', () => {
  it('are handed to the window as one command over all of them', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    one.remove(['Entropy.md', 'Cover.png'])

    expect(done).toStrictEqual(['runs remove Entropy.md Cover.png Entropy.md'])
  })

  it('ask for nothing where nothing is selected', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    one.remove([])

    expect(done).toStrictEqual([])
  })
})

describe('a name given to a row', () => {
  it('moves the file within the folder it is in', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    await one.rename('Entropy.md', 'Order.md')

    expect(done).toStrictEqual(['moves Entropy.md Order.md'])
  })

  it('asks the vault for nothing where the name is the one it carries', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    await one.rename('Entropy.md', 'Entropy.md')

    expect(done).toStrictEqual([])
  })

  it('keeps the ending the file carries, so a note typed over stays a note', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    await one.rename('Entropy.md', 'Order')

    expect(done).toStrictEqual(['moves Entropy.md Order.md'])
  })

  it('takes what was typed for a folder', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    await one.rename('physics', 'heat')

    expect(done).toStrictEqual(['moves physics heat'])
  })
})

describe('an item chosen in the menu on a row', () => {
  /** The menu on a row of the vault, standing open. */
  const asked = async (path: string | null = 'Entropy.md', refuses = false) => {
    const held = tab(refuses)
    await held.list.opens(ROOT)
    held.one.asks({ path, at: { x: 0, y: 0 } })
    return held
  }

  it('puts the name of the row in a field, and asks the window for nothing', async () => {
    const { done, one } = await asked()

    one.chose(RENAME)

    expect(one.renaming.value).toBe('Entropy.md')
    expect(done).toStrictEqual([])
  })

  it('makes a folder beside the row, under a name nothing there carries', async () => {
    const { done, one } = await asked()

    one.chose(NEW_FOLDER)
    await settles()

    expect(done).toStrictEqual(['makes New folder'])
  })

  it('makes a folder inside the row where the row is a folder', async () => {
    const { done, one } = await asked('physics')

    one.chose(NEW_FOLDER)
    await settles()

    expect(done).toStrictEqual(['makes physics/New folder'])
  })

  it('puts the name of a folder it made in a field', async () => {
    const { one } = await asked()

    one.chose(NEW_FOLDER)
    await settles()

    expect(one.renaming.value).toBe('New folder')
  })

  // A field over a folder the vault does not hold renames nothing, and asks
  // the person to name what is not there.
  it('puts no name in a field where the folder was refused', async () => {
    const { one } = await asked('Entropy.md', true)

    one.chose(NEW_FOLDER)
    await settles()

    expect(one.renaming.value).toBeNull()
  })

  it('makes a note beside the row, and puts its name in a field', async () => {
    const { done, one } = await asked()

    one.chose(NEW_NOTE)
    await settles()

    expect(done).toStrictEqual(['writes Untitled note.md'])
    expect(one.renaming.value).toBe('Untitled note.md')
  })

  it('makes a note inside the row where the row is a folder', async () => {
    const { done, one } = await asked('physics')

    one.chose(NEW_NOTE)
    await settles()

    expect(done).toStrictEqual(['writes physics/Untitled note.md'])
  })

  it('makes a note and a folder at the root, asked off every row', async () => {
    const { done, one } = await asked(null)

    one.chose(NEW_NOTE)
    await settles()
    one.asks({ path: null, at: { x: 0, y: 0 } })
    one.chose(NEW_FOLDER)
    await settles()

    expect(done).toStrictEqual(['writes Untitled note.md', 'makes New folder'])
  })

  it('renames nothing where the menu was asked off every row', async () => {
    const { one } = await asked(null)

    one.chose(RENAME)

    expect(one.renaming.value).toBeNull()
  })

  it('hands a command over the file to the window', async () => {
    const { done, one } = await asked()

    one.chose('remove')

    expect(done).toStrictEqual(['runs remove Entropy.md Entropy.md'])
  })

  it('hands the whole selection to the window, on a row standing in it', async () => {
    const held = await asked()
    held.list.chooses(['Entropy.md', 'Cover.png'])

    held.one.chose('remove')

    expect(held.done).toStrictEqual(['runs remove Entropy.md Cover.png Entropy.md'])
  })

  it('hands the one row to the window, on a row standing outside the selection', async () => {
    const held = await asked()
    held.list.chooses(['Cover.png'])

    held.one.chose('remove')

    expect(held.done).toStrictEqual(['runs remove Entropy.md Entropy.md'])
  })

  it('does nothing at all for a choice the menu does not offer', async () => {
    const { done, one } = await asked()

    one.chose('destroyEverything')

    expect(done).toStrictEqual([])
  })

  it('puts the menu away whatever was chosen', async () => {
    const { one } = await asked()

    one.chose('remove')

    expect(one.menu.value).toBeNull()
  })
})
