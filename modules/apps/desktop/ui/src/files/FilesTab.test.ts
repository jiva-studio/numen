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
import { filing, type Held } from './kind'
import { listing, ROOT } from './listing'

const file = (path: string, over: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  folder: false,
  kind: 'note',
  size: 1,
  ...over,
})

const folder = (path: string): Entry => file(path, { folder: true, kind: 'other', size: 0 })

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
  const tab: Held = filing(list, {
    lands: (landing) => void done.push(`lands ${landing ? `${landing.at} ${landing.path}` : '—'}`),
    runs: (id, path, name) => void done.push(`runs ${id} ${path} ${name}`),
    moves: async (from, to) => void done.push(`moves ${from} ${to}`),
    makes: async (path) => void done.push(`makes ${path}`),
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

  it('draws a mark of its own beside every row', async () => {
    const { window } = await drawn()

    expect(window.findAll('.files__mark')).toHaveLength(3)
  })
})

describe('a row the tree reports', () => {
  it('takes the person to the note it stands for', async () => {
    const { done, window } = await drawn()

    window.findComponent(Tree).vm.$emit('activate', 'Entropy.md')
    await settles()

    expect(done).toStrictEqual(['lands note Entropy.md'])
  })

  it('takes the person nowhere for a file the vault holds no source for', async () => {
    const { done, window } = await drawn()

    window.findComponent(Tree).vm.$emit('activate', 'Cover.png')
    await settles()

    expect(done).toStrictEqual(['lands —'])
  })

  it('is filed where it was let go of', async () => {
    const { done, window } = await drawn()

    window.findComponent(Tree).vm.$emit('move', 'Entropy.md', { into: 'physics' })
    await settles()

    expect(done).toStrictEqual(['moves Entropy.md physics/Entropy.md'])
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

  const itemsOn = async (path: string) => {
    const { window } = await drawn()
    window.findComponent(Tree).vm.$emit('menu', path, { x: 4, y: 8 })
    await settles()
    return (window.findComponent(Menu).props('items') as readonly { id: string }[]).map(
      (one) => one.id,
    )
  }

  it('offers everything that can be done to a note it was asked for on', async () => {
    expect(await itemsOn('Entropy.md')).toContain('travel')
  })

  it('offers no command over a note on a file the vault holds no source for', async () => {
    expect(await itemsOn('Cover.png')).not.toContain('travel')
  })

  it('offers no command over a note on a folder', async () => {
    expect(await itemsOn('physics')).not.toContain('title')
  })

  it('offers a folder and a name on every row', async () => {
    expect(await itemsOn('Cover.png')).toStrictEqual(['rename', 'newFolder', 'remove'])
  })
})

describe('a folder that could not be read', () => {
  it('is said in the tab', async () => {
    const list = listing({
      list: async () => {
        throw new Error('the vault is not there')
      },
    })
    const tab: Held = filing(list, {
      lands: () => {},
      runs: () => {},
      moves: async () => {},
      makes: async () => {},
    })
    await list.opens(ROOT)

    const window = mount(FilesTab, { props: { held: tab } })
    await settles()

    expect(window.find('.warning').text()).toContain('the vault is not there')
  })

  it('is said nowhere while the vault answers', async () => {
    const { window } = await drawn()

    expect(window.find('.warning').exists()).toBe(false)
  })
})
