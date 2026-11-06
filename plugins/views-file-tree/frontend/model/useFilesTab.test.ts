/**
 * Unit tests for useFilesTab composable and file interactions.
 */
import { describe, expect, it } from 'vitest'
import type { Entry } from '../types'
import { useFilesTab } from './useFilesTab'
import { getFolderPath, ROOT, createFileTree } from './useFileTree'
import type { CommandTarget } from '@numen/plugin-sdk'
import { NEW_MENU, ROW_MENU, TRASH } from '../fixtures/host'
import { WORDS as words } from '../words'

const file = (path: string, overrides: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  isFolder: false,
  isNote: true,
  type: 'note',
  ...overrides,
})

const folder = (path: string): Entry => file(path, { isFolder: true, isNote: false })

/** A vault of three folders, a note, a book and a picture. */
const target: Record<string, readonly Entry[]> = {
  [ROOT]: [
    folder('physics'),
    folder('notes'),
    folder('heat'),
    file('Entropy.md'),
    file('Heat.pdf', { isNote: false }),
    file('Cover.png', { isNote: false }),
  ],
  physics: [file('physics/Kelvin.md')],
  notes: [],
  heat: [file('heat/Entropy.md')],
}

/** What the window draws the rows with. */
const windowCommands = {
  getEntryIcon: () => ({}),
}

/** What a run of a command records: its id, the paths it is over and the title of the target. */
const describeRun = (command: string, target: CommandTarget): string =>
  `runs ${command} ${[target.path, ...target.others].join(' ')} ${target.title}`

/** Waits for the vault requests issued by a gesture to complete. */
const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

