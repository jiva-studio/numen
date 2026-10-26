import { effectScope } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import type { WindowTab } from './host'
import { createWindowKeys } from './windowKeys'

const VAULTS = [
  { id: 'physics', name: 'Physics', path: '/vaults/Physics' },
  { id: 'words', name: 'Words', path: '/vaults/Words' },
]

/** A tab whose kind keeps escape from the page and leaves every other key to it. */
const createTab = (): WindowTab & { pressed: string[] } => {
  const pressed: string[] = []
  return {
    id: 'review#1',
    vault: 'words',
    state: { pressed },
    scope: effectScope(),
    pressed,
    view: {
      pluginId: 'md.numen.cards',
      registration: {
        kind: 'review',
        component: {},
        onKeyPress: (_, event) => {
          pressed.push(event.key)
          return event.key === 'Escape'
        },
      },
    },
  }
}

const press = (key: string, more: KeyboardEventInit = {}) =>
  new KeyboardEvent('keydown', { key, cancelable: true, ...more })

describe('a key pressed on the list of vaults', () => {
  const createKeys = () => {
    const openVault = vi.fn()
    const onKeyDown = createWindowKeys({ getTab: () => null, getVaults: () => VAULTS, openVault })
    return { onKeyDown, openVault }
  }

  it('opens the vault standing at its letter, and keeps the letter from the page', () => {
    const { onKeyDown, openVault } = createKeys()
    const event = press('b')

    onKeyDown(event)

    expect(openVault).toHaveBeenCalledExactlyOnceWith('words')
    expect(event.defaultPrevented).toBe(true)
  })

  it('opens nothing for a letter past the list or a letter with a modifier', () => {
    const { onKeyDown, openVault } = createKeys()
    const events = [press('c'), press('a', { ctrlKey: true })]

    for (const event of events) onKeyDown(event)

    expect(openVault).not.toHaveBeenCalled()
    expect(events.some((one) => one.defaultPrevented)).toBe(false)
  })
})

describe('a key pressed while a tab is open', () => {
  it('goes to the tab, which says whether the page is kept from it', () => {
    const tab = createTab()
    const openVault = vi.fn()
    const onKeyDown = createWindowKeys({ getTab: () => tab, getVaults: () => VAULTS, openVault })
    const letter = press('a')
    const escape = press('Escape')

    onKeyDown(letter)
    onKeyDown(escape)

    expect(tab.pressed).toStrictEqual(['a', 'Escape'])
    expect(letter.defaultPrevented).toBe(false)
    expect(escape.defaultPrevented).toBe(true)
    expect(openVault).not.toHaveBeenCalled()
  })

  it('is left to the page by a kind that reads no keys', () => {
    const tab = createTab()
    const { onKeyPress: _, ...registration } = tab.view.registration
    const onKeyDown = createWindowKeys({
      getTab: () => ({ ...tab, view: { ...tab.view, registration } }),
      getVaults: () => VAULTS,
      openVault: vi.fn(),
    })
    const event = press('Escape')

    onKeyDown(event)

    expect(event.defaultPrevented).toBe(false)
  })
})
