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
import { listing, ROOT } from './listing'
import { NEW_FOLDER, RENAME } from './menu'

const file = (path: string, over: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  folder: false,
  kind: 'note',
  size: 1,
  ...over,
})

const folder = (path: string): Entry => file(path, { folder: true, kind: 'other', size: 0 })

/** A vault of two folders, a note, a book and a picture. */
const held: Record<string, readonly Entry[]> = {
  [ROOT]: [
    folder('physics'),
    folder('notes'),
    file('Entropy.md'),
    file('Heat.pdf', { kind: 'book' }),
    file('Cover.png', { kind: 'other' }),
  ],
  physics: [file('physics/Kelvin.md')],
  notes: [],
}

/** A moment for whatever a gesture asked the vault for to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

/** A tab of that vault, writing down everything it asked of the window. */
const tab = () => {
  const done: string[] = []
  const list = listing({ list: async (at: string) => held[at] ?? [] })
  const gestures = filing(list, {
    lands: (landing) => void done.push(`lands ${landing ? `${landing.at} ${landing.path}` : '—'}`),
    runs: (id, path, name) => void done.push(`runs ${id} ${path} ${name}`),
    moves: async (from, to) => void done.push(`moves ${from} ${to}`),
    makes: async (path) => void done.push(`makes ${path}`),
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

  it('leaves it chosen', async () => {
    const { list, one } = tab()
    await list.opens(ROOT)

    one.activate('Entropy.md')

    expect(list.chosen.value).toBe('Entropy.md')
  })

  it('does nothing at all for a row the tree does not draw', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    one.activate('physics/Kelvin.md')

    expect(done).toStrictEqual([])
  })
})

describe('a row let go of', () => {
  it('is filed in the folder it went into, under the name it carries', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    await one.move('Entropy.md', { into: 'physics' })

    expect(done).toStrictEqual(['moves Entropy.md physics/Entropy.md'])
  })

  it('is filed in the folder holding the row it came before', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)
    await list.opens('physics')

    await one.move('Entropy.md', { before: 'physics/Kelvin.md' })

    expect(done).toStrictEqual(['moves Entropy.md physics/Entropy.md'])
  })

  it('is a folder carried whole, under the name it carries', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)

    await one.move('physics', { into: 'notes' })

    expect(done).toStrictEqual(['moves physics notes/physics'])
  })

  it('asks nothing where it landed where it already was', async () => {
    const { done, list, one } = tab()
    await list.opens(ROOT)
    await list.opens('physics')

    await one.move('physics/Kelvin.md', { into: 'physics' })

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
})

describe('an item chosen in the menu on a row', () => {
  /** The menu on a row of the vault, standing open. */
  const asked = async (path = 'Entropy.md') => {
    const held = tab()
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

  it('hands a command over the file to the window', async () => {
    const { done, one } = await asked()

    one.chose('remove')

    expect(done).toStrictEqual(['runs remove Entropy.md Entropy.md'])
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
