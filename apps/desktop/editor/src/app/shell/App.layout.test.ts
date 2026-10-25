/**
 * The window as it opens, and the screen it shows holding nothing.
 *
 * What is asked here is the layout: which kinds a vault showing opens with,
 * what a window holding no tab draws instead, and what a gesture in the tree
 * puts in front of the person.
 */
import { describe, expect, it } from 'vitest'
import { type VueWrapper } from '@vue/test-utils'
import { WelcomePage, WorkspaceLayout } from '@numen/ui'
import { requests, cards, listed, mountWindow, said, settle, getTabs } from '@/testing/window'

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

  it('evaluates to the same screen once every tab it opened with is closed', async () => {
    const window = await mountWindow()

    await closeEveryTab(window)

    expect(getTabs(window)).toStrictEqual([])
    expect(window.findComponent(WelcomePage).exists()).toBe(true)
  })

  it('draws the vault the list is showing on it, returned to be the one in front', async () => {
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

  it('shows nothing where no vault remains at the letter', async () => {
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
})

describe('the vault offered below the list', () => {
  it('carries the keystroke that reaches it, drawn on its own row', async () => {
    listed.showing = ''

    const window = await mountWindow()

    expect(window.findComponent(WelcomePage).props('offer')).toMatchObject({
      keys: { icons: ['control', 'shift'], letter: 'N' },
    })
  })

  it('is requested for by that keystroke, which is a folder chosen on this machine', async () => {
    listed.showing = ''
    await mountWindow()

    globalThis.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'N', ctrlKey: true, shiftKey: true, cancelable: true }),
    )
    await settle()

    expect(requests.chose).toBe(1)
  })
})

describe('tab linking context menu', () => {
  it('detaches and releases tab when tear-off is emitted', async () => {
    const window = await mountWindow()
    const layout = window.findComponent(WorkspaceLayout)

    const plexTabId = getTabs(window).find((t) => t.id.startsWith('plex:'))?.id ?? 'plex:main'
    layout.vm.$emit('tear-off', plexTabId, { x: 300, y: 400 })
    await settle()

    const remaining = getTabs(window).map((t) => t.id)
    expect(remaining).not.toContain(plexTabId)
  })
})
