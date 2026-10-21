/**
 * A files tab drawn, in a document.
 */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { Menu, Tree } from '@numen/ui'
import type { Entry } from '../types'
import FilesTab from './FilesTab.vue'
import { useFilesTab, type FilesTabState } from '../model/useFilesTab'
import { useFileTree, ROOT } from '../model/useFileTree'
import { drawing } from '../fixtures/host'

const file = (path: string, over: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  isFolder: false,
  kind: 'note',
  type: 'note',
  ...over,
})

const folder = (path: string): Entry => file(path, { isFolder: true, kind: 'other' })

const held: Record<string, readonly Entry[]> = {
  [ROOT]: [folder('physics'), file('Entropy.md'), file('Cover.png', { kind: 'other' })],
  physics: [file('physics/Kelvin.md')],
}

/** A moment for whatever the tab asked the vault for to come back. */
const settle = () => new Promise((done) => setTimeout(done, 0))

/** A tab of that vault, drawn, and what it asked of the window written down. */
const mountFiles = async (open: readonly string[] = []) => {
  const done: string[] = []
  const list = useFileTree({ list: async (at: string) => held[at] ?? [] })
  const tab: FilesTabState = useFilesTab(list, {
    ...drawing,
    openEntry: (entry) => void done.push(`lands file ${entry.path}`),
    runCommand: (id, paths, details) =>
      void done.push(`runs ${id} ${paths.join(' ')} ${details.name}`),
    movePath: async (from, to) => void done.push(`moves ${from} ${to}`),
    setDraggedPaths: (paths) => void done.push(`drags ${paths.join(' ') || '—'}`),
    createFolder: async (path) => void done.push(`makes ${path}`),
    createNote: async (folderPath) => `${folderPath}Untitled note.md`,
    createDeck: async (folderPath, name) => `${folderPath}${name}`,
    createStencil: async (folderPath, name) => `${folderPath}${name}`,
    createPreset: async (folderPath, name) => `${folderPath}${name}`,
    importUrl: async (folderPath, url) => void done.push(`imports ${folderPath} ${url}`) as never,
    importUrls: async (folderPath, urls) => {
      done.push(`imports-multi ${folderPath || '/'} ${urls.join(' ')}`)
      return []
    },
    showError: (text) => void done.push(`says ${text}`),
  })
  await list.openFolder(ROOT)
  for (const at of open) await list.openFolder(at)
  const window = mount(FilesTab, { props: { state: tab } })
  await settle()
  return { done, list, tab, window }
}

/** What the tree was handed, as a path and the paths under it. */
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
    const { done, window } = await mountFiles()

    window.findComponent(Tree).vm.$emit('activate', 'Entropy.md')
    await settle()

    expect(done).toStrictEqual(['lands file Entropy.md'])
  })

  it('takes the person to a file the vault holds no source for', async () => {
    const { done, window } = await mountFiles()

    window.findComponent(Tree).vm.$emit('activate', 'Cover.png')
    await settle()

    expect(done).toStrictEqual(['lands file Cover.png'])
  })

  it('is filed where it was release', async () => {
    const { done, window } = await mountFiles()

    window.findComponent(Tree).vm.$emit('move', ['Entropy.md'], { into: 'physics' })
    await settle()

    expect(done).toStrictEqual(['moves Entropy.md physics/Entropy.md'])
  })

  it('is one of several filed where they were all release', async () => {
    const { done, window } = await mountFiles()

    window.findComponent(Tree).vm.$emit('move', ['Entropy.md', 'Cover.png'], { into: 'physics' })
    await settle()

    expect(done).toStrictEqual([
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
    const { done, window } = await mountFiles()

    window.findComponent(Tree).vm.$emit('remove', ['Entropy.md', 'Cover.png'])
    await settle()

    expect(done).toStrictEqual(['runs remove Entropy.md Cover.png Entropy.md'])
  })

  it('is filed under the name that was typed over it', async () => {
    const { done, window } = await mountFiles()

    window.findComponent(Tree).vm.$emit('rename', 'Entropy.md', 'Order.md')
    await settle()

    expect(done).toStrictEqual(['moves Entropy.md Order.md'])
  })

  it('opens the folder it remains for, and what it holds is drawn', async () => {
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
    const { list, window } = await mountFiles()
    if (selection.length) list.selectPaths(selection)
    window.findComponent(Tree).vm.$emit('menu', path, { x: 4, y: 8 })
    await settle()
    return window.findComponent(Menu).props('items') as readonly { id: string; group?: string }[]
  }

  const getItemIds = async (path: string | null, selection: readonly string[] = []) =>
    (await openMenu(path, selection)).map((one) => one.id)

  it('offers everything that can be done to a note it was requested for on', async () => {
    expect(await getItemIds('Entropy.md')).toStrictEqual([
      'read',
      'travel',
      'newNote',
      'newDeck',
      'newStencil',
      'newPreset',
      'newFolder',
      'rename',
      'copy',
      'child',
      'parent',
      'jump',
      'title',
      'ask',
      'remove',
    ])
  })

  it('remains the items of a note in the groups they belong to', async () => {
    expect((await openMenu('Entropy.md')).map((one) => one.group)).toStrictEqual([
      'open',
      'open',
      'file',
      'file',
      'file',
      'file',
      'file',
      'file',
      'file',
      'plex',
      'plex',
      'plex',
      'plex',
      'agent',
      'remove',
    ])
  })

  it('offers no command over a note on a file the vault holds no source for', async () => {
    expect(await getItemIds('Cover.png')).toStrictEqual([
      'newNote',
      'newDeck',
      'newStencil',
      'newPreset',
      'newFolder',
      'rename',
      'copy',
      'remove',
    ])
  })

  it('offers no command over a note on a folder', async () => {
    expect(await getItemIds('physics')).not.toContain('title')
  })

  it('offers what can be made at the root, requested off every row', async () => {
    expect(await getItemIds(null)).toStrictEqual([
      'newNote',
      'newDeck',
      'newStencil',
      'newPreset',
      'newFolder',
    ])
  })

  it('offers removal alone over a selection of several', async () => {
    expect(await getItemIds('Entropy.md', ['Entropy.md', 'Cover.png'])).toStrictEqual(['remove'])
  })
})

describe('a folder that could not be read', () => {
  it('is returned in the tab', async () => {
    const list = useFileTree({
      list: async () => {
        throw new Error('the vault is not there')
      },
    })
    const tab: FilesTabState = useFilesTab(list, {
      ...drawing,
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

    const window = mount(FilesTab, { props: { state: tab } })
    await settle()

    expect(window.find('.caution').text()).toContain('numen did not answer')
  })

  it('is returned nowhere while the vault answers', async () => {
    const { window } = await mountFiles()

    expect(window.find('.caution').exists()).toBe(false)
  })
})

describe('several addresses imported into the files tab', () => {
  it('imports all addresses into the folder', async () => {
    const { done, tab } = await mountFiles()
    await tab.importUrls(['https://first.org/', 'https://second.org/'])
    expect(done).toContain('imports-multi / https://first.org/ https://second.org/')
  })
})
