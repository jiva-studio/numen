/**
 * A files tab drawn, in a document.
 *
 * What is asked here is what the tree is handed and what it is not: a row's
 * identity is the path the vault files it under, a folder that is closed hands
 * over nothing it holds, and the menu offers what can be done to the row it was
 * asked for on and nothing else.
 */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { Menu, Tree } from '@numen/ui'
import type { Entry } from '../core'
import FilesTab from './FilesTab.vue'
import { filing, type FilesTabState } from './kind'
import { listing, ROOT } from './listing'

const file = (path: string, over: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  folder: false,
  kind: 'note',
  type: 'note',
  ...over,
})

const folder = (path: string): Entry => file(path, { folder: true, kind: 'other' })

const held: Record<string, readonly Entry[]> = {
  [ROOT]: [folder('physics'), file('Entropy.md'), file('Cover.png', { kind: 'other' })],
  physics: [file('physics/Kelvin.md')],
}

/** A moment for whatever the tab asked the vault for to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

/** A tab of that vault, drawn, and what it asked of the window written down. */
const drawn = async (open: readonly string[] = []) => {
  const done: string[] = []
  const list = listing({ list: async (at: string) => held[at] ?? [] })
  const tab: FilesTabState = filing(list, {
    lands: (landing) => void done.push(`lands ${landing ? `${landing.at} ${landing.path}` : '—'}`),
    runs: (id, paths, name) => void done.push(`runs ${id} ${paths.join(' ')} ${name}`),
    moves: async (from, to) => void done.push(`moves ${from} ${to}`),
    carries: (paths) => void done.push(`carries ${paths.join(' ') || '—'}`),
    makes: async (path) => void done.push(`makes ${path}`),
    writes: async (folder) => `${folder}Untitled note.md`,
    cuts: async (folder, name) => `${folder}${name}`,
    stencils: async (folder, name) => `${folder}${name}`,
    presets: async (folder, name) => `${folder}${name}`,
    says: (text) => void done.push(`says ${text}`),
  })
  await list.opens(ROOT)
  for (const at of open) await list.opens(at)
  const window = mount(FilesTab, { props: { held: tab } })
  await settles()
  return { done, list, tab, window }
}

/** What the tree was handed, as a path and the paths under it. */
const rowsOf = (rows: readonly { id: string; rows?: readonly unknown[] }[]): unknown =>
  rows.map((one) => [one.id, rowsOf((one.rows ?? []) as never)])

describe('the tree the tab draws', () => {
  it('is handed a row per file the vault holds, under the path it is filed at', async () => {
    const { window } = await drawn()

    expect(rowsOf(window.findComponent(Tree).props('rows'))).toStrictEqual([
      ['physics', []],
      ['Entropy.md', []],
      ['Cover.png', []],
    ])
  })

  it('is handed what an open folder holds, inside the row that stands for it', async () => {
    const { window } = await drawn(['physics'])

    expect(rowsOf(window.findComponent(Tree).props('rows'))).toStrictEqual([
      ['physics', [['physics/Kelvin.md', []]]],
      ['Entropy.md', []],
      ['Cover.png', []],
    ])
  })

  it('says which folders are open, and names no folder that is closed', async () => {
    const { window } = await drawn()

    expect(window.findComponent(Tree).props('open')).not.toContain('physics')
  })

  it('draws an icon of its own beside every row', async () => {
    const { window } = await drawn()

    expect(window.findAll('.files__icon')).toHaveLength(3)
  })

  it('names the folder a file carried in from outside is filed in', async () => {
    const { window } = await drawn(['physics'])
    const marking = window.findComponent(Tree).props('marking') as {
      attribute: string
      valueFor: (row: string | null) => string
    }

    // The attribute the window's own drag and drop looks a target up by.
    expect(marking.attribute).toBe('data-file-drop-target')
    expect(marking.valueFor('physics')).toBe('physics')
    expect(marking.valueFor('physics/Kelvin.md')).toBe('physics')
    expect(marking.valueFor('Entropy.md')).toBe(ROOT)
    expect(marking.valueFor(null)).toBe(ROOT)
  })
})

