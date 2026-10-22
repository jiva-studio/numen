/**
 * Unit tests for useFilesTab composable and file interactions.
 */
import { describe, expect, it } from 'vitest'
import type { Entry } from '../types'
import { useFilesTab } from './useFilesTab'
import { getFolderPath, ROOT, useFileTree } from './useFileTree'
import { NEW_DECK, NEW_FOLDER, NEW_NOTE, NEW_PRESET, NEW_STENCIL, RENAME } from '../lib/menu'
import { WORDS as words } from '../words'

const file = (path: string, overrides: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  isFolder: false,
  kind: 'note',
  type: 'note',
  ...overrides,
})

const folder = (path: string): Entry => file(path, { isFolder: true, kind: 'other' })

/** A vault of three folders, a note, a book and a picture. */
const target: Record<string, readonly Entry[]> = {
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

/** The window's commands over a row: every one the menu can offer, on any row, and no icons. */
const windowCommands = {
  getPathCommands: () =>
    ['remove', 'transcribe', 'recognise', 'proofread', 'copy'].map((id) => ({
      id,
      text: id,
      group: 'run',
    })),
  getCommandIcon: () => null,
  getEntryIcon: () => ({}),
}

/** A moment for whatever a gesture asked the vault for to come back. */
const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

/** A tab of that vault, writing down everything it asked of the window. */
const tab = (fails = false, extra: Record<string, readonly Entry[]> = {}) => {
  const isDone: string[] = []
  const request: string[] = []
  const sources: string[] = []
  const vault: Record<string, readonly Entry[]> = { ...target, ...extra }
  const list = useFileTree({
    list: async (folderPath: string) => {
      request.push(at)
      return vault[at] ?? []
    },
  })
  const gestures = useFilesTab(list, {
    ...windowCommands,
    openEntry: (entry) => void isDone.push(`lands file ${entry.path}`),
    runCommand: (id, paths, target) => {
      isDone.push(`runs ${id} ${paths.join(' ')} ${target.name}`)
      sources.push(target.source)
    },
    movePath: async (from, to) => void isDone.push(`moves ${from} ${to}`),
    setDraggedPaths: (paths) => void isDone.push(`drags ${paths.join(' ') || '—'}`),
    createFolder: async (path) => {
      isDone.push(`makes ${path}`)
      if (fails) return
      const into = getFolderPath(path)
      vault[into] = [...(vault[into] ?? []), folder(path)]
      vault[path] = []
    },
    createNote: async (folderPath) => {
      const createdItem = folderPath === ROOT ? 'Untitled note.md' : `${folderPath}/Untitled note.md`
      isDone.push(`writes ${createdItem}`)
      if (fails) return ''
      return createdItem
    },
    createDeck: async (folderPath, name) => {
      isDone.push(`decks ${folderPath === ROOT ? '/' : folderPath} ${name}`)
      if (fails) return ''
      const createdItem = folderPath === ROOT ? `${name}.note` : `${folderPath}/${name}.note`
      vault[folderPath] = [...(vault[folderPath] ?? []), file(createdItem, { type: 'deck' })]
      return createdItem
    },
    createStencil: async (folderPath, name) => {
      isDone.push(`stencils ${folderPath === ROOT ? '/' : folderPath} ${name}`)
      if (fails) return ''
      const createdItem = folderPath === ROOT ? `${name}.note` : `${folderPath}/${name}.note`
      vault[folderPath] = [...(vault[folderPath] ?? []), file(createdItem, { type: 'stencil' })]
      return createdItem
    },
    createPreset: async (folderPath, name) => {
      isDone.push(`presets ${folderPath === ROOT ? '/' : folderPath} ${name}`)
      if (fails) return ''
      const createdItem = folderPath === ROOT ? `${name}.note` : `${folderPath}/${name}.note`
      vault[folderPath] = [...(vault[folderPath] ?? []), file(createdItem, { type: 'preset' })]
      return createdItem
    },
    importUrl: async (folderPath, url) => {
      isDone.push(`imports ${folderPath === ROOT ? '/' : folderPath} ${url}`)
      return fails ? '' : 'made.url'
    },
    importUrls: async (folderPath, urls) => {
      isDone.push(`imports-multi ${folderPath === ROOT ? '/' : folderPath} ${urls.join(' ')}`)
      return fails ? [] : urls.map(() => 'made.url')
    },
    showError: (text) => void isDone.push(`says ${text}`),
  })
  return { recorded: isDone, list, one: gestures, requests: request, sources }
}

describe('a row activated', () => {
  it('takes the person to the note it remains for', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    one.activate('Entropy.md')

    expect(recorded).toStrictEqual(['lands file Entropy.md'])
  })

  it('takes the person to the book it remains for', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    one.activate('Heat.pdf')

    expect(recorded).toStrictEqual(['lands file Heat.pdf'])
  })

  it('takes the person to a file the vault holds no source for', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    one.activate('Cover.png')

    expect(recorded).toStrictEqual(['lands file Cover.png'])
  })

  it('leaves it the whole of what is chosen', async () => {
    const { list, one } = tab()
    await list.openFolder(ROOT)

    one.activate('Entropy.md')

    expect(list.selectedPaths.value).toStrictEqual(['Entropy.md'])
  })

  it('does nothing at all for a row the tree does not draw', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    one.activate('physics/Kelvin.md')

    expect(recorded).toStrictEqual([])
  })

  it('opens nothing for a folder, and selects it', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    one.activate('physics')

    expect(recorded).toStrictEqual([])
    expect(list.selectedPaths.value).toStrictEqual(['physics'])
  })
})

