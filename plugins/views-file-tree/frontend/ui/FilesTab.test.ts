/**
 * A files tab drawn, in a document.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { Menu, Tree } from '@numen/ui'
import type { Entry } from '../types'
import FilesTab from './FilesTab.vue'
import { useFilesTab, type FilesTabState } from '../model/useFilesTab'
import { createFileTree, ROOT } from '../model/useFileTree'
import { ASK, NEW_MENU, OPEN, renderDeps, ROW_MENU, TRASH } from '../fixtures/host'

enableAutoUnmount(afterEach)

const file = (path: string, overrides: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  isFolder: false,
  isNote: true,
  type: 'note',
  ...overrides,
})

const folder = (path: string): Entry => file(path, { isFolder: true, isNote: false })

const target: Record<string, readonly Entry[]> = {
  [ROOT]: [folder('physics'), file('Entropy.md'), file('Cover.png', { isNote: false })],
  physics: [file('physics/Kelvin.md')],
}

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

const mountFiles = async (open: readonly string[] = []) => {
  const isDone: string[] = []
  const list = createFileTree({ list: async (folderPath: string) => target[folderPath] ?? [] })
  const tab: FilesTabState = useFilesTab(list, {
    ...renderDeps,
    openEntry: (entry) => {
      isDone.push(`lands file ${entry.path}`)
    },
    listItems: async () => [],
    getMenuEntries: () => ROW_MENU,
    runCommand: async (command, rowTarget) => {
      isDone.push(
        `runs ${command} ${[rowTarget.path, ...rowTarget.others].join(' ')} ${rowTarget.title}`,
      )
    },
    movePath: async (from, to) => {
      isDone.push(`moves ${from} ${to}`)
    },
    setDraggedPaths: (paths) => {
      isDone.push(`drags ${paths.join(' ') || '—'}`)
    },
    createFolder: async (path) => {
      isDone.push(`makes ${path}`)
    },
    createNote: async (folderPath) => `${folderPath}Untitled note.md`,
    canDropData: (types) => types.includes('text/uri-list'),
    dropData: async (folderPath, data) => {
      isDone.push(`drops into ${folderPath || '/'} ${data['text/uri-list']}`)
      return []
    },
    showError: (text) => {
      isDone.push(`says ${text}`)
    },
  })
  await list.openFolder(ROOT)
  for (const folderPath of open) await list.openFolder(folderPath)
  const window = mount(FilesTab, { props: { state: tab } })
  await settle()
  return { recorded: isDone, list, tab, window }
}

const getRowIds = (rows: readonly { id: string; rows?: readonly unknown[] }[]): unknown =>
  rows.map((one) => [one.id, getRowIds((one.rows ?? []) as never)])

describe('the tree the tab draws', () => {
  it('is handed a row per file the vault holds, under the path it is filed at', async () => {
    const { window } = await mountFiles()

    expect(getRowIds(window.findComponent(Tree).props('rows'))).toStrictEqual([
      ['physics', []],
      ['Entropy.md', []],
      ['Cover.png', []],
    ])
  })

  it('is handed what an open folder holds, inside the row that remains for it', async () => {
    const { window } = await mountFiles(['physics'])

    expect(getRowIds(window.findComponent(Tree).props('rows'))).toStrictEqual([
      ['physics', [['physics/Kelvin.md', []]]],
      ['Entropy.md', []],
      ['Cover.png', []],
    ])
  })

  it('says which folders are open, and names no folder that is closed', async () => {
    const { window } = await mountFiles()

    expect(window.findComponent(Tree).props('open')).not.toContain('physics')
  })

  it('draws an icon of its own alongside every row', async () => {
    const { window } = await mountFiles()

    expect(window.findAll('.files__icon')).toHaveLength(3)
  })

  it('names the folder a file dragged in from outside is filed in', async () => {
    const { window } = await mountFiles(['physics'])
    const marking = window.findComponent(Tree).props('marking') as {
      attribute: string
      getAttributeValue: (row: string | null) => string
    }

    expect(marking.attribute).toBe('data-file-drop-target')
    expect(marking.getAttributeValue('physics')).toBe('physics')
    expect(marking.getAttributeValue('physics/Kelvin.md')).toBe('physics')
    expect(marking.getAttributeValue('Entropy.md')).toBe(ROOT)
    expect(marking.getAttributeValue(null)).toBe(ROOT)
  })
})

describe('a row the tree reports', () => {
  it('takes the person to the note it remains for', async () => {
    const { recorded, window } = await mountFiles()

    window.findComponent(Tree).vm.$emit('activate', 'Entropy.md')
    await settle()

    expect(recorded).toStrictEqual(['lands file Entropy.md'])
  })

  it('takes the person to a file the vault holds no source for', async () => {
    const { recorded, window } = await mountFiles()

    window.findComponent(Tree).vm.$emit('activate', 'Cover.png')
    await settle()

    expect(recorded).toStrictEqual(['lands file Cover.png'])
  })

  it('is filed where it was release', async () => {
    const { recorded, window } = await mountFiles()

    window.findComponent(Tree).vm.$emit('move', ['Entropy.md'], { into: 'physics' })
    await settle()

    expect(recorded).toStrictEqual(['moves Entropy.md physics/Entropy.md'])
  })

  it('is one of several filed where they were all release', async () => {
    const { recorded, window } = await mountFiles()

    window.findComponent(Tree).vm.$emit('move', ['Entropy.md', 'Cover.png'], { into: 'physics' })
    await settle()

    expect(recorded).toStrictEqual([
      'moves Entropy.md physics/Entropy.md',
      'moves Cover.png physics/Cover.png',
    ])
  })

  it('is one of the rows the tree returns as chosen', async () => {
    const { list, window } = await mountFiles()

    window.findComponent(Tree).vm.$emit('select', ['Entropy.md', 'Cover.png'])
    await settle()

    expect(list.selectedPaths.value).toStrictEqual(['Entropy.md', 'Cover.png'])
    expect(window.findComponent(Tree).props('selected')).toStrictEqual(['Entropy.md', 'Cover.png'])
  })

  it('is one of the rows requested to go, handed to the window as one command', async () => {
    const { recorded, window } = await mountFiles()

    window.findComponent(Tree).vm.$emit('remove', ['Entropy.md', 'Cover.png'])
    await settle()

    expect(recorded).toStrictEqual(['runs app:note.trash Entropy.md Cover.png Entropy.md'])
  })

  it('is filed under the name that was typed over it', async () => {
    const { recorded, window } = await mountFiles()

    window.findComponent(Tree).vm.$emit('rename', 'Entropy.md', 'Order.md')
    await settle()

    expect(recorded).toStrictEqual(['moves Entropy.md Order.md'])
  })

  it('opens the folder it remains for, and its contents are drawn', async () => {
    const { window } = await mountFiles()

    window.findComponent(Tree).vm.$emit('open', 'physics')
    await settle()

    expect(window.findComponent(Tree).props('open')).toContain('physics')
  })
})

describe('the menu on a row', () => {
  it('is drawn nowhere until a row asks for one', async () => {
    const { window } = await mountFiles()

    expect(window.findComponent(Menu).exists()).toBe(false)
  })

  const openMenu = async (path: string | null, selection: readonly string[] = []) => {
    const mounted = await mountFiles()
    if (selection.length) mounted.list.selectPaths(selection)
    mounted.window.findComponent(Tree).vm.$emit('menu', path, { x: 4, y: 8 })
    await settle()
    return mounted
  }

  const getItems = (window: Awaited<ReturnType<typeof mountFiles>>['window']) =>
    window.findComponent(Menu).props('items') as readonly {
      id: string
      group?: string
      children?: readonly { id: string }[]
    }[]

  it('draws a row per entry of the slot, in the order of the slot', async () => {
    const { window } = await openMenu('Entropy.md')

    expect(getItems(window).map((one) => one.id)).toStrictEqual(
      ROW_MENU.filter((one) => one.kind !== 'separator').map((one) => one.key),
    )
  })

  it('draws the groups the separators of the slot divide', async () => {
    const { window } = await openMenu('Entropy.md')

    expect(getItems(window).map((one) => one.group)).toStrictEqual([
      '0',
      '1',
      '2',
      '2',
      '3',
      '3',
      '4',
    ])
  })

  it('draws a submenu with its own entries', async () => {
    const { window } = await openMenu('Entropy.md')

    expect(
      getItems(window)
        .find((one) => one.id === NEW_MENU.key)
        ?.children?.map((one) => one.id),
    ).toStrictEqual(NEW_MENU.entries.map((one) => one.key))
  })

  it('draws a command with the icon of its entry', async () => {
    await openMenu('Entropy.md')

    const rowOf = (text: string) =>
      [...document.body.querySelectorAll('[role="menuitem"]')].find((one) =>
        one.textContent?.includes(text),
      )
    expect(rowOf(OPEN.title)?.querySelector('svg')).not.toBeNull()
    expect(rowOf(ASK.title)?.querySelector('svg')).toBeNull()
  })

  it('draws the component of a view in its row', async () => {
    await openMenu('Entropy.md')

    expect(document.body.querySelector('[role="menuitem"] em')?.textContent).toBe('sketch')
  })

  it('runs the command of the chosen entry over the row', async () => {
    const { recorded, window } = await openMenu('Entropy.md', ['Entropy.md', 'Cover.png'])

    window.findComponent(Menu).vm.$emit('choose', TRASH.key)
    await settle()

    expect(recorded).toStrictEqual(['runs app:note.trash Entropy.md Cover.png Entropy.md'])
    expect(window.findComponent(Menu).exists()).toBe(false)
  })
})

describe('a folder that could not be read', () => {
  it('is returned in the tab', async () => {
    const list = createFileTree({
      list: async () => {
        throw new Error('the vault is not there')
      },
    })
    const tab: FilesTabState = useFilesTab(list, {
      ...renderDeps,
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

    const window = mount(FilesTab, { props: { state: tab } })
    await settle()

    expect(window.find('.caution').text()).toContain('Numen is not responding')
  })

  it('is returned nowhere while the vault answers', async () => {
    const { window } = await mountFiles()

    expect(window.find('.caution').exists()).toBe(false)
  })
})

describe('an address dropped on the files tab', () => {
  it('goes to the plugin that takes it, into the folder of the selection', async () => {
    const { recorded, window } = await mountFiles()
    const dataTransfer = {
      types: ['text/uri-list'],
      getData: (type: string) => (type === 'text/uri-list' ? 'https://video.test/clip' : ''),
    }
    await window.find('.files').trigger('drop', { dataTransfer })
    expect(recorded).toContain('drops into / https://video.test/clip')
  })

  it('is left to the window where it carries files alone', async () => {
    const { recorded, window } = await mountFiles()
    const dataTransfer = { types: ['Files'], getData: () => '' }
    await window.find('.files').trigger('drop', { dataTransfer })
    expect(recorded.some((one) => one.startsWith('drops into'))).toBe(false)
  })
})
