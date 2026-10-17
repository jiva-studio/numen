/**
 * The window as it opens, and the screen it shows holding nothing.
 *
 * What is asked here is the layout: which kinds a vault showing opens with,
 * what a window holding no tab draws instead, and what a gesture in the tree
 * puts in front of the person.
 */
import { describe, expect, it } from 'vitest'
import { type VueWrapper } from '@vue/test-utils'
import { Plex, Tree, WelcomePage, WorkspaceLayout } from '@numen/ui'
import { AgentTab } from '@/pages/agent-chat'
import { DocumentTab } from '@plugins/reader-pdf/frontend'
import { FilesTab } from '@/pages/file-manager'
import { PlexTab } from '@plugins/view-plex-graph/frontend'
import { getPanes } from '@numen/ui'
import {
  requests,
  cards,
  getLayout,
  listed,
  mountWindow,
  paneKinds,
  said,
  settle,
  getTabs,
} from '@/testing/window'

describe('the window as it opens', () => {
  it('draws a plex in the room, and an agent in front of the files beside it', async () => {
    const window = await mountWindow()

    expect(paneKinds(window)).toStrictEqual([['plex'], ['agent', 'files']])
    expect(getPanes(getLayout(window).root)[1]?.active).toMatch(/^agent:/)
    expect(window.findComponent(PlexTab).exists()).toBe(true)
    expect(window.findComponent(AgentTab).exists()).toBe(true)
    expect(window.findComponent(FilesTab).exists()).toBe(true)
  })

  it('leaves the person in the plex, which holds the greater share', async () => {
    const window = await mountWindow()
    const root = getLayout(window).root

    expect(getLayout(window).focus).toBe('main')
    expect(root.kind === 'branch' ? root.sizes : []).toStrictEqual([0.72, 0.28])
  })

  it('hands the tree what the root of the vault holds', async () => {
    const window = await mountWindow()

    expect(
      (window.findComponent(Tree).props('rows') as readonly { id: string }[]).map((one) => one.id),
    ).toStrictEqual(['physics', 'Root.md', 'Cover.png'])
  })
})

describe('the window holding no tab', () => {
  /** Every tab closed, by the keystroke that closes the one in front. */
  const closeEveryTab = async (window: VueWrapper) => {
    for (let each = getTabs(window).length; each > 0; each -= 1) {
      globalThis.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'W', ctrlKey: true, shiftKey: true, cancelable: true }),
      )
      await settle()
    }
  }

  it('opens on the welcome screen, holding nothing, while the list shows no vault', async () => {
    listed.showing = ''

    const window = await mountWindow()

    expect(getTabs(window)).toStrictEqual([])
    expect(window.findComponent(WelcomePage).exists()).toBe(true)
  })

  it('comes to the same screen once every tab it opened with is closed', async () => {
    const window = await mountWindow()

    await closeEveryTab(window)

    expect(getTabs(window)).toStrictEqual([])
    expect(window.findComponent(WelcomePage).exists()).toBe(true)
  })

  it('draws the vault the list is showing on it, said to be the one in front', async () => {
    const window = await mountWindow()

    await closeEveryTab(window)

    expect(window.findComponent(WelcomePage).props('vaults')).toStrictEqual([
      { id: 'physics', name: 'Physics', path: '/vaults/Physics', detail: 'Current' },
    ])
  })

  it('says in the corner that the vaults could not be listed, and draws none', async () => {
    said.listable = false

    const window = await mountWindow()

    expect(cards(window).join(' ')).toContain('The vaults could not be listed')
    expect(window.findComponent(WelcomePage).props('vaults')).toStrictEqual([])
  })
})