describe('what the menu offers', () => {
  it('is what the tab makes off every row', () => {
    expect(
      tab()
        .one.getMenuItems(null)
        .map((one) => one.id),
    ).toStrictEqual(['newNote', 'newDeck', 'newStencil', 'newPreset', 'newFolder'])
  })

  it('is the window’s commands alone over a selection of several', async () => {
    const { list, one } = tab()
    await list.openFolder(ROOT)
    list.selectPaths(['Entropy.md', 'Cover.png'])

    expect(one.getMenuItems('Entropy.md').map((item) => item.id)).not.toContain('rename')
  })

  it('draws a command and an entry with the icons the window gives', () => {
    const { one } = tab()

    expect(one.getCommandIcon('remove')).toBeNull()
    expect(one.getEntryIcon(null, false)).toStrictEqual({})
  })
})

describe('rows dragged out of the tree', () => {
  it('are the note the row remains for', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    one.drag(['Entropy.md'])

    expect(recorded).toStrictEqual(['drags Entropy.md'])
  })

  it('are every note of a selection, in the order the rows were dragged', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)
    await list.openFolder('physics')

    one.drag(['Entropy.md', 'physics/Kelvin.md'])

    expect(recorded).toStrictEqual(['drags Entropy.md physics/Kelvin.md'])
  })

  it('are the notes of a mixed selection, and the rest stay where they are', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    one.drag(['physics', 'Entropy.md', 'Heat.pdf', 'Cover.png'])

    expect(recorded).toStrictEqual(['drags Entropy.md'])
  })

  it('are nothing where the rows hold no note at all', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    one.drag(['physics', 'Heat.pdf', 'Cover.png'])

    expect(recorded).toStrictEqual(['drags —'])
  })

  it('are nothing for a row the tree is no longer drawing', async () => {
    const { recorded, one } = tab()

    one.drag(['Entropy.md'])

    expect(recorded).toStrictEqual(['drags —'])
  })

  it('are nothing once they have been release', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    one.drag(['Entropy.md'])
    one.drop()

    expect(recorded).toStrictEqual(['drags Entropy.md', 'drags —'])
  })
})