/** A tab of that vault, recording everything it requested from the window. */
const tab = (fails = false, extra: Record<string, readonly Entry[]> = {}) => {
  const isDone: string[] = []
  const request: string[] = []
  const targets: CommandTarget[] = []
  const runs: CommandTarget[] = []
  const vault: Record<string, readonly Entry[]> = { ...target, ...extra }
  const list = createFileTree({
    list: async (folderPath: string) => {
      request.push(folderPath)
      return vault[folderPath] ?? []
    },
  })
  const gestures = useFilesTab(list, {
    ...windowCommands,
    openEntry: (entry) => {
      isDone.push(`lands file ${entry.path}`)
    },
    listItems: async () => [],
    getMenuEntries: (menuTarget) => {
      targets.push(menuTarget)
      return ROW_MENU
    },
    runCommand: async (command, menuTarget) => {
      runs.push(menuTarget)
      isDone.push(describeRun(command, menuTarget))
      if (fails) throw new Error('the window is gone')
    },
    movePath: async (from, to) => {
      isDone.push(`moves ${from} ${to}`)
    },
    setDraggedPaths: (paths) => {
      isDone.push(`drags ${paths.join(' ') || '—'}`)
    },
    createFolder: async (path) => {
      isDone.push(`makes ${path}`)
      if (fails) return
      const into = getFolderPath(path)
      vault[into] = [...(vault[into] ?? []), folder(path)]
      vault[path] = []
    },
    createNote: async (folderPath) => {
      const createdItem =
        folderPath === ROOT ? 'Untitled note.md' : `${folderPath}/Untitled note.md`
      isDone.push(`writes ${createdItem}`)
      if (fails) return ''
      return createdItem
    },
    canDropData: (types) => types.includes('text/uri-list'),
    dropData: async (folderPath, data) => {
      isDone.push(`drops into ${folderPath === ROOT ? '/' : folderPath} ${data['text/uri-list']}`)
      return []
    },
    showError: (text) => {
      isDone.push(`says ${text}`)
    },
  })
  return { recorded: isDone, list, one: gestures, requests: request, targets, runs }
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
  it('is the entries of the slot over the row it was requested on', async () => {
    const { list, one, targets } = tab()
    await list.openFolder(ROOT)

    const items = one.getMenuItems('Entropy.md')

    expect(items.map((item) => item.id)).toContain(TRASH.key)
    expect(targets).toStrictEqual([
      {
        path: 'Entropy.md',
        title: 'Entropy.md',
        isFolder: false,
        others: [],
        tab: '',
        noteType: 'note',
      },
    ])
  })

  it('asks for the entries over a note with the type the listing gives it', async () => {
    const { list, one, targets } = tab(false, {
      [ROOT]: [file('Pier.md', { type: 'link' }), file('Cover.png', { isNote: false })],
    })
    await list.openFolder(ROOT)

    one.getMenuItems('Pier.md')
    one.getMenuItems('Cover.png')

    expect(targets.map((one) => one.noteType)).toStrictEqual(['link', undefined])
  })

  it('asks for the entries over a folder as a folder', async () => {
    const { list, one, targets } = tab()
    await list.openFolder(ROOT)

    one.getMenuItems('physics')

    expect(targets).toStrictEqual([
      { path: 'physics', title: 'physics', isFolder: true, others: [], tab: '' },
    ])
  })

  it('asks for the entries over the root where the menu was requested off every row', () => {
    const { one, targets } = tab()

    one.getMenuItems(null)

    expect(targets).toStrictEqual([{ path: ROOT, title: '', isFolder: true, others: [], tab: '' }])
  })

  it('asks for the entries over the row and the others selected with it', async () => {
    const { list, one, targets } = tab()
    await list.openFolder(ROOT)
    list.selectPaths(['Entropy.md', 'Cover.png'])

    one.getMenuItems('Entropy.md')

    expect(targets).toStrictEqual([
      {
        path: 'Entropy.md',
        title: 'Entropy.md',
        isFolder: false,
        others: ['Cover.png'],
        tab: '',
        noteType: 'note',
      },
    ])
  })

  it('draws a command with the icon of its entry, and an entry of the vault with the window’s', async () => {
    const { list, one } = tab()
    await list.openFolder(ROOT)
    one.openMenu({ path: 'Entropy.md', at: { x: 0, y: 0 } })

    expect(one.getMenuIcon(ROW_MENU[0]!.key)).toBe((ROW_MENU[0] as { icon: unknown }).icon)
    expect(one.getMenuIcon(TRASH.key)).toBeNull()
    expect(one.getEntryIcon(null, false)).toStrictEqual({})
  })

  it('draws the component of a view entry in its row', async () => {
    const { list, one } = tab()
    await list.openFolder(ROOT)
    one.openMenu({ path: 'Entropy.md', at: { x: 0, y: 0 } })
    const view = ROW_MENU.find((entry) => entry.kind === 'view')!
    const row = one.getMenuItems('Entropy.md').find((item) => item.id === view.key)

    expect(row?.component).toBe((view as { component: unknown }).component)
    expect(one.getMenuItems('Entropy.md').find((item) => item.id === TRASH.key)?.component).toBe(
      undefined,
    )
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
    const list = createFileTree({
      list: async (folderPath: string) => {
        request.push(folderPath)
        return target[folderPath] ?? []
      },
    })
    const one = useFilesTab(list, {
      ...windowCommands,
      openEntry: () => undefined,
      listItems: async () => [],
      getMenuEntries: () => [],
      runCommand: async () => undefined,
      movePath: async () => undefined,
      setDraggedPaths: () => undefined,
      createFolder: async () => undefined,
      createNote: async () => '',
      canDropData: () => false,
      dropData: async () => [],
      showError: () => undefined,
    })
    await list.openFolder(ROOT)
    request.length = 0

    await one.move(['Entropy.md', 'Heat.pdf'], { into: 'notes' })

    expect(request.filter((itemPath) => itemPath === ROOT)).toStrictEqual([ROOT])
  })

  it('leave behind the one whose name that folder holds, and say which', async () => {
    const { recorded, list, one } = tab()
    await list.openFolder(ROOT)

    await one.move(['Entropy.md', 'Heat.pdf'], { into: 'heat' })

    expect(recorded).toStrictEqual([
      'moves Heat.pdf heat/Heat.pdf',
      `says ${words.taken} Entropy.md`,
    ])
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

    expect(recorded).toStrictEqual(['runs app:note.trash Entropy.md Cover.png Entropy.md'])
  })

  it('shows the failure of the command the window runs', async () => {
    const { recorded, list, one } = tab(true)
    await list.openFolder(ROOT)

    one.remove(['Entropy.md'])
    await settle()

    expect(recorded.at(-1)).toBe('says Numen is not responding; retrying')
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
    const pendingState = tab(fails)
    await pendingState.list.openFolder(ROOT)
    pendingState.one.openMenu({ path, at: { x: 0, y: 0 } })
    return pendingState
  }

  it('runs the command of the entry over the row', async () => {
    const { recorded, one } = await openRowMenu()

    one.chooseMenuItem(TRASH.key)

    expect(recorded).toStrictEqual(['runs app:note.trash Entropy.md Entropy.md'])
  })

  it('runs the command of an entry inside a submenu', async () => {
    const { recorded, one } = await openRowMenu('physics')

    one.chooseMenuItem(NEW_MENU.entries[1]!.key)

    expect(recorded).toStrictEqual(['runs md.numen.views.file.tree:tree.newFolder physics physics'])
  })

  it('runs the command over the root where the menu was requested off every row', async () => {
    const { recorded, one } = await openRowMenu(null)

    one.chooseMenuItem(NEW_MENU.entries[0]!.key)

    expect(recorded).toStrictEqual(['runs md.numen.views.file.tree:tree.newNote  '])
  })

  it('passes the whole selection, on a row standing in it', async () => {
    const pendingState = await openRowMenu()
    pendingState.list.selectPaths(['Entropy.md', 'Cover.png'])

    pendingState.one.chooseMenuItem(TRASH.key)

    expect(pendingState.recorded).toStrictEqual([
      'runs app:note.trash Entropy.md Cover.png Entropy.md',
    ])
  })

  it('passes the one row, on a row standing outside the selection', async () => {
    const pendingState = await openRowMenu()
    pendingState.list.selectPaths(['Cover.png'])

    pendingState.one.chooseMenuItem(TRASH.key)

    expect(pendingState.recorded).toStrictEqual(['runs app:note.trash Entropy.md Entropy.md'])
  })

  it('runs nothing for a submenu, a view, or a choice the menu does not offer', async () => {
    const { recorded, one } = await openRowMenu()

    one.chooseMenuItem(NEW_MENU.key)
    one.openMenu({ path: 'Entropy.md', at: { x: 0, y: 0 } })
    one.chooseMenuItem(ROW_MENU.find((entry) => entry.kind === 'view')!.key)
    one.openMenu({ path: 'Entropy.md', at: { x: 0, y: 0 } })
    one.chooseMenuItem('destroyEverything')

    expect(recorded).toStrictEqual([])
  })

  it('shows the failure of the command it ran', async () => {
    const { recorded, one } = await openRowMenu('Entropy.md', true)

    one.chooseMenuItem(TRASH.key)
    await settle()

    expect(recorded).toStrictEqual([
      'runs app:note.trash Entropy.md Entropy.md',
      'says Numen is not responding; retrying',
    ])
  })

  it('puts the menu away whatever was chosen', async () => {
    const { one } = await openRowMenu()

    one.chooseMenuItem(TRASH.key)

    expect(one.menu.value).toBeNull()
  })

  it('runs nothing where no menu is open', () => {
    const { recorded, one } = tab()

    one.chooseMenuItem(TRASH.key)

    expect(recorded).toStrictEqual([])
  })
})

