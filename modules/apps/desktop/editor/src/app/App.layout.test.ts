/**
 * The window as it opens, and the screen it shows holding nothing.
 *
 * What is asked here is the layout: which kinds a vault showing opens with,
 * what a window holding no tab draws instead, and what a gesture in the tree
 * puts in front of the person.
 */
import { describe, expect, it } from 'vitest'
import { type VueWrapper } from '@vue/test-utils'
import { Plex, Tree, WelcomePage } from '@numen/ui'
import { AgentTab } from '@/pages/agent-chat'
import { DocumentTab } from '@/pages/document-viewer'
import { FilesTab } from '@/pages/file-manager'
import { NoteTab } from '@/pages/note-editor'
import { PlexTab } from '@/pages/plex-graph'
import { panesOf } from '@numen/ui'
import {
  asked,
  cards,
  drawn,
  layoutOf,
  listed,
  paneKinds,
  said,
  settles,
  tabsOf,
} from '@/testing/window'

describe('the window as it opens', () => {
  it('draws a plex in the room, and an agent in front of the files beside it', async () => {
    const window = await drawn()

    expect(paneKinds(window)).toStrictEqual([['plex'], ['agent', 'files']])
    expect(panesOf(layoutOf(window).root)[1]?.active).toMatch(/^agent:/)
    expect(window.findComponent(PlexTab).exists()).toBe(true)
    expect(window.findComponent(AgentTab).exists()).toBe(true)
    expect(window.findComponent(FilesTab).exists()).toBe(true)
  })

  it('leaves the person in the plex, which holds the greater share', async () => {
    const window = await drawn()
    const root = layoutOf(window).root

    expect(layoutOf(window).focus).toBe('main')
    expect(root.kind === 'branch' ? root.sizes : []).toStrictEqual([0.72, 0.28])
  })

  it('hands the tree what the root of the vault holds', async () => {
    const window = await drawn()

    expect(
      (window.findComponent(Tree).props('rows') as readonly { id: string }[]).map((one) => one.id),
    ).toStrictEqual(['physics', 'Root.md', 'Cover.png'])
  })
})

describe('the window holding no tab', () => {
  /** Every tab closed, by the keystroke that closes the one in front. */
  const closesEvery = async (window: VueWrapper) => {
    for (let each = tabsOf(window).length; each > 0; each -= 1) {
      globalThis.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'W', ctrlKey: true, shiftKey: true, cancelable: true }),
      )
      await settles()
    }
  }

  it('opens on the welcome screen, holding nothing, while the list shows no vault', async () => {
    listed.showing = ''

    const window = await drawn()

    expect(tabsOf(window)).toStrictEqual([])
    expect(window.findComponent(WelcomePage).exists()).toBe(true)
  })

  it('comes to the same screen once every tab it opened with is closed', async () => {
    const window = await drawn()

    await closesEvery(window)

    expect(tabsOf(window)).toStrictEqual([])
    expect(window.findComponent(WelcomePage).exists()).toBe(true)
  })

  it('draws the vault the list is showing on it, said to be the one in front', async () => {
    const window = await drawn()

    await closesEvery(window)

    expect(window.findComponent(WelcomePage).props('vaults')).toStrictEqual([
      { id: 'physics', name: 'Physics', path: '/vaults/Physics', detail: 'Current' },
    ])
  })

  it('says in the corner that the vaults could not be listed, and draws none', async () => {
    said.listable = false

    const window = await drawn()

    expect(cards(window).join(' ')).toContain('The vaults could not be listed')
    expect(window.findComponent(WelcomePage).props('vaults')).toStrictEqual([])
  })
})

