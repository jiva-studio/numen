/**
 * What the menu does, not where it puts things. The negatives matter most:
 * a menu that will not go away, and one the keyboard can walk out of, both
 * look right in a picture.
 *
 * It is drawn at the end of the document, so everything here is read off the
 * document rather than off the wrapper.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import Menu from './Menu.vue'
import type { MenuItem } from './item'

const ITEMS: MenuItem[] = [
  { id: 'open', text: 'Open' },
  { id: 'child', text: 'New child note' },
  { id: 'copy', text: 'Copy path' },
]

type MenuProps = InstanceType<typeof Menu>['$props']

let mounted: { unmount: () => void } | null = null

const mountMenu = (props: Partial<MenuProps> = {}) => {
  const menu = mount(Menu, {
    props: { items: ITEMS, at: { x: 40, y: 40 }, open: true, ...props },
    attachTo: document.body,
  })
  mounted = menu
  return menu
}

/** Twice: the menu measures and takes the keyboard a tick after it is opened. */
const settle = async () => {
  await nextTick()
  await nextTick()
}

const getDrawnMenu = () => document.body.querySelector<HTMLElement>('.menu')
const choices = () => Array.from(document.body.querySelectorAll<HTMLElement>('.menu__item'))

afterEach(() => {
  mounted?.unmount()
  mounted = null
})

describe('being open and being closed', () => {
  it('draws nothing at all until it is opened', async () => {
    const menu = mountMenu({ open: false })
    expect(getDrawnMenu()).toBeNull()

    await menu.setProps({ open: true })
    await settle()
    expect(getDrawnMenu()).not.toBeNull()
  })

  it('draws itself outside whatever asked for it', async () => {
    mountMenu()
    await settle()
    expect(getDrawnMenu()?.parentElement).toBe(document.body)
  })

  it('goes when the caller says so, and takes its listeners with it', async () => {
    const menu = mountMenu()
    await settle()

    await menu.setProps({ open: false })
    expect(getDrawnMenu()).toBeNull()

    // Nothing left behind answering for a menu nobody has open.
    window.dispatchEvent(new Event('resize'))
    expect(menu.emitted('dismiss')).toBeUndefined()
  })
})

describe('choosing an item', () => {
  it('hands back the identifier it was given, and asks to be put away', async () => {
    const menu = mountMenu()
    await settle()

    choices()[1]?.click()
    expect(menu.emitted('choose')).toStrictEqual([['child']])
    expect(menu.emitted('dismiss')).toHaveLength(1)
  })

  it('says nothing for an item that cannot be chosen', async () => {
    const menu = mountMenu({
      items: [{ id: 'open', text: 'Open', disabled: true }],
    })
    await settle()

    choices()[0]?.click()
    expect(menu.emitted('choose')).toBeUndefined()
    expect(menu.emitted('dismiss')).toBeUndefined()
  })
})

describe('being put away', () => {
  it('asks to go on Escape', async () => {
    const menu = mountMenu()
    await settle()

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(menu.emitted('dismiss')).toHaveLength(1)
  })

  it('asks to go under a pointer outside it, and stays under one inside', async () => {
    const menu = mountMenu()
    await settle()

    getDrawnMenu()!.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    expect(menu.emitted('dismiss')).toBeUndefined()

    document.body.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    expect(menu.emitted('dismiss')).toHaveLength(1)
  })

  it('asks to go when the area it stands over is resized', async () => {
    const menu = mountMenu()
    await settle()

    window.dispatchEvent(new Event('resize'))
    expect(menu.emitted('dismiss')).toHaveLength(1)
  })

  it('asks to go when the page under it scrolls, and stays when its own list does', async () => {
    const menu = mountMenu()
    await settle()

    getDrawnMenu()!.dispatchEvent(new Event('scroll'))
    expect(menu.emitted('dismiss')).toBeUndefined()

    document.dispatchEvent(new Event('scroll'))
    expect(menu.emitted('dismiss')).toHaveLength(1)
  })

  it('is left alone by any other key', async () => {
    const menu = mountMenu()
    await settle()

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'a' }))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect(menu.emitted('dismiss')).toBeUndefined()
  })
})

