/**
 * Unit tests for useFilesTab composable and file interactions.
 */
import { describe, expect, it } from 'vitest'
import type { Entry } from '@/shared/file'
import { getLandingDestination, useFilesTab } from './useFilesTab'
import { getFolderPath, ROOT, useFileTree } from './useFileTree'
import { NEW_DECK, NEW_FOLDER, NEW_NOTE, NEW_PRESET, NEW_STENCIL, RENAME } from '../lib/menu'
import { WORDS as words } from '../words'

const file = (path: string, over: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  folder: false,
  kind: 'note',
  type: 'note',
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
const settle = () => new Promise((done) => setTimeout(done, 0))

/** A tab of that vault, writing down everything it asked of the window. */
const tab = (fails = false) => {
  const done: string[] = []
  const vault: Record<string, readonly Entry[]> = { ...held }
  const list = useFileTree({ list: async (at: string) => vault[at] ?? [] })
  const gestures = useFilesTab(list, {
    openDestination: (landing) =>
      void done.push(`lands ${landing ? `${landing.at} ${landing.path}` : '—'}`),
    runCommand: (id, paths, name) => void done.push(`runs ${id} ${paths.join(' ')} ${name}`),
    movePath: async (from, to) => void done.push(`moves ${from} ${to}`),
    setDraggedPaths: (paths) => void done.push(`drags ${paths.join(' ') || '—'}`),
    createFolder: async (path) => {
      done.push(`makes ${path}`)
      if (fails) return
      const into = getFolderPath(path)
      vault[into] = [...(vault[into] ?? []), folder(path)]
      vault[path] = []
    },
    createNote: async (folderPath) => {
      const made = folderPath === ROOT ? 'Untitled note.md' : `${folderPath}/Untitled note.md`
      done.push(`writes ${made}`)
      return made
    },
    createDeck: async (folderPath, name) => {
      done.push(`decks ${folderPath === ROOT ? '/' : folderPath} ${name}`)
      if (fails) return ''
      const made = folderPath === ROOT ? `${name}.note` : `${folderPath}/${name}.note`
      vault[folderPath] = [...(vault[folderPath] ?? []), file(made, { type: 'deck' })]
      return made
    },
    createStencil: async (folderPath, name) => {
      done.push(`stencils ${folderPath === ROOT ? '/' : folderPath} ${name}`)
      if (fails) return ''
      const made = folderPath === ROOT ? `${name}.note` : `${folderPath}/${name}.note`
      vault[folderPath] = [...(vault[folderPath] ?? []), file(made, { type: 'stencil' })]
      return made
    },
    createPreset: async (folderPath, name) => {
      done.push(`presets ${folderPath === ROOT ? '/' : folderPath} ${name}`)
      if (fails) return ''
      const made = folderPath === ROOT ? `${name}.note` : `${folderPath}/${name}.note`
      vault[folderPath] = [...(vault[folderPath] ?? []), file(made, { type: 'preset' })]
      return made
    },
    importUrl: async (folderPath, url) => {
      done.push(`imports ${folderPath === ROOT ? '/' : folderPath} ${url}`)
      return fails ? '' : 'made.url'
    },
    showError: (text) => void done.push(`says ${text}`),
  })
  return { done, list, one: gestures }
}

describe('where a row activated takes the person', () => {
  it('is the note it stands for, in a tab of its own', () => {
    expect(getLandingDestination(file('Entropy.md'))).toStrictEqual({
      at: 'file',
      path: 'Entropy.md',
      title: 'Entropy.md',
    })
  })

  it('is the deck it stands for, named and no more', () => {
    expect(getLandingDestination(file('Animals.md', { type: 'deck' }))).toStrictEqual({
      at: 'file',
      path: 'Animals.md',
      title: 'Animals.md',
    })
  })

  it('is the stencil it stands for, named and no more', () => {
    expect(getLandingDestination(file('Animal.md', { type: 'stencil' }))).toStrictEqual({
      at: 'file',
      path: 'Animal.md',
      title: 'Animal.md',
    })
  })

  it('is the book it stands for, named and no more', () => {
    expect(getLandingDestination(file('Heat.pdf', { kind: 'book' }))).toStrictEqual({
      at: 'file',
      path: 'Heat.pdf',
      title: 'Heat.pdf',
    })
  })

  it('is the file it stands for even where the vault holds no source there', () => {
    expect(getLandingDestination(file('Cover.png', { kind: 'other' }))).toStrictEqual({
      at: 'file',
      path: 'Cover.png',
      title: 'Cover.png',
    })
  })

  it('is nowhere for a folder, which opens where it stands', () => {
    expect(getLandingDestination(folder('physics'))).toBeNull()
  })
})

describe('a row activated', () => {
  it('takes the person to the note it stands for', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    one.activate('Entropy.md')

    expect(done).toStrictEqual(['lands file Entropy.md'])
  })

  it('takes the person to the book it stands for', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    one.activate('Heat.pdf')

    expect(done).toStrictEqual(['lands file Heat.pdf'])
  })

  it('takes the person to a file the vault holds no source for', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    one.activate('Cover.png')

    expect(done).toStrictEqual(['lands file Cover.png'])
  })

  it('leaves it the whole of what is chosen', async () => {
    const { list, one } = tab()
    await list.openFolder(ROOT)

    one.activate('Entropy.md')

    expect(list.selectedPaths.value).toStrictEqual(['Entropy.md'])
  })

  it('does nothing at all for a row the tree does not draw', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    one.activate('physics/Kelvin.md')

    expect(done).toStrictEqual([])
  })
})