describe('a folder or a note made at a row', () => {
  const setUp = async (fails = false) => {
    const pendingState = tab(fails)
    await pendingState.list.openFolder(ROOT)
    return pendingState
  }

  it('makes a folder alongside the row, under a name nothing there carries', async () => {
    const { recorded, one } = await setUp()

    await one.createFolder('Entropy.md')

    expect(recorded).toStrictEqual(['makes New folder'])
  })

  it('makes a folder inside the row where the row is a folder', async () => {
    const { recorded, one } = await setUp()

    await one.createFolder('physics')

    expect(recorded).toStrictEqual(['makes physics/New folder'])
  })

  it('puts the name of a folder it made in a field', async () => {
    const { one } = await setUp()

    await one.createFolder('Entropy.md')

    expect(one.renamingPath.value).toBe('New folder')
  })

  it('puts no name in a field where the folder failed', async () => {
    const { one } = await setUp(true)

    await one.createFolder('Entropy.md')

    expect(one.renamingPath.value).toBeNull()
  })

  it('makes a note alongside the row, and puts its name in a field', async () => {
    const { recorded, one } = await setUp()

    await one.createNote('Entropy.md')

    expect(recorded).toStrictEqual(['writes Untitled note.md'])
    expect(one.renamingPath.value).toBe('Untitled note.md')
  })

  it('makes a note inside the row where the row is a folder', async () => {
    const { recorded, one } = await setUp()

    await one.createNote('physics')

    expect(recorded).toStrictEqual(['writes physics/Untitled note.md'])
  })

  it('makes a note and a folder at the root', async () => {
    const { recorded, one } = await setUp()

    await one.createNote(null)
    await one.createFolder(null)

    expect(recorded).toStrictEqual(['writes Untitled note.md', 'makes New folder'])
  })
})