describe('a letter pressed on the welcome screen', () => {
  /** The screen with two vaults on it, and no tab over them. */
  const two = async () => {
    listed.vaults = [
      { id: 'physics', name: 'Physics', path: '/vaults/Physics', isMissing: false },
      { id: 'heat', name: 'Heat', path: '/vaults/Heat', isMissing: false },
    ]
    listed.showing = ''
    return mountWindow()
  }

  const press = async (key: string, more: KeyboardEventInit = {}) => {
    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key, cancelable: true, ...more }))
    await settle()
    await settle()
  }

  it('shows the vault standing at it', async () => {
    await two()

    await press('b')

    expect(requests.opened).toStrictEqual(['heat'])
  })

  it('shows the first of them for the first letter of the alphabet', async () => {
    await two()

    await press('a')

    expect(requests.opened).toStrictEqual(['physics'])
  })

  it('shows nothing where no vault stands at the letter', async () => {
    await two()

    await press('c')

    expect(requests.opened).toStrictEqual([])
  })

  it('shows nothing while the palette is up, where the letter is being typed', async () => {
    await two()

    await press('k', { ctrlKey: true })
    await press('a')

    expect(requests.opened).toStrictEqual([])
  })

  it('shows nothing while the window holds a tab, where the screen is not up', async () => {
    await mountWindow()

    await press('a')

    expect(requests.opened).toStrictEqual([])
  })
})

describe('the vault offered below the list', () => {
  it('carries the keystroke that reaches it, drawn on its own row', async () => {
    listed.showing = ''

    const window = await mountWindow()

    expect(window.findComponent(WelcomePage).props('offer')).toMatchObject({
      keys: { icons: ['control', 'shift'], letter: 'N' },
    })
  })

  it('is asked for by that keystroke, which is a folder chosen on this machine', async () => {
    listed.showing = ''
    await mountWindow()

    globalThis.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'N', ctrlKey: true, shiftKey: true, cancelable: true }),
    )
    await settle()

    expect(requests.chose).toBe(1)
  })
})

describe('a row activated in the files', () => {
  const activateRow = async (path: string) => {
    const window = await mountWindow()
    window.findComponent(Tree).vm.$emit('activate', path)
    await settle()
    await settle()
    return window
  }

  it('opens a note in a tab of its own', async () => {
    const window = await activateRow('Root.md')

    expect(paneKinds(window).flat()).toContain('note')
  })

  it('opens nothing at all for a file the vault holds no source for', async () => {
    const window = await activateRow('Cover.png')

    expect(paneKinds(window).flat()).not.toContain('note')
    expect(window.findComponent(DocumentTab).exists()).toBe(false)
  })

  it('opens nothing for a folder, which turns where it stands', async () => {
    const window = await activateRow('physics')

    expect(paneKinds(window).flat()).not.toContain('note')
  })
})

describe('a tab asked to close', () => {
  it('is told to hold while a kind with something to finish is still closing it', async () => {
    said.types['Root.md'] = 'deck'
    const window = await mountWindow()
    window.findComponent(Tree).vm.$emit('activate', 'Root.md')
    await settle()
    await settle()

    const deckId = getTabs(window).find((tab) => tab.id.startsWith('deck:'))?.id
    expect(deckId).toBeTruthy()

    let held = false
    window.findComponent(WorkspaceLayout).vm.$emit('close', deckId, () => {
      held = true
    })

    // A deck tab finishes its own close asynchronously, so the layout is
    // told to hold it open rather than drop it from the tree on the spot.
    expect(held).toBe(true)
  })
})