describe('rows release', () => {
  it('are filed in the folder they went into, under the names they carry', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    await one.move(['Entropy.md'], { into: 'physics' })

    expect(recorded).toStrictEqual(['moves Entropy.md physics/Entropy.md'])
  })

  it('are filed in the folder holding the row they came before', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)
    await list.openFolder('physics')

    await one.move(['Entropy.md'], { before: 'physics/Kelvin.md' })

    expect(recorded).toStrictEqual(['moves Entropy.md physics/Entropy.md'])
  })

  it('are a folder dragged whole, under the name it carries', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    await one.move(['physics'], { into: 'notes' })

    expect(recorded).toStrictEqual(['moves physics notes/physics'])
  })

  it('all land in the one folder', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    await one.move(['Entropy.md', 'Heat.pdf', 'Cover.png'], { into: 'notes' })

    expect(recorded).toStrictEqual([
      'moves Entropy.md notes/Entropy.md',
      'moves Heat.pdf notes/Heat.pdf',
      'moves Cover.png notes/Cover.png',
    ])
  })

  it('read the folders again once, when all of them are done', async () => {
    const request: string[] = []
    const list = useFileTree({
      list: async (folderPath: string) => {
        request.push(at)
        return target[at] ?? []
      },
    })
    const one = useFilesTab(list, {
      ...windowCommands,
      openEntry: () => {},
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
    request.length = 0

    await one.move(['Entropy.md', 'Heat.pdf'], { into: 'notes' })

    expect(request.filter((at) => at === ROOT)).toStrictEqual([ROOT])
  })

  it('leave behind the one whose name that folder holds, and say which', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    await one.move(['Entropy.md', 'Heat.pdf'], { into: 'heat' })

    expect(recorded).toStrictEqual(['moves Heat.pdf heat/Heat.pdf', `says ${words.taken} Entropy.md`])
  })

  it('ask nothing where they landed where they already were', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)
    await list.openFolder('physics')

    await one.move(['physics/Kelvin.md'], { into: 'physics' })

    expect(recorded).toStrictEqual([])
  })

  it('ask nothing at all where nothing was dragged', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    await one.move([], { into: 'physics' })

    expect(recorded).toStrictEqual([])
  })
})

describe('rows requested to go', () => {
  it('are handed to the window as one command over all of them', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    one.remove(['Entropy.md', 'Cover.png'])

    expect(recorded).toStrictEqual(['runs remove Entropy.md Cover.png Entropy.md'])
  })

  it('ask for nothing where nothing is selected', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    one.remove([])

    expect(recorded).toStrictEqual([])
  })
})

