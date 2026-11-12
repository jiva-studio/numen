/**
 * The menu inside a menu: what opens it, what is chosen in it, and what puts
 * it away. The negatives matter most: a row with a submenu is not an item to
 * choose, and the key that closes a submenu must not close the menu under it.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMenuMount, settle } from './fixtures/mount'
import Menu from './Menu.vue'
import { SUBMENU_CLOSE_GRACE_MS, SUBMENU_OPEN_DELAY_MS } from './interaction/aim'
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

const { mountMenu, advanceClock } = createMenuMount(ITEMS)

const getMenus = () => Array.from(document.body.querySelectorAll<HTMLElement>('.menu'))
const getRows = (menu: HTMLElement | undefined) =>
  Array.from(menu?.querySelectorAll<HTMLElement>('.menu__item') ?? [])

const restOnRow = async (
  row: HTMLElement | undefined,
  point: { clientX?: number; clientY?: number } = {},
) => {
  row?.dispatchEvent(new MouseEvent('mouseenter', point))
  advanceClock(SUBMENU_OPEN_DELAY_MS)
  row?.dispatchEvent(new MouseEvent('mousemove', point))
  await settle()
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('opening a submenu', () => {
  it('opens after resting on a row with children for SUBMENU_OPEN_DELAY_MS', async () => {
    mountMenu()
    await settle()
    expect(getMenus()).toHaveLength(1)

    const row = getRows(getMenus()[0])[1]
    row?.dispatchEvent(new MouseEvent('mouseenter'))
    await settle()
    expect(getMenus()).toHaveLength(1)

    advanceClock(SUBMENU_OPEN_DELAY_MS)
    row?.dispatchEvent(new MouseEvent('mousemove'))
    await settle()

    const menus = getMenus()
    expect(menus).toHaveLength(2)
    expect(getRows(menus[1]).map((r) => r.textContent?.trim())).toEqual(['Inbox', 'Archive'])
  })

  it('does not open when passing over a parent row before the delay', async () => {
    mountMenu()
    await settle()
    const rows = getRows(getMenus()[0])

    rows[1]?.dispatchEvent(new MouseEvent('mouseenter', { clientX: 80, clientY: 65 }))
    advanceClock(SUBMENU_OPEN_DELAY_MS - 30)
    rows[2]?.dispatchEvent(new MouseEvent('mouseenter', { clientX: 80, clientY: 95 }))
    advanceClock(SUBMENU_OPEN_DELAY_MS)
    rows[2]?.dispatchEvent(new MouseEvent('mousemove', { clientX: 80, clientY: 95 }))
    await settle()

    expect(getMenus()).toHaveLength(1)
  })

  it('is placed at the right edge and the top of the row that opened it', async () => {
    const menu = mountMenu()
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

    await restOnRow(row)

    expect(menu.findAllComponents(Menu).at(-1)?.props('at')).toStrictEqual({ x: 180, y: 55 })
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
    await restOnRow(getRows(getMenus()[0])[1])

    getRows(getMenus()[1])[1]?.click()
    await settle()

    expect(menu.emitted('choose')).toEqual([['archive']])
    expect(menu.emitted('dismiss')).toHaveLength(1)
    expect(getMenus()).toHaveLength(1)
  })
})

describe('putting a submenu away', () => {
  it('goes when moving onto a leaf row outside the safe triangle', async () => {
    const menu = mountMenu()
    await settle()
    const rows = getRows(getMenus()[0])
    await restOnRow(rows[1], { clientX: 80, clientY: 60 })
    expect(getMenus()).toHaveLength(2)

    rows[2]?.dispatchEvent(new MouseEvent('mouseenter', { clientX: 80, clientY: 95 }))
    await settle()

    expect(getMenus()).toHaveLength(1)
    expect(menu.emitted('dismiss')).toBeUndefined()
  })

  it('keeps the submenu open inside the safe triangle until the pointer rests', async () => {
    mountMenu()
    await settle()
    const rows = getRows(getMenus()[0])
    await restOnRow(rows[1], { clientX: 100, clientY: 60 })
    const sub = getMenus()[1]!
    vi.spyOn(sub, 'getBoundingClientRect').mockReturnValue({
      left: 200,
      right: 360,
      top: 40,
      bottom: 200,
      width: 160,
      height: 160,
    } as DOMRect)

    rows[2]?.dispatchEvent(new MouseEvent('mouseenter', { clientX: 150, clientY: 95 }))
    await settle()
    expect(getMenus()).toHaveLength(2)

    advanceClock(SUBMENU_OPEN_DELAY_MS)
    rows[2]?.dispatchEvent(new MouseEvent('mousemove', { clientX: 150, clientY: 95 }))
    await settle()
    expect(getMenus()).toHaveLength(1)
  })

  it('delays close on menu leave and cancels close when reaching the submenu', async () => {
    mountMenu()
    await settle()
    const top = getMenus()[0]!
    const rows = getRows(top)
    await restOnRow(rows[1], { clientX: 100, clientY: 60 })
    const sub = getMenus()[1]!
    vi.spyOn(sub, 'getBoundingClientRect').mockReturnValue({
      left: 200,
      right: 360,
      top: 40,
      bottom: 200,
      width: 160,
      height: 160,
    } as DOMRect)

    top.dispatchEvent(new MouseEvent('mouseleave', { clientX: 30, clientY: 60 }))
    await settle()
    expect(getMenus()).toHaveLength(2)

    advanceClock(100)
    sub.dispatchEvent(new MouseEvent('mousemove', { bubbles: true, clientX: 220, clientY: 80 }))
    advanceClock(SUBMENU_CLOSE_GRACE_MS)
    await settle()
    expect(getMenus()).toHaveLength(2)

    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 10, clientY: 10 }))
    advanceClock(SUBMENU_CLOSE_GRACE_MS)
    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 5, clientY: 10 }))
    await settle()
    expect(getMenus()).toHaveLength(1)
  })

  it('stays where the pointer moves onto the row that opened it again', async () => {
    mountMenu()
    await settle()
    const row = getRows(getMenus()[0])[1]
    await restOnRow(row)

    row?.dispatchEvent(new MouseEvent('mouseenter'))
    await settle()

    expect(getMenus()).toHaveLength(2)
  })

  it('goes on the left arrow, and the menu under it stays', async () => {
    const menu = mountMenu({ trigger: 'keyboard' })
    await settle()
    const top = getMenus()[0]!
    await restOnRow(getRows(top)[1])
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
    await restOnRow(getRows(getMenus()[0])[1])

    await menu.setProps({ isOpen: false })
    await settle()
    await menu.setProps({ isOpen: true })
    await settle()

    expect(getMenus()).toHaveLength(1)
  })
})

describe('the size of a menu that changes while it is open', () => {
  it('is measured again when its items change', async () => {
    const measure = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect')
    const menu = mountMenu({ at: { x: 750, y: 10 }, viewport: { width: 800, height: 600 } })
    await settle()
    measure.mockClear()

    await menu.setProps({ items: [...ITEMS, { id: 'more', text: 'More' }] })
    await settle()

    expect(measure).toHaveBeenCalled()
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

describe('the highlight event', () => {
  it('emits the item id under the pointer once per change, for the menu and its submenu, and null when the pointer leaves', async () => {
    const menu = mountMenu()
    await settle()
    const rows = getRows(getMenus()[0])

    rows[0]?.dispatchEvent(new MouseEvent('mouseenter'))
    rows[0]?.dispatchEvent(new MouseEvent('mousemove'))
    await restOnRow(rows[1])
    getRows(getMenus()[1])[1]?.dispatchEvent(new MouseEvent('mouseenter'))
    getMenus()[1]?.dispatchEvent(new MouseEvent('mouseleave'))
    await settle()

    expect(menu.emitted('highlight')).toEqual([['open'], ['move'], ['archive'], [null]])
  })
})

describe('the active row with a submenu', () => {
  it('stays on the parent row while its submenu is open and the pointer is inside it', async () => {
    mountMenu()
    await settle()
    const parent = getRows(getMenus()[0])[1]
    await restOnRow(parent, { clientX: 5, clientY: 5 })

    getMenus()[0]?.dispatchEvent(new MouseEvent('mouseleave'))
    await settle()

    expect(parent?.hasAttribute('data-active')).toBe(true)
    expect(getMenus()[0]?.querySelectorAll('[data-active]')).toHaveLength(1)
  })
})