describe('a row the tree reports', () => {
  it('takes the person to the note it stands for', async () => {
    const { done, window } = await drawn()

    window.findComponent(Tree).vm.$emit('activate', 'Entropy.md')
    await settles()

    expect(done).toStrictEqual(['lands file Entropy.md'])
  })

  it('takes the person to a file the vault holds no source for', async () => {
    const { done, window } = await drawn()

    window.findComponent(Tree).vm.$emit('activate', 'Cover.png')
    await settles()

    expect(done).toStrictEqual(['lands file Cover.png'])
  })

  it('is filed where it was let go of', async () => {
    const { done, window } = await drawn()

    window.findComponent(Tree).vm.$emit('move', ['Entropy.md'], { into: 'physics' })
    await settles()

    expect(done).toStrictEqual(['moves Entropy.md physics/Entropy.md'])
  })

  it('is one of several filed where they were all let go of', async () => {
    const { done, window } = await drawn()

    window.findComponent(Tree).vm.$emit('move', ['Entropy.md', 'Cover.png'], { into: 'physics' })
    await settles()

    expect(done).toStrictEqual([
      'moves Entropy.md physics/Entropy.md',
      'moves Cover.png physics/Cover.png',
    ])
  })

  it('is one of the rows the tree hands back as chosen', async () => {
    const { list, window } = await drawn()

    window.findComponent(Tree).vm.$emit('select', ['Entropy.md', 'Cover.png'])
    await settles()

    expect(list.chosen.value).toStrictEqual(['Entropy.md', 'Cover.png'])
    expect(window.findComponent(Tree).props('selected')).toStrictEqual([
      'Entropy.md',
      'Cover.png',
    ])
  })

  it('is one of the rows asked to go, handed to the window as one command', async () => {
    const { done, window } = await drawn()

    window.findComponent(Tree).vm.$emit('remove', ['Entropy.md', 'Cover.png'])
    await settles()

    expect(done).toStrictEqual(['runs remove Entropy.md Cover.png Entropy.md'])
  })

  it('is filed under the name that was typed over it', async () => {
    const { done, window } = await drawn()

    window.findComponent(Tree).vm.$emit('rename', 'Entropy.md', 'Order.md')
    await settles()

    expect(done).toStrictEqual(['moves Entropy.md Order.md'])
  })

  it('opens the folder it stands for, and what it holds is drawn', async () => {
    const { window } = await drawn()

    window.findComponent(Tree).vm.$emit('open', 'physics')
    await settles()

    expect(window.findComponent(Tree).props('open')).toContain('physics')
  })
})

describe('the menu on a row', () => {
  it('is drawn nowhere until a row asks for one', async () => {
    const { window } = await drawn()

    expect(window.findComponent(Menu).exists()).toBe(false)
  })

  const menuOn = async (path: string | null, chosen: readonly string[] = []) => {
    const { list, window } = await drawn()
    if (chosen.length) list.chooses(chosen)
    window.findComponent(Tree).vm.$emit('menu', path, { x: 4, y: 8 })
    await settles()
    return window.findComponent(Menu).props('items') as readonly { id: string; band?: string }[]
  }

  const itemsOn = async (path: string | null, chosen: readonly string[] = []) =>
    (await menuOn(path, chosen)).map((one) => one.id)

  it('offers everything that can be done to a note it was asked for on', async () => {
    expect(await itemsOn('Entropy.md')).toStrictEqual([
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

  it('stands the items of a note in the bands they belong to', async () => {
    expect((await menuOn('Entropy.md')).map((one) => one.band)).toStrictEqual([
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
    expect(await itemsOn('Cover.png')).toStrictEqual([
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
    expect(await itemsOn('physics')).not.toContain('title')
  })

  it('offers what can be made at the root, asked off every row', async () => {
    expect(await itemsOn(null)).toStrictEqual([
      'newNote',
      'newDeck',
      'newStencil',
      'newPreset',
      'newFolder',
    ])
  })

  it('offers removal alone over a selection of several', async () => {
    expect(await itemsOn('Entropy.md', ['Entropy.md', 'Cover.png'])).toStrictEqual(['remove'])
  })
})

describe('a folder that could not be read', () => {
  it('is said in the tab', async () => {
    const list = listing({
      list: async () => {
        throw new Error('the vault is not there')
      },
    })
    const tab: FilesTabState = filing(list, {
      lands: () => {},
      runs: () => {},
      moves: async () => {},
      carries: () => {},
      makes: async () => {},
      writes: async () => '',
      cuts: async () => '',
      stencils: async () => '',
      presets: async () => '',
      says: () => {},
    })
    await list.opens(ROOT)

    const window = mount(FilesTab, { props: { held: tab } })
    await settles()

    expect(window.find('.caution').text()).toContain('numen did not answer')
  })

  it('is said nowhere while the vault answers', async () => {
    const { window } = await drawn()

    expect(window.find('.caution').exists()).toBe(false)
  })
})