describe('a name given to a row', () => {
  it('moves the file within the folder it is in', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    await one.rename('Entropy.md', 'Order.md')

    expect(recorded).toStrictEqual(['moves Entropy.md Order.md'])
  })

  it('asks the vault for nothing where the name is the one it carries', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    await one.rename('Entropy.md', 'Entropy.md')

    expect(recorded).toStrictEqual([])
  })

  it('keeps the ending the file carries, so a note typed over stays a note', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    await one.rename('Entropy.md', 'Order')

    expect(recorded).toStrictEqual(['moves Entropy.md Order.md'])
  })

  it('takes what was typed for a folder', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    await one.rename('physics', 'heat')

    expect(recorded).toStrictEqual(['moves physics heat'])
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
    const { recorded, one } = await openRowMenu()

    one.chooseMenuItem(RENAME)

    expect(one.renamingPath.value).toBe('Entropy.md')
    expect(recorded).toStrictEqual([])
  })

  it('makes a folder alongside the row, under a name nothing there carries', async () => {
    const { recorded, one } = await openRowMenu()

    one.chooseMenuItem(NEW_FOLDER)
    await settle()

    expect(recorded).toStrictEqual(['makes New folder'])
  })

  it('makes a folder inside the row where the row is a folder', async () => {
    const { recorded, one } = await openRowMenu('physics')

    one.chooseMenuItem(NEW_FOLDER)
    await settle()

    expect(recorded).toStrictEqual(['makes physics/New folder'])
  })

  it('puts the name of a folder it made in a field', async () => {
    const { one } = await openRowMenu()

    one.chooseMenuItem(NEW_FOLDER)
    await settle()

    expect(one.renamingPath.value).toBe('New folder')
  })

  it('puts no name in a field where the folder failed', async () => {
    const { one } = await openRowMenu('Entropy.md', true)

    one.chooseMenuItem(NEW_FOLDER)
    await settle()

    expect(one.renamingPath.value).toBeNull()
  })

  it('makes a note alongside the row, and puts its name in a field', async () => {
    const { recorded, one } = await openRowMenu()

    one.chooseMenuItem(NEW_NOTE)
    await settle()

    expect(recorded).toStrictEqual(['writes Untitled note.md'])
    expect(one.renamingPath.value).toBe('Untitled note.md')
  })

  it('makes a deck alongside the row, and puts its name in a field', async () => {
    const { recorded, one } = await openRowMenu()

    one.chooseMenuItem(NEW_DECK)
    await settle()

    expect(recorded).toStrictEqual([`decks / ${words.newDeck}`])
    expect(one.renamingPath.value).toBe(`${words.newDeck}.note`)
  })

  it('asks under the name alone, putting no ending on it', async () => {
    const { recorded, one } = await openRowMenu()

    one.chooseMenuItem(NEW_DECK)
    await settle()

    expect(recorded[0]).not.toContain('.md')
  })

  it('makes a stencil the same way, under a name of its own', async () => {
    const { recorded, one } = await openRowMenu()

    one.chooseMenuItem(NEW_STENCIL)
    await settle()

    expect(recorded).toStrictEqual([`stencils / ${words.newStencil}`])
  })

  it('makes a preset the same way, under a name of its own', async () => {
    const { recorded, one } = await openRowMenu()

    one.chooseMenuItem(NEW_PRESET)
    await settle()

    expect(recorded).toStrictEqual([`presets / ${words.newPreset}`])
    expect(one.renamingPath.value).toBe(`${words.newPreset}.note`)
  })

  it('remains the preset in the tree as a preset', async () => {
    const { one } = await openRowMenu()

    one.chooseMenuItem(NEW_PRESET)
    await settle()

    expect(one.list.getEntryAt(`${words.newPreset}.note`)?.type).toBe('preset')
  })

  it('makes a preset inside the row where the row is a folder', async () => {
    const { recorded, one } = await openRowMenu('physics')

    one.chooseMenuItem(NEW_PRESET)
    await settle()

    expect(recorded).toStrictEqual([`presets physics ${words.newPreset}`])
  })

  it('is empty where the vault made no preset', async () => {
    const { one } = await openRowMenu(undefined, true)

    one.chooseMenuItem(NEW_PRESET)
    await settle()

    expect(one.renamingPath.value).toBeNull()
  })

  it('makes a deck inside the row where the row is a folder', async () => {
    const { recorded, one } = await openRowMenu('physics')

    one.chooseMenuItem(NEW_DECK)
    await settle()

    expect(recorded).toStrictEqual([`decks physics ${words.newDeck}`])
  })

  it('is empty where the vault made no deck', async () => {
    const { one } = await openRowMenu(undefined, true)

    one.chooseMenuItem(NEW_DECK)
    await settle()

    expect(one.renamingPath.value).toBeNull()
  })

  it('makes a note inside the row where the row is a folder', async () => {
    const { recorded, one } = await openRowMenu('physics')

    one.chooseMenuItem(NEW_NOTE)
    await settle()

    expect(recorded).toStrictEqual(['writes physics/Untitled note.md'])
  })

  it('makes a note and a folder at the root, requested off every row', async () => {
    const { recorded, one } = await openRowMenu(null)

    one.chooseMenuItem(NEW_NOTE)
    await settle()
    one.openMenu({ path: null, at: { x: 0, y: 0 } })
    one.chooseMenuItem(NEW_FOLDER)
    await settle()

    expect(recorded).toStrictEqual(['writes Untitled note.md', 'makes New folder'])
  })

  it('does not rename where the menu was requested off every row', async () => {
    const { one } = await openRowMenu(null)

    one.chooseMenuItem(RENAME)

    expect(one.renamingPath.value).toBeNull()
  })

  it('passes a command over the file to the window', async () => {
    const { recorded, one } = await openRowMenu()

    one.chooseMenuItem('remove')

    expect(recorded).toStrictEqual(['runs remove Entropy.md Entropy.md'])
  })

  it('passes the whole selection to the window, on a row standing in it', async () => {
    const heldState = await openRowMenu()
    heldState.list.selectPaths(['Entropy.md', 'Cover.png'])

    heldState.one.chooseMenuItem('remove')

    expect(heldState.recorded).toStrictEqual(['runs remove Entropy.md Cover.png Entropy.md'])
  })

  it('passes the one row to the window, on a row standing outside the selection', async () => {
    const heldState = await openRowMenu()
    heldState.list.selectPaths(['Cover.png'])

    heldState.one.chooseMenuItem('remove')

    expect(heldState.recorded).toStrictEqual(['runs remove Entropy.md Entropy.md'])
  })

  it('does nothing at all for a choice the menu does not offer', async () => {
    const { recorded, one } = await openRowMenu()

    one.chooseMenuItem('destroyEverything')

    expect(recorded).toStrictEqual([])
  })

  it('puts the menu away whatever was chosen', async () => {
    const { one } = await openRowMenu()

    one.chooseMenuItem('remove')

    expect(one.menu.value).toBeNull()
  })
})