describe('where the keyboard is while it is open', () => {
  /** Opened from the keyboard, which is the opening that lands on an item. */
  const openMenu = (props: Partial<MenuProps> = {}) => mountMenu({ opening: 'keyboard', ...props })

  it('is on the first item that can be chosen', async () => {
    openMenu({ items: [{ id: 'open', text: 'Open', disabled: true }, ...ITEMS] })
    await settle()
    expect(document.activeElement).toBe(choices()[1])
  })

  it('is on the menu itself when there is nothing to be on', async () => {
    openMenu({ items: [] })
    await settle()
    expect(document.activeElement).toBe(getDrawnMenu())
  })

  it('walks the items with the arrows, and wraps', async () => {
    openMenu()
    await settle()
    const menu = getDrawnMenu()!

    menu.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }))
    expect(document.activeElement).toBe(choices()[1])

    menu.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true }))
    expect(document.activeElement).toBe(choices()[0])

    menu.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true }))
    expect(document.activeElement).toBe(choices()[2])
  })

  it('goes to the ends on Home and End', async () => {
    openMenu()
    await settle()
    const menu = getDrawnMenu()!

    menu.dispatchEvent(new KeyboardEvent('keydown', { key: 'End', bubbles: true }))
    expect(document.activeElement).toBe(choices()[2])

    menu.dispatchEvent(new KeyboardEvent('keydown', { key: 'Home', bubbles: true }))
    expect(document.activeElement).toBe(choices()[0])
  })

  it('is kept inside: tab moves within the items rather than out of them', async () => {
    openMenu()
    await settle()
    const menu = getDrawnMenu()!

    const tab = () => {
      const event = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true })
      menu.dispatchEvent(event)
      return event
    }

    expect(tab().defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(choices()[1])
    tab()
    tab()
    expect(document.activeElement).toBe(choices()[0])
  })
})

describe('where the keyboard is in a menu opened by hand', () => {
  const step = (key: string) =>
    getDrawnMenu()?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true }))

  it('is on no item at all, and on the menu itself', async () => {
    mountMenu()
    await settle()
    expect(choices().some((item) => item === document.activeElement)).toBe(false)
    expect(document.activeElement).toBe(getDrawnMenu())
  })

  it('goes to the first item on the first step down', async () => {
    mountMenu()
    await settle()

    step('ArrowDown')
    expect(document.activeElement).toBe(choices()[0])
  })

  it('goes to the last on the first step up', async () => {
    mountMenu()
    await settle()

    step('ArrowUp')
    expect(document.activeElement).toBe(choices()[2])
  })
})

describe('where the keyboard goes when it closes', () => {
  const opener = () => {
    const button = document.createElement('button')
    document.body.append(button)
    button.focus()
    return button
  }

  it('goes back to what the menu was opened from', async () => {
    const from = opener()
    const menu = mountMenu({ from })
    await settle()
    expect(document.activeElement).not.toBe(from)

    await menu.setProps({ open: false })
    expect(document.activeElement).toBe(from)
    from.remove()
  })

  it('goes back when the menu is taken away while it is still open', async () => {
    const from = opener()
    const menu = mountMenu({ from })
    await settle()

    menu.unmount()
    mounted = null
    expect(document.activeElement).toBe(from)
    from.remove()
  })

  it('is left where it is when what it was opened from has gone', async () => {
    const from = opener()
    const menu = mountMenu({ from })
    await settle()
    from.remove()

    await menu.setProps({ open: false })
    expect(document.activeElement).not.toBe(from)
  })
})

describe('where it is drawn', () => {
  it('is placed against the area it is drawn into, not against its caller', async () => {
    mountMenu({ at: { x: 30, y: 20 }, viewport: { width: 900, height: 700 } })
    await settle()
    expect(getDrawnMenu()?.style.left).toBe('30px')
    expect(getDrawnMenu()?.style.top).toBe('20px')
  })

  it('is brought in off an edge it was asked for right against', async () => {
    mountMenu({ at: { x: 0, y: 0 }, viewport: { width: 900, height: 700 }, margin: 12 })
    await settle()
    expect(getDrawnMenu()?.style.left).toBe('12px')
    expect(getDrawnMenu()?.style.top).toBe('12px')
  })
})