describe('rows dragged out of the tree', () => {
  it('are the note the row stands for', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    one.drag(['Entropy.md'])

    expect(done).toStrictEqual(['drags Entropy.md'])
  })

  it('are every note of a selection, in the order the rows were dragged', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)
    await list.openFolder('physics')

    one.drag(['Entropy.md', 'physics/Kelvin.md'])

    expect(done).toStrictEqual(['drags Entropy.md physics/Kelvin.md'])
  })

  it('are the notes of a mixed selection, and the rest stay where they are', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    one.drag(['physics', 'Entropy.md', 'Heat.pdf', 'Cover.png'])

    expect(done).toStrictEqual(['drags Entropy.md'])
  })

  it('are nothing where the rows hold no note at all', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    one.drag(['physics', 'Heat.pdf', 'Cover.png'])

    expect(done).toStrictEqual(['drags —'])
  })

  it('are nothing for a row the tree is no longer drawing', async () => {
    const { done, one } = tab()

    one.drag(['Entropy.md'])

    expect(done).toStrictEqual(['drags —'])
  })

  it('are nothing once they have been let go of', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    one.drag(['Entropy.md'])
    one.drop()

    expect(done).toStrictEqual(['drags Entropy.md', 'drags —'])
  })
})

describe('rows let go of', () => {
  it('are filed in the folder they went into, under the names they carry', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    await one.move(['Entropy.md'], { into: 'physics' })

    expect(done).toStrictEqual(['moves Entropy.md physics/Entropy.md'])
  })

  it('are filed in the folder holding the row they came before', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)
    await list.openFolder('physics')

    await one.move(['Entropy.md'], { before: 'physics/Kelvin.md' })

    expect(done).toStrictEqual(['moves Entropy.md physics/Entropy.md'])
  })

  it('are a folder dragged whole, under the name it carries', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    await one.move(['physics'], { into: 'notes' })

    expect(done).toStrictEqual(['moves physics notes/physics'])
  })

  it('all land in the one folder', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    await one.move(['Entropy.md', 'Heat.pdf', 'Cover.png'], { into: 'notes' })

    expect(done).toStrictEqual([
      'moves Entropy.md notes/Entropy.md',
      'moves Heat.pdf notes/Heat.pdf',
      'moves Cover.png notes/Cover.png',
    ])
  })

  it('read the folders again once, when all of them are done', async () => {
    const asked: string[] = []
    const list = useFileTree({
      list: async (at: string) => {
        asked.push(at)
        return held[at] ?? []
      },
    })
    const one = useFilesTab(list, {
      openDestination: () => {},
      runCommand: () => {},
      movePath: async () => {},
      setDraggedPaths: () => {},
      createFolder: async () => {},
      createNote: async () => '',
      createDeck: async () => '',
      createStencil: async () => '',
      createPreset: async () => '',
      importUrl: async () => '',
      showError: () => {},
    })
    await list.openFolder(ROOT)
    asked.length = 0

    await one.move(['Entropy.md', 'Heat.pdf'], { into: 'notes' })

    expect(asked.filter((at) => at === ROOT)).toStrictEqual([ROOT])
  })

  it('leave behind the one whose name that folder holds, and say which', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    await one.move(['Entropy.md', 'Heat.pdf'], { into: 'heat' })

    expect(done).toStrictEqual(['moves Heat.pdf heat/Heat.pdf', `says ${words.taken} Entropy.md`])
  })

  it('ask nothing where they landed where they already were', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)
    await list.openFolder('physics')

    await one.move(['physics/Kelvin.md'], { into: 'physics' })

    expect(done).toStrictEqual([])
  })

  it('ask nothing at all where nothing was dragged', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    await one.move([], { into: 'physics' })

    expect(done).toStrictEqual([])
  })
})