describe('a file dragged out of the tree', () => {
  /** The window with the physics folder open, and a row dragged out of it. */
  const dragRows = async (rows: readonly string[]) => {
    const window = await mountWindow()
    const tree = window.findComponent(Tree)
    tree.vm.$emit('open', 'physics')
    await settle()
    tree.vm.$emit('drag', rows)
    await settle()
    return window
  }

  const getDragged = (window: VueWrapper) => window.findComponent(Plex).props('dragged')

  it('is what the plex draws a line to, though neither knows the other is there', async () => {
    expect(getDragged(await dragRows(['physics/Entropy.md']))).toStrictEqual(['physics/Entropy.md'])
  })

  it('is every note of the selection, all of them at once', async () => {
    const window = await dragRows(['physics/Entropy.md', 'physics/Kelvin.md'])

    expect(getDragged(window)).toStrictEqual(['physics/Entropy.md', 'physics/Kelvin.md'])
  })

  it('leaves out the note the plex is standing on, and keeps the rest', async () => {
    const window = await dragRows(['Root.md', 'physics/Entropy.md'])

    expect(getDragged(window)).toStrictEqual(['physics/Entropy.md'])
  })

  it('is nothing once it has been let go of, wherever that was', async () => {
    const window = await dragRows(['physics/Entropy.md'])

    window.findComponent(Tree).vm.$emit('drop')
    await settle()

    expect(getDragged(window)).toStrictEqual([])
  })

  it('leaves out a file the vault holds no note for, and a folder', async () => {
    const window = await dragRows(['physics', 'physics/Entropy.md', 'Cover.png'])

    expect(getDragged(window)).toStrictEqual(['physics/Entropy.md'])
  })

  it('is nothing where the rows hold no note at all', async () => {
    expect(getDragged(await dragRows(['Cover.png', 'physics']))).toStrictEqual([])
  })
})

describe('tab linking context menu', () => {
  it('opens link menu on tab contextmenu and links tabs', async () => {
    const window = await mountWindow()
    const layout = window.findComponent(WorkspaceLayout)
    expect(layout.exists()).toBe(true)

    const plexTabId = getTabs(window).find((t) => t.id.startsWith('plex:'))?.id ?? 'plex:main'
    const filesTabId = getTabs(window).find((t) => t.id.startsWith('files:'))?.id ?? 'files:side'

    layout.vm.$emit('tab-context-menu', plexTabId, { x: 50, y: 50 })
    await settle()

    const linkMenu = window.findComponent({ name: 'TabLinkMenu' })
    expect(linkMenu.exists()).toBe(true)

    linkMenu.vm.$emit('link', filesTabId)
    await settle()

    expect(window.findComponent({ name: 'TabLinkMenu' }).exists()).toBe(false)

    // Reopen menu to verify unlink
    layout.vm.$emit('tab-context-menu', plexTabId, { x: 50, y: 50 })
    await settle()

    const openedMenu = window.findComponent({ name: 'TabLinkMenu' })
    expect(openedMenu.exists()).toBe(true)
    expect(openedMenu.props('isLinked')).toBe(true)

    openedMenu.vm.$emit('unlink')
    await settle()

    expect(window.findComponent({ name: 'TabLinkMenu' }).exists()).toBe(false)
  })

  it('closes link menu when close event is emitted', async () => {
    const window = await mountWindow()
    const layout = window.findComponent(WorkspaceLayout)

    const plexTabId = getTabs(window).find((t) => t.id.startsWith('plex:'))?.id ?? 'plex:main'
    layout.vm.$emit('tab-context-menu', plexTabId, { x: 50, y: 50 })
    await settle()

    const linkMenu = window.findComponent({ name: 'TabLinkMenu' })
    linkMenu.vm.$emit('close')
    await settle()

    expect(window.findComponent({ name: 'TabLinkMenu' }).exists()).toBe(false)
  })

  it('detaches and releases tab when tear-off is emitted', async () => {
    const window = await mountWindow()
    const layout = window.findComponent(WorkspaceLayout)

    const plexTabId = getTabs(window).find((t) => t.id.startsWith('plex:'))?.id ?? 'plex:main'
    layout.vm.$emit('tear-off', plexTabId, { x: 300, y: 400 })
    await settle()

    const remaining = getTabs(window).map((t) => t.id)
    expect(remaining).not.toContain(plexTabId)
  })

  it('keeps tab open when tear-off fails', async () => {
    said.detachError = 'detach rejected'
    const window = await mountWindow()
    const layout = window.findComponent(WorkspaceLayout)

    const plexTabId = getTabs(window).find((t) => t.id.startsWith('plex:'))?.id ?? 'plex:main'
    layout.vm.$emit('tear-off', plexTabId, { x: 300, y: 400 })
    await settle()

    const remaining = getTabs(window).map((t) => t.id)
    expect(remaining).toContain(plexTabId)
  })
})