describe('useFilesTab helper and interaction methods', () => {
  it('covers folder operations, selections, renaming and dismiss', async () => {
    const { list, one } = tab()
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

    expect(one.getNameOf('Entropy.md')).toBe('Entropy.md')
    expect(one.getNameOf('nonexistent/file.txt')).toBe('file.txt')
    expect(one.getFolderFor(null)).toBe(ROOT)
  })
})

describe('a field already holding a name, and a file rejected', () => {
  const refuse = async (create: 'createFolder' | 'createNote') => {
    const { one, list } = tab(true)
    await list.openFolder(ROOT)
    one.setRenamingPath('Cover.png')
    await one[create]('Entropy.md')
    return one.renamingPath.value
  }

  it.each([
    ['folder', 'createFolder'],
    ['note', 'createNote'],
  ] as const)('leaves the field alone where the %s failed', async (_kind, create) => {
    expect(await refuse(create)).toBe('Cover.png')
  })
})

describe('the target a command of the menu runs over', () => {
  it('says whether the row is a folder', async () => {
    const { list, one, runs } = tab()
    await list.openFolder(ROOT)

    one.openMenu({ path: 'physics', at: { x: 0, y: 0 } })
    one.chooseMenuItem(TRASH.key)
    one.openMenu({ path: 'Heat.pdf', at: { x: 0, y: 0 } })
    one.chooseMenuItem(TRASH.key)

    expect(runs.map((one) => one.isFolder)).toStrictEqual([true, false])
  })

  it('is a file named after its path for a row the tree does not hold', () => {
    const { one, recorded } = tab()

    one.openMenu({ path: 'ghost/Scan.bin', at: { x: 0, y: 0 } })
    one.chooseMenuItem(TRASH.key)

    expect(recorded).toStrictEqual(['runs app:note.trash ghost/Scan.bin Scan.bin'])
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
  it('leaves the folder shut where the note failed', async () => {
    const { one, list } = tab(true)
    await list.openFolder(ROOT)

    await one.createNote('physics')

    expect(list.isFolderOpen('physics')).toBe(false)
  })

  it('leaves the folder shut where the note failed on a row the tree does not hold', async () => {
    const { one, list } = tab(true)

    await one.createNote('ghost/Note.md')

    expect(list.isFolderOpen('ghost')).toBe(false)
  })

  it('makes a note alongside a row the tree does not hold', async () => {
    const { one, recorded } = tab()

    await one.createNote('ghost/Note.md')

    expect(recorded).toStrictEqual(['writes ghost/Untitled note.md'])
  })
})
