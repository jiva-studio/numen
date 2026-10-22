/**
 * The keyboard in an open menu, without the menu drawn around it.
 */
import { describe, expect, it, vi } from 'vitest'
import { shallowRef } from 'vue'
import type { MenuItem } from './item'
import { useMenuKeys } from './keys'

const ITEMS: MenuItem[] = [
  { id: 'open', text: 'Open' },
  { id: 'copy', text: 'Copy' },
  { id: 'cut', text: 'Cut' },
]

const createMenuElement = (): HTMLElement => {
  const element = document.createElement('div')
  element.tabIndex = -1
  return element
}

const createKey = (key: string, init: KeyboardEventInit = {}): KeyboardEvent =>
  new KeyboardEvent('keydown', { key, cancelable: true, ...init })

const setup = () => {
  const element = createMenuElement()
  const menu = shallowRef<HTMLElement | null>(element)
  const focusMenu = vi.spyOn(element, 'focus')
  const keys = useMenuKeys(
    () => ITEMS,
    menu,
    () => 0,
  )
  return { keys, menu, focusMenu }
}

describe('where the keyboard starts', () => {
  it('is on no item, and the first step down lands on the first', () => {
    const { keys } = setup()
    expect(keys.here.value).toBe(-1)

    keys.onKey(createKey('ArrowDown'))
    expect(keys.here.value).toBe(0)
  })

  it('lands on the last item for the first step up', () => {
    const { keys } = setup()
    keys.onKey(createKey('ArrowUp'))
    expect(keys.here.value).toBe(2)
  })
})

describe('taking the keyboard onto a row', () => {
  it('focuses the row retained for the item', () => {
    const { keys, focusMenu } = setup()
    const focus = vi.fn<() => void>()
    keys.holdRow('copy', { focus })

    keys.goTo(1)
    expect(focus).toHaveBeenCalledTimes(1)
    expect(focusMenu).not.toHaveBeenCalled()
  })

  it('focuses the menu once the row is let go', () => {
    const { keys, focusMenu } = setup()
    const focus = vi.fn<() => void>()
    keys.holdRow('copy', { focus })
    keys.holdRow('copy', null)

    keys.goTo(1)
    expect(focus).not.toHaveBeenCalled()
    expect(focusMenu).toHaveBeenCalledTimes(1)
  })

  it('goes on without a menu element to focus', () => {
    const { keys, menu } = setup()
    menu.value = null
    expect(() => keys.goTo(-1)).not.toThrow()
    expect(keys.here.value).toBe(-1)
  })
})

describe('keys pressed inside the menu', () => {
  it('stays where it is for a letter no item begins with', () => {
    const { keys } = setup()
    keys.goTo(1)

    keys.onKey(createKey('z'))
    expect(keys.here.value).toBe(1)
  })

  it('jumps to the item a letter begins and holds the key back from the page', () => {
    const { keys } = setup()
    const event = createKey('o')
    keys.onKey(event)
    expect(keys.here.value).toBe(0)
    expect(event.defaultPrevented).toBe(true)
  })

  it.each([
    ['Enter', {}],
    ['F5', {}],
    ['a', { ctrlKey: true }],
    [' ', {}],
  ])('leaves %s to the page', (key, init) => {
    const { keys } = setup()
    keys.goTo(1)
    const event = createKey(key, init)

    keys.onKey(event)
    expect(event.defaultPrevented).toBe(false)
    expect(keys.here.value).toBe(1)
  })
})

describe('Tab and the ends', () => {
  it('walks down on Tab and up on Shift+Tab', () => {
    const { keys } = setup()
    keys.goTo(1)
    keys.onKey(createKey('Tab'))
    expect(keys.here.value).toBe(2)

    keys.onKey(createKey('Tab', { shiftKey: true }))
    expect(keys.here.value).toBe(1)
  })

  it('lands on the first item for Home and the last for End', () => {
    const { keys } = setup()
    keys.goTo(1)
    keys.onKey(createKey('End'))
    expect(keys.here.value).toBe(2)

    keys.onKey(createKey('Home'))
    expect(keys.here.value).toBe(0)
  })
})