describe('a letter pressed on the welcome screen', () => {
  /** The screen with two vaults on it, and no tab over them. */
  const two = async () => {
    listed.vaults = [
      { id: 'physics', name: 'Physics', path: '/vaults/Physics', missing: false },
      { id: 'heat', name: 'Heat', path: '/vaults/Heat', missing: false },
    ]
    listed.showing = ''
    return drawn()
  }

  const press = async (key: string, more: KeyboardEventInit = {}) => {
    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key, cancelable: true, ...more }))
    await settles()
    await settles()
  }

  it('shows the vault standing at it', async () => {
    await two()

    await press('b')

    expect(asked.opened).toStrictEqual(['heat'])
  })

  it('shows the first of them for the first letter of the alphabet', async () => {
    await two()

    await press('a')

    expect(asked.opened).toStrictEqual(['physics'])
  })

  it('shows nothing where no vault stands at the letter', async () => {
    await two()

    await press('c')

    expect(asked.opened).toStrictEqual([])
  })

  it('shows nothing while the palette is up, where the letter is being typed', async () => {
    await two()

    await press('k', { ctrlKey: true })
    await press('a')

    expect(asked.opened).toStrictEqual([])
  })

  it('shows nothing while the window holds a tab, where the screen is not up', async () => {
    await drawn()

    await press('a')

    expect(asked.opened).toStrictEqual([])
  })
})

describe('the vault offered below the list', () => {
  it('carries the keystroke that reaches it, drawn on its own row', async () => {
    listed.showing = ''

    const window = await drawn()

    expect(window.findComponent(WelcomePage).props('offer')).toMatchObject({
      keys: { icons: ['control', 'shift'], letter: 'N' },
    })
  })

  it('is asked for by that keystroke, which is a folder chosen on this machine', async () => {
    listed.showing = ''
    await drawn()

    globalThis.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'N', ctrlKey: true, shiftKey: true, cancelable: true }),
    )
    await settles()

    expect(asked.chose).toBe(1)
  })
})

describe('a row activated in the files', () => {
  const activated = async (path: string) => {
    const window = await drawn()
    window.findComponent(Tree).vm.$emit('activate', path)
    await settles()
    await settles()
    return window
  }

  it('opens a note in a tab of its own', async () => {
    const window = await activated('Root.md')

    expect(window.findComponent(NoteTab).exists()).toBe(true)
  })

  it('opens nothing at all for a file the vault holds no source for', async () => {
    const window = await activated('Cover.png')

    expect(window.findComponent(NoteTab).exists()).toBe(false)
    expect(window.findComponent(DocumentTab).exists()).toBe(false)
  })

  it('opens nothing for a folder, which turns where it stands', async () => {
    const window = await activated('physics')

    expect(window.findComponent(NoteTab).exists()).toBe(false)
  })
})

describe('a file dragged out of the tree', () => {
  /** The window with the physics folder open, and a row dragged out of it. */
  const dragging = async (rows: readonly string[]) => {
    const window = await drawn()
    const tree = window.findComponent(Tree)
    tree.vm.$emit('open', 'physics')
    await settles()
    tree.vm.$emit('drag', rows)
    await settles()
    return window
  }

  const dragged = (window: VueWrapper) => window.findComponent(Plex).props('dragged')

  it('is what the plex draws a line to, though neither knows the other is there', async () => {
    expect(dragged(await dragging(['physics/Entropy.md']))).toStrictEqual([
      'physics/Entropy.md',
    ])
  })

  it('is every note of the selection, all of them at once', async () => {
    const window = await dragging(['physics/Entropy.md', 'physics/Kelvin.md'])

    expect(dragged(window)).toStrictEqual(['physics/Entropy.md', 'physics/Kelvin.md'])
  })

  it('leaves out the note the plex is standing on, and keeps the rest', async () => {
    const window = await dragging(['Root.md', 'physics/Entropy.md'])

    expect(dragged(window)).toStrictEqual(['physics/Entropy.md'])
  })

  it('is nothing once it has been let go of, wherever that was', async () => {
    const window = await dragging(['physics/Entropy.md'])

    window.findComponent(Tree).vm.$emit('drop')
    await settles()

    expect(dragged(window)).toStrictEqual([])
  })

  it('leaves out a file the vault holds no note for, and a folder', async () => {
    const window = await dragging(['physics', 'physics/Entropy.md', 'Cover.png'])

    expect(dragged(window)).toStrictEqual(['physics/Entropy.md'])
  })

  it('is nothing where the rows hold no note at all', async () => {
    expect(dragged(await dragging(['Cover.png', 'physics']))).toStrictEqual([])
  })
})