describe('when there is nothing to choose', () => {
  it('says so rather than drawing an empty box', async () => {
    mountMenu({ items: [] })
    await settle()
    expect(getDrawnMenu()?.querySelector('.menu__silence')?.textContent?.trim()).toBe(
      'Nothing to do',
    )
  })

  it('says it in the words it was given', async () => {
    mounted = mount(Menu, {
      props: { items: [], at: { x: 0, y: 0 }, open: true },
      slots: { silence: 'ничего' },
      attachTo: document.body,
    })
    await settle()
    expect(getDrawnMenu()?.querySelector('.menu__silence')?.textContent?.trim()).toBe('ничего')
  })
})

describe('the groups the items stand in', () => {
  const SHELVED: MenuItem[] = [
    { id: 'numen', text: 'Numen', group: 'Ships with numen' },
    { id: 'sea', text: 'Sea', group: 'Yours' },
    { id: 'sand', text: 'Sand', group: 'Yours' },
  ]

  const shelves = () =>
    Array.from(document.body.querySelectorAll<HTMLElement>('.menu__group-name')).map((one) =>
      one.textContent?.trim(),
    )

  it('are named where the menu is told to name them', async () => {
    mountMenu({ items: SHELVED, hasGroups: true })
    await settle()
    expect(shelves()).toStrictEqual(['Ships with numen', 'Yours'])
    expect(document.body.querySelectorAll('.menu__rule')).toHaveLength(0)
  })

  it('are parted by a rule where the menu names none', async () => {
    mountMenu({ items: SHELVED })
    await settle()
    expect(shelves()).toStrictEqual([])
    expect(document.body.querySelectorAll('.menu__rule')).toHaveLength(1)
  })
})

describe('what an item says beside its words', () => {
  it('is drawn under them where the item carries one', async () => {
    mountMenu({ items: [{ id: 'small', text: 'Small', detail: 'somewhere/small.onnx' }] })
    await settle()
    expect(document.body.querySelector('.menu__detail')?.textContent?.trim()).toBe(
      'somewhere/small.onnx',
    )
  })

  it('is drawn nowhere where the item carries none', async () => {
    mountMenu()
    await settle()
    expect(document.body.querySelector('.menu__detail')).toBeNull()
  })
})

describe('typing to jump', () => {
  const typeLetter = (letter: string) =>
    getDrawnMenu()?.dispatchEvent(new KeyboardEvent('keydown', { key: letter, bubbles: true }))

  it('lands on the first item the letter begins', async () => {
    mountMenu()
    await settle()
    typeLetter('c')
    await nextTick()
    expect(document.activeElement).toBe(choices()[2])
  })

  it('walks the items one letter begins', async () => {
    mountMenu({ items: [...ITEMS, { id: 'cut', text: 'Cut' }] })
    await settle()
    typeLetter('c')
    await nextTick()
    typeLetter('c')
    await nextTick()
    expect(document.activeElement).toBe(choices()[3])
  })

  it('lands on nothing where no item begins with it', async () => {
    mountMenu()
    await settle()
    typeLetter('z')
    await nextTick()
    expect(document.activeElement).toBe(getDrawnMenu())
  })

  it('leaves the space bar to the item it is on', async () => {
    mountMenu()
    await settle()
    typeLetter(' ')
    await nextTick()
    expect(document.activeElement).toBe(getDrawnMenu())
  })
})

describe('the width it is told to keep to', () => {
  it('carries the width of what asked for it into its own rule', async () => {
    mountMenu({ asking: 420 })
    await settle()
    expect(getDrawnMenu()?.style.getPropertyValue('--asking')).toBe('420px')
  })

  it('asks for nothing where nothing said how wide it asked', async () => {
    mountMenu()
    await settle()
    expect(getDrawnMenu()?.style.getPropertyValue('--asking')).toBe('0px')
  })
})
