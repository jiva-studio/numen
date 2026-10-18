/**
 * Unit tests for useFilesTab composable and file interactions.
 */
import { describe, expect, it } from 'vitest'
import type { Entry } from '@/entities/file'
import { getLandingDestination, useFilesTab, type FilesTabState } from './useFilesTab'
import { getFolderPath, ROOT, useFileTree } from './useFileTree'
import { NEW_DECK, NEW_FOLDER, NEW_NOTE, NEW_PRESET, NEW_STENCIL, RENAME } from '../lib/menu'
import { WORDS as words } from '../words'

const file = (path: string, over: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  isFolder: false,
  kind: 'note',
  type: 'note',
  ...over,
})

const folder = (path: string): Entry => file(path, { isFolder: true, kind: 'other' })

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
const tab = (fails = false, extra: Record<string, readonly Entry[]> = {}) => {
  const done: string[] = []
  const asked: string[] = []
  const sources: string[] = []
  const vault: Record<string, readonly Entry[]> = { ...held, ...extra }
  const list = useFileTree({
    list: async (at: string) => {
      asked.push(at)
      return vault[at] ?? []
    },
  })
  let receivedState: FilesTabState | undefined = undefined
  const gestures = useFilesTab(list, {
    openDestination: (landing, state) => {
      receivedState = state
      done.push(`lands ${landing ? `${landing.at} ${landing.path}` : '—'}`)
    },
    runCommand: (id, paths, target) => {
      done.push(`runs ${id} ${paths.join(' ')} ${target.name}`)
      sources.push(target.source)
    },
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
      if (fails) return ''
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
    importUrls: async (folderPath, urls) => {
      done.push(`imports-multi ${folderPath === ROOT ? '/' : folderPath} ${urls.join(' ')}`)
      return fails ? [] : urls.map(() => 'made.url')
    },
    showError: (text) => void done.push(`says ${text}`),
  })
  return { done, list, one: gestures, asked, sources, getState: () => receivedState }
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
    const { done, list, one, getState } = tab()
    await list.openFolder(ROOT)

    one.activate('Entropy.md')

    expect(done).toStrictEqual(['lands file Entropy.md'])
    expect(getState()).toBe(one)
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
      importUrls: async () => [],
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

describe('useFilesTab helper and interaction methods', () => {
  it('covers folder operations, selections, renaming and dismiss', async () => {
    const { list, one, done } = tab()
    await list.openFolder(ROOT)

    one.open('physics')
    one.close('physics')
    one.select(['physics'])
    expect(list.selectedPaths.value).toEqual(['physics'])

    one.setRenamingPath('physics')
    expect(one.renamingPath.value).toBe('physics')

    one.openMenu({ path: 'physics', at: { x: 10, y: 10 } })
    expect(one.menu.value).not.toBeNull()
    one.dismissMenu()
    expect(one.menu.value).toBeNull()

    one.importUrl('https://example.com/video')
    expect(done).toContain('imports physics https://example.com/video')

    one.importUrls(['https://example.com/1', 'https://example.com/2'])
    expect(done).toContain('imports-multi physics https://example.com/1 https://example.com/2')

    expect(one.getNameOf('Entropy.md')).toBe('Entropy.md')
    expect(one.getNameOf('nonexistent/file.txt')).toBe('file.txt')
    expect(one.canRun({} as never)).toBe(true)
    expect(one.getFolderFor(null)).toBe(ROOT)
  })
})

describe('a field already holding a name, and a file refused', () => {
  const refuse = async (id: string) => {
    const { one, list } = tab(true)
    await list.openFolder(ROOT)
    one.setRenamingPath('Cover.png')
    one.openMenu({ path: 'Entropy.md', at: { x: 0, y: 0 } })
    one.chooseMenuItem(id)
    await settle()
    return one.renamingPath.value
  }

  it.each([
    ['folder', NEW_FOLDER],
    ['note', NEW_NOTE],
    ['deck', NEW_DECK],
    ['stencil', NEW_STENCIL],
    ['preset', NEW_PRESET],
  ])('leaves the field alone where the %s was refused', async (_kind, id) => {
    expect(await refuse(id)).toBe('Cover.png')
  })
})

describe('a menu item chosen off every row', () => {
  it('leaves the field alone and runs nothing for an item that needs a row', async () => {
    const { done, one } = tab()
    one.setRenamingPath('Cover.png')

    one.openMenu({ path: null, at: { x: 0, y: 0 } })
    one.chooseMenuItem(RENAME)
    one.openMenu({ path: null, at: { x: 0, y: 0 } })
    one.chooseMenuItem('transcribe')

    expect(one.renamingPath.value).toBe('Cover.png')
    expect(done).toStrictEqual([])
  })
})

describe('the source a command is run over', () => {
  it('is the kind of file the row stands for', async () => {
    const { list, one, sources } = tab()
    await list.openFolder(ROOT)

    one.openMenu({ path: 'Heat.pdf', at: { x: 0, y: 0 } })
    one.chooseMenuItem('recognise')
    one.remove(['Entropy.md'])

    expect(sources).toStrictEqual(['book', 'note'])
  })

  it('is other for a row the tree does not hold', () => {
    const { one, sources } = tab()

    one.openMenu({ path: 'ghost.bin', at: { x: 0, y: 0 } })
    one.chooseMenuItem('recognise')

    expect(sources).toStrictEqual(['other'])
  })
})

describe('rows let go of, in more detail', () => {
  it('are filed at the top of the vault under the names they carry', async () => {
    const { done, list, one } = tab()
    await list.openFolder(ROOT)
    await list.openFolder('physics')

    await one.move(['physics/Kelvin.md'], { into: null })

    expect(done).toStrictEqual(['moves physics/Kelvin.md Kelvin.md'])
  })

  it('leave behind the second of two that carry the one name', async () => {
    const { done, one } = tab()

    await one.move(['heat/Entropy.md', 'physics/Entropy.md'], { into: 'notes' })

    expect(done).toStrictEqual([
      'moves heat/Entropy.md notes/Entropy.md',
      `says ${words.taken} Entropy.md`,
    ])
  })

  it('say every name left behind, parted by commas', async () => {
    const { done, one } = tab(false, {
      notes: [file('notes/Entropy.md'), file('notes/Heat.pdf')],
    })

    await one.move(['Entropy.md', 'Heat.pdf'], { into: 'notes' })

    expect(done).toStrictEqual([`says ${words.taken} Entropy.md, Heat.pdf`])
  })

  it('read nothing from the vault where nothing was dragged', async () => {
    const { asked, done, one } = tab()

    await one.move([], { into: 'physics' })

    expect(asked).toStrictEqual([])
    expect(done).toStrictEqual([])
  })
})

describe('a folder closed in the tab', () => {
  it('draws nothing under it', async () => {
    const { list, one } = tab()
    await list.openFolder(ROOT)
    one.open('physics')
    await settle()
    expect(list.isFolderOpen('physics')).toBe(true)

    one.close('physics')

    expect(list.isFolderOpen('physics')).toBe(false)
  })
})

describe('a name given to a row, in more detail', () => {
  it('is taken whole for a folder whose name carries a dot', async () => {
    const { done, list, one } = tab(false, { [ROOT]: [...held[ROOT]!, folder('v1.0')] })
    await list.openFolder(ROOT)

    await one.rename('v1.0', 'v2')

    expect(done).toStrictEqual(['moves v1.0 v2'])
  })

  it('is given to a row the tree does not hold, which is taken for a file', async () => {
    const { done, one } = tab()

    await one.rename('ghost/Note.md', 'Other')

    expect(done).toStrictEqual(['moves ghost/Note.md ghost/Other.md'])
  })
})

describe('a file refused in a folder that is shut', () => {
  it.each([
    ['note', NEW_NOTE],
    ['deck', NEW_DECK],
  ])('leaves the folder shut where the %s was refused', async (_kind, id) => {
    const { one, list } = tab(true)
    await list.openFolder(ROOT)
    one.openMenu({ path: 'physics', at: { x: 0, y: 0 } })

    one.chooseMenuItem(id)
    await settle()

    expect(list.isFolderOpen('physics')).toBe(false)
  })

  it('leaves the folder shut where the folder was refused on a row the tree does not hold', async () => {
    const { one, list } = tab(true)
    one.openMenu({ path: 'ghost/Note.md', at: { x: 0, y: 0 } })

    one.chooseMenuItem(NEW_NOTE)
    await settle()

    expect(list.isFolderOpen('ghost')).toBe(false)
  })

  it('makes a note beside a row the tree does not hold', async () => {
    const { one, done } = tab()
    one.openMenu({ path: 'ghost/Note.md', at: { x: 0, y: 0 } })

    one.chooseMenuItem(NEW_NOTE)
    await settle()

    expect(done).toStrictEqual(['writes ghost/Untitled note.md'])
  })
})
