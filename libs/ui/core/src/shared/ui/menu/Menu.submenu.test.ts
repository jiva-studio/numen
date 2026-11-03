/**
 * The menu inside a menu: what opens it, what is chosen in it, and what puts
 * it away. The negatives matter most: a row with a submenu is not an item to
 * choose, and the key that closes a submenu must not close the menu under it.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMenuMount, settle } from './fixtures/mount'
import type { MenuItem } from './items/item'

const ITEMS: MenuItem[] = [
  { id: 'open', text: 'Open' },
  {
    id: 'move',
    text: 'Move to',
    children: [
      { id: 'inbox', text: 'Inbox' },
      { id: 'archive', text: 'Archive' },
    ],
  },
  { id: 'copy', text: 'Copy path' },
]

const { mountMenu } = createMenuMount(ITEMS)

const getMenus = () => Array.from(document.body.querySelectorAll<HTMLElement>('.menu'))
const getRows = (menu: HTMLElement | undefined) =>
  Array.from(menu?.querySelectorAll<HTMLElement>('.menu__item') ?? [])

afterEach(() => {
  vi.restoreAllMocks()
})

describe('opening a submenu', () => {
  it('opens on hover over a row with children', async () => {
    mountMenu()
    await settle()
    expect(getMenus()).toHaveLength(1)

    getRows(getMenus()[0])[1]?.dispatchEvent(new MouseEvent('mouseenter'))
    await settle()

    const menus = getMenus()
    expect(menus).toHaveLength(2)
    expect(getRows(menus[1]).map((row) => row.textContent?.trim())).toEqual(['Inbox', 'Archive'])
  })

  it('is placed at the right edge and the top of the row that opened it', async () => {
    mountMenu()
    await settle()
    const row = getRows(getMenus()[0])[1]!
    vi.spyOn(row, 'getBoundingClientRect').mockReturnValue({
      left: 10,
      right: 180,
      top: 55,
      bottom: 80,
      width: 170,
      height: 25,
    } as DOMRect)

    row.dispatchEvent(new MouseEvent('mouseenter'))
    await settle()

    expect(getMenus()[1]?.style.left).toBe('180px')
    expect(getMenus()[1]?.style.top).toBe('55px')
  })

  it('is drawn when the row is pressed, and nothing is chosen', async () => {
    const menu = mountMenu()
    await settle()

    getRows(getMenus()[0])[1]?.click()
    await settle()

    expect(getMenus()).toHaveLength(2)
    expect(menu.emitted('choose')).toBeUndefined()
    expect(menu.emitted('dismiss')).toBeUndefined()
  })

  it('is not drawn for a disabled row, which cannot be chosen', async () => {
    const menu = mountMenu({
      items: [
        { id: 'move', text: 'Move to', isDisabled: true, children: [{ id: 'a', text: 'A' }] },
      ],
    })
    await settle()

    getRows(getMenus()[0])[0]?.click()
    await settle()

    expect(getMenus()).toHaveLength(1)
    expect(menu.emitted('choose')).toBeUndefined()
  })

  it('is not drawn for a row whose list of children is empty', async () => {
    const menu = mountMenu({ items: [{ id: 'move', text: 'Move to', children: [] }] })
    await settle()

    getRows(getMenus()[0])[0]?.click()
    await settle()

    expect(getMenus()).toHaveLength(1)
    expect(menu.emitted('choose')).toEqual([['move']])
  })

  it('is announced on its row as a menu that can be opened', async () => {
    mountMenu()
    await settle()
    const rows = getRows(getMenus()[0])
    expect(rows[1]?.getAttribute('aria-haspopup')).toBe('menu')
    expect(rows[0]?.getAttribute('aria-haspopup')).toBeNull()
  })

  it('is opened by the right arrow on a row with children, and the keyboard goes into it', async () => {
    mountMenu({ trigger: 'keyboard' })
    await settle()
    const top = getMenus()[0]!
    top.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }))
    await settle()
    expect(document.activeElement).toBe(getRows(top)[1])

    top.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }))
    await settle()

    const menus = getMenus()
    expect(menus).toHaveLength(2)
    expect(document.activeElement).toBe(getRows(menus[1])[0])
  })

  it('is not opened by the right arrow on a row without children', async () => {
    mountMenu({ trigger: 'keyboard' })
    await settle()
    const top = getMenus()[0]!
    expect(document.activeElement).toBe(getRows(top)[0])

    top.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }))
    await settle()

    expect(getMenus()).toHaveLength(1)
  })
})

describe('choosing in a submenu', () => {
  it('returns the identifier of the child, and asks the whole menu to go', async () => {
    const menu = mountMenu()
    await settle()
    getRows(getMenus()[0])[1]?.dispatchEvent(new MouseEvent('mouseenter'))
    await settle()

    getRows(getMenus()[1])[1]?.click()
    await settle()

    expect(menu.emitted('choose')).toEqual([['archive']])
    expect(menu.emitted('dismiss')).toHaveLength(1)
    expect(getMenus()).toHaveLength(1)
  })
})

describe('putting a submenu away', () => {
  it('goes when the pointer moves onto a row that has no children', async () => {
    const menu = mountMenu()
    await settle()
    const rows = getRows(getMenus()[0])
    rows[1]?.dispatchEvent(new MouseEvent('mouseenter'))
    await settle()
    expect(getMenus()).toHaveLength(2)

    rows[2]?.dispatchEvent(new MouseEvent('mouseenter'))
    await settle()

    expect(getMenus()).toHaveLength(1)
    expect(menu.emitted('dismiss')).toBeUndefined()
  })

  it('stays where the pointer moves onto the row that opened it again', async () => {
    mountMenu()
    await settle()
    const row = getRows(getMenus()[0])[1]
    row?.dispatchEvent(new MouseEvent('mouseenter'))
    await settle()

    row?.dispatchEvent(new MouseEvent('mouseenter'))
    await settle()

    expect(getMenus()).toHaveLength(2)
  })

  it('goes on the left arrow, and the menu under it stays', async () => {
    const menu = mountMenu({ trigger: 'keyboard' })
    await settle()
    const top = getMenus()[0]!
    getRows(top)[1]?.dispatchEvent(new MouseEvent('mouseenter'))
    await settle()
    const sub = getMenus()[1]!

    sub.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', bubbles: true }))
    await settle()

    expect(getMenus()).toHaveLength(1)
    expect(menu.emitted('dismiss')).toBeUndefined()
  })

  it('ignores the left arrow in a menu that has no parent', async () => {
    const menu = mountMenu({ trigger: 'keyboard' })
    await settle()

    getMenus()[0]?.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', bubbles: true }))
    await settle()

    expect(getMenus()).toHaveLength(1)
    expect(menu.emitted('dismiss')).toBeUndefined()
  })

  it('goes with the menu that holds it', async () => {
    const menu = mountMenu()
    await settle()
    getRows(getMenus()[0])[1]?.dispatchEvent(new MouseEvent('mouseenter'))
    await settle()

    await menu.setProps({ isOpen: false })
    await settle()
    await menu.setProps({ isOpen: true })
    await settle()

    expect(getMenus()).toHaveLength(1)
  })
})

describe('the size of a menu that changes while it is open', () => {
  it('is measured again, and the menu is placed by the new size', async () => {
    let width = 100
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(
      () => ({ width, height: 50, left: 0, right: width, top: 0, bottom: 50 }) as DOMRect,
    )
    const menu = mountMenu({ at: { x: 750, y: 10 }, viewport: { width: 800, height: 600 } })
    await settle()
    const before = getMenus()[0]?.style.left

    width = 300
    await menu.setProps({ items: [...ITEMS, { id: 'more', text: 'More' }] })
    await settle()

    expect(getMenus()[0]?.style.left).not.toBe(before)
  })

  it('is not measured while it is shut', async () => {
    const measure = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect')
    const menu = mountMenu({ isOpen: false })
    await settle()
    measure.mockClear()

    await menu.setProps({ items: [...ITEMS, { id: 'more', text: 'More' }] })
    await settle()

    expect(measure).not.toHaveBeenCalled()
  })
})