describe('useFilesTab helper and interaction methods', () => {
  it('covers folder operations, selections, renaming and dismiss', async () => {
    const { list, one, recorded } = tab()
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
    expect(recorded).toContain('imports physics https://example.com/video')

    one.importUrls(['https://example.com/1', 'https://example.com/2'])
    expect(recorded).toContain('imports-multi physics https://example.com/1 https://example.com/2')

    expect(one.getNameOf('Entropy.md')).toBe('Entropy.md')
    expect(one.getNameOf('nonexistent/file.txt')).toBe('file.txt')
    expect(one.getFolderFor(null)).toBe(ROOT)
  })
})

describe('a field already holding a name, and a file rejected', () => {
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
  ])('leaves the field alone where the %s failed', async (_kind, id) => {
    expect(await refuse(id)).toBe('Cover.png')
  })
})

describe('a menu item chosen off every row', () => {
  it('leaves the field alone and runs nothing for an item that needs a row', async () => {
    const { recorded, one } = tab()
    one.setRenamingPath('Cover.png')

    one.openMenu({ path: null, at: { x: 0, y: 0 } })
    one.chooseMenuItem(RENAME)
    one.openMenu({ path: null, at: { x: 0, y: 0 } })
    one.chooseMenuItem('transcribe')

    expect(one.renamingPath.value).toBe('Cover.png')
    expect(recorded).toStrictEqual([])
  })
})

describe('the source a command is run over', () => {
  it('is the kind of file the row remains for', async () => {
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

describe('rows release, in more detail', () => {
  it('are filed at the top of the vault under the names they carry', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)
    await list.openFolder('physics')

    await one.move(['physics/Kelvin.md'], { into: null })

    expect(recorded).toStrictEqual(['moves physics/Kelvin.md Kelvin.md'])
  })

  it('leave behind the second of two that carry the one name', async () => {
    const { recorded, one } = tab()

    await one.move(['heat/Entropy.md', 'physics/Entropy.md'], { into: 'notes' })

    expect(recorded).toStrictEqual([
      'moves heat/Entropy.md notes/Entropy.md',
      `says ${words.taken} Entropy.md`,
    ])
  })

  it('formats remaining names as comma-separated list', async () => {
    const { recorded, one } = tab(false, {
      notes: [file('notes/Entropy.md'), file('notes/Heat.pdf')],
    })

    await one.move(['Entropy.md', 'Heat.pdf'], { into: 'notes' })

    expect(recorded).toStrictEqual([`says ${words.taken} Entropy.md, Heat.pdf`])
  })

  it('read nothing from the vault where nothing was dragged', async () => {
    const { requests, recorded, one } = tab()

    await one.move([], { into: 'physics' })

    expect(requests).toStrictEqual([])
    expect(recorded).toStrictEqual([])
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
    const { recorded, list, one } = tab(false, { [ROOT]: [...target[ROOT]!, folder('v1.0')] })
    await list.openFolder(ROOT)

    await one.rename('v1.0', 'v2')

    expect(recorded).toStrictEqual(['moves v1.0 v2'])
  })

  it('is given to a row the tree does not hold, which is taken for a file', async () => {
    const { recorded, one } = tab()

    await one.rename('ghost/Note.md', 'Other')

    expect(recorded).toStrictEqual(['moves ghost/Note.md ghost/Other.md'])
  })
})

describe('a file rejected in a folder that is shut', () => {
  it.each([
    ['note', NEW_NOTE],
    ['deck', NEW_DECK],
  ])('leaves the folder shut where the %s failed', async (_kind, id) => {
    const { one, list } = tab(true)
    await list.openFolder(ROOT)
    one.openMenu({ path: 'physics', at: { x: 0, y: 0 } })

    one.chooseMenuItem(id)
    await settle()

    expect(list.isFolderOpen('physics')).toBe(false)
  })

  it('leaves the folder shut where the folder failed on a row the tree does not hold', async () => {
    const { one, list } = tab(true)
    one.openMenu({ path: 'ghost/Note.md', at: { x: 0, y: 0 } })

    one.chooseMenuItem(NEW_NOTE)
    await settle()

    expect(list.isFolderOpen('ghost')).toBe(false)
  })

  it('makes a note alongside a row the tree does not hold', async () => {
    const { one, recorded } = tab()
    one.openMenu({ path: 'ghost/Note.md', at: { x: 0, y: 0 } })

    one.chooseMenuItem(NEW_NOTE)
    await settle()

    expect(recorded).toStrictEqual(['writes ghost/Untitled note.md'])
  })
})