describe('rows asked to go', () => {
  it('are handed to the window as one command over all of them', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    one.remove(['Entropy.md', 'Cover.png'])

    expect(done).toStrictEqual(['runs remove Entropy.md Cover.png Entropy.md'])
  })

  it('ask for nothing where nothing is selected', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    one.remove([])

    expect(done).toStrictEqual([])
  })
})

describe('a name given to a row', () => {
  it('moves the file within the folder it is in', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    await one.rename('Entropy.md', 'Order.md')

    expect(done).toStrictEqual(['moves Entropy.md Order.md'])
  })

  it('asks the vault for nothing where the name is the one it carries', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    await one.rename('Entropy.md', 'Entropy.md')

    expect(done).toStrictEqual([])
  })

  it('keeps the ending the file carries, so a note typed over stays a note', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    await one.rename('Entropy.md', 'Order')

    expect(done).toStrictEqual(['moves Entropy.md Order.md'])
  })

  it('takes what was typed for a folder', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)

    await one.rename('physics', 'heat')

    expect(done).toStrictEqual(['moves physics heat'])
  })
})

describe('an item chosen in the menu on a row', () => {
  const openRowMenu = async (path: string | null = 'Entropy.md', fails = false) => {
    const heldState = tab(fails)
    await heldState.list.openFolder(ROOT)
    heldState.one.openMenu({ path, at: { x: 0, y: 0 } })
    return heldState
  }

  it('puts the name of the row in a field, and asks the window for nothing', async () => {
    const { done, one } = await openRowMenu()

    one.chooseMenuItem(RENAME)

    expect(one.renamingPath.value).toBe('Entropy.md')
    expect(done).toStrictEqual([])
  })

  it('makes a folder beside the row, under a name nothing there carries', async () => {
    const { done, one } = await openRowMenu()

    one.chooseMenuItem(NEW_FOLDER)
    await settle()

    expect(done).toStrictEqual(['makes New folder'])
  })

  it('makes a folder inside the row where the row is a folder', async () => {
    const { done, one } = await openRowMenu('physics')

    one.chooseMenuItem(NEW_FOLDER)
    await settle()

    expect(done).toStrictEqual(['makes physics/New folder'])
  })

  it('puts the name of a folder it made in a field', async () => {
    const { one } = await openRowMenu()

    one.chooseMenuItem(NEW_FOLDER)
    await settle()

    expect(one.renamingPath.value).toBe('New folder')
  })

  it('puts no name in a field where the folder was refused', async () => {
    const { one } = await openRowMenu('Entropy.md', true)

    one.chooseMenuItem(NEW_FOLDER)
    await settle()

    expect(one.renamingPath.value).toBeNull()
  })

  it('makes a note beside the row, and puts its name in a field', async () => {
    const { done, one } = await openRowMenu()

    one.chooseMenuItem(NEW_NOTE)
    await settle()

    expect(done).toStrictEqual(['writes Untitled note.md'])
    expect(one.renamingPath.value).toBe('Untitled note.md')
  })

  it('makes a deck beside the row, and puts its name in a field', async () => {
    const { done, one } = await openRowMenu()

    one.chooseMenuItem(NEW_DECK)
    await settle()

    expect(done).toStrictEqual([`decks / ${words.newDeck}`])
    expect(one.renamingPath.value).toBe(`${words.newDeck}.note`)
  })

  it('asks under the name alone, putting no ending on it', async () => {
    const { done, one } = await openRowMenu()

    one.chooseMenuItem(NEW_DECK)
    await settle()

    expect(done[0]).not.toContain('.md')
  })

  it('makes a stencil the same way, under a name of its own', async () => {
    const { done, one } = await openRowMenu()

    one.chooseMenuItem(NEW_STENCIL)
    await settle()

    expect(done).toStrictEqual([`stencils / ${words.newStencil}`])
  })

  it('makes a preset the same way, under a name of its own', async () => {
    const { done, one } = await openRowMenu()

    one.chooseMenuItem(NEW_PRESET)
    await settle()

    expect(done).toStrictEqual([`presets / ${words.newPreset}`])
    expect(one.renamingPath.value).toBe(`${words.newPreset}.note`)
  })

  it('stands the preset in the tree as a preset', async () => {
    const { one } = await openRowMenu()

    one.chooseMenuItem(NEW_PRESET)
    await settle()

    expect(one.list.getEntryAt(`${words.newPreset}.note`)?.type).toBe('preset')
  })

  it('makes a preset inside the row where the row is a folder', async () => {
    const { done, one } = await openRowMenu('physics')

    one.chooseMenuItem(NEW_PRESET)
    await settle()

    expect(done).toStrictEqual([`presets physics ${words.newPreset}`])
  })

  it('names nothing where the vault made no preset', async () => {
    const { one } = await openRowMenu(undefined, true)

    one.chooseMenuItem(NEW_PRESET)
    await settle()

    expect(one.renamingPath.value).toBeNull()
  })

  it('makes a deck inside the row where the row is a folder', async () => {
    const { done, one } = await openRowMenu('physics')

    one.chooseMenuItem(NEW_DECK)
    await settle()

    expect(done).toStrictEqual([`decks physics ${words.newDeck}`])
  })

  it('names nothing where the vault made no deck', async () => {
    const { one } = await openRowMenu(undefined, true)

    one.chooseMenuItem(NEW_DECK)
    await settle()

    expect(one.renamingPath.value).toBeNull()
  })

  it('makes a note inside the row where the row is a folder', async () => {
    const { done, one } = await openRowMenu('physics')

    one.chooseMenuItem(NEW_NOTE)
    await settle()

    expect(done).toStrictEqual(['writes physics/Untitled note.md'])
  })

  it('makes a note and a folder at the root, asked off every row', async () => {
    const { done, one } = await openRowMenu(null)

    one.chooseMenuItem(NEW_NOTE)
    await settle()
    one.openMenu({ path: null, at: { x: 0, y: 0 } })
    one.chooseMenuItem(NEW_FOLDER)
    await settle()

    expect(done).toStrictEqual(['writes Untitled note.md', 'makes New folder'])
  })

  it('renames nothing where the menu was asked off every row', async () => {
    const { one } = await openRowMenu(null)

    one.chooseMenuItem(RENAME)

    expect(one.renamingPath.value).toBeNull()
  })

  it('hands a command over the file to the window', async () => {
    const { done, one } = await openRowMenu()

    one.chooseMenuItem('remove')

    expect(done).toStrictEqual(['runs remove Entropy.md Entropy.md'])
  })

  it('hands the whole selection to the window, on a row standing in it', async () => {
    const heldState = await openRowMenu()
    heldState.list.selectPaths(['Entropy.md', 'Cover.png'])

    heldState.one.chooseMenuItem('remove')

    expect(heldState.done).toStrictEqual(['runs remove Entropy.md Cover.png Entropy.md'])
  })

  it('hands the one row to the window, on a row standing outside the selection', async () => {
    const heldState = await openRowMenu()
    heldState.list.selectPaths(['Cover.png'])

    heldState.one.chooseMenuItem('remove')

    expect(heldState.done).toStrictEqual(['runs remove Entropy.md Entropy.md'])
  })

  it('does nothing at all for a choice the menu does not offer', async () => {
    const { done, one } = await openRowMenu()

    one.chooseMenuItem('destroyEverything')

    expect(done).toStrictEqual([])
  })

  it('puts the menu away whatever was chosen', async () => {
    const { one } = await openRowMenu()

    one.chooseMenuItem('remove')

    expect(one.menu.value).toBeNull()
  })
})
