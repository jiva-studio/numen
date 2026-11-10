/**
 * The keyboard in an open menu, without the menu drawn around it.
 */
import { describe, expect, it, vi } from 'vitest'
import { effectScope, shallowRef } from 'vue'
import type { MenuItem } from '../items/item'
import { useMenuKeys } from './keys'
import { useMenuNavigation } from './useMenuNavigation'

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

  it('focuses the menu once the row is released', () => {
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
  it('keeps its position for a letter no item begins with', () => {
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
    ['ArrowRight', {}],
    ['ArrowLeft', {}],
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

  it('opens and dismisses submenus on ArrowRight and ArrowLeft', () => {
    const parent: MenuItem = { id: 'sub', text: 'Sub', children: [{ id: 'c', text: 'Child' }] }
    const emptyParent: MenuItem = { id: 'empty', text: 'Empty', children: [] }
    const items: MenuItem[] = [parent, emptyParent, ITEMS[0]!]
    const onOpenSubmenu = vi.fn<(item: MenuItem) => void>()
    const onDismissSubmenu = vi.fn<() => void>()
    const menu = shallowRef<HTMLElement | null>(createMenuElement())
    const keys = useMenuKeys(
      () => items,
      menu,
      () => 0,
      { onOpenSubmenu, onDismissSubmenu },
    )

    // No item selected yet
    const rightNone = createKey('ArrowRight')
    keys.onKey(rightNone)
    expect(onOpenSubmenu).not.toHaveBeenCalled()
    expect(rightNone.defaultPrevented).toBe(false)

    // Item with children
    keys.goTo(0)
    const rightParent = createKey('ArrowRight')
    keys.onKey(rightParent)
    expect(onOpenSubmenu).toHaveBeenCalledWith(parent)
    expect(rightParent.defaultPrevented).toBe(true)

    // Item with empty children
    keys.goTo(1)
    const rightEmpty = createKey('ArrowRight')
    keys.onKey(rightEmpty)
    expect(onOpenSubmenu).toHaveBeenCalledTimes(1)
    expect(rightEmpty.defaultPrevented).toBe(false)

    // Leaf item
    keys.goTo(2)
    keys.onKey(createKey('ArrowRight'))
    expect(onOpenSubmenu).toHaveBeenCalledTimes(1)

    // ArrowLeft dismisses submenu
    const leftEvent = createKey('ArrowLeft')
    keys.onKey(leftEvent)
    expect(onDismissSubmenu).toHaveBeenCalledTimes(1)
    expect(leftEvent.defaultPrevented).toBe(true)

    // Without callbacks in submenuActions
    const emptyActionsKeys = useMenuKeys(
      () => items,
      menu,
      () => 0,
      {},
    )
    emptyActionsKeys.goTo(0)
    const rightNoCb = createKey('ArrowRight')
    emptyActionsKeys.onKey(rightNoCb)
    expect(rightNoCb.defaultPrevented).toBe(false)
    const leftNoCb = createKey('ArrowLeft')
    emptyActionsKeys.onKey(leftNoCb)
    expect(leftNoCb.defaultPrevented).toBe(false)
  })
})

describe('useMenuNavigation', () => {
  it('manages submenu state, choosing, resetting, and scope disposal', () => {
    const parent: MenuItem = { id: 'sub', text: 'Sub', children: [{ id: 'c', text: 'Child' }] }
    const disabledParent: MenuItem = {
      id: 'dis-sub',
      text: 'Dis',
      isDisabled: true,
      children: [{ id: 'c2', text: 'Child 2' }],
    }
    const emptyParent: MenuItem = { id: 'empty', text: 'Empty', children: [] }
    const disabledLeaf: MenuItem = { id: 'dis', text: 'Disabled', isDisabled: true }
    const leaf: MenuItem = { id: 'leaf', text: 'Leaf' }
    const items = [parent, disabledParent, emptyParent, disabledLeaf, leaf]
    const menuEl = createMenuElement()
    const childInMenu = document.createElement('span')
    menuEl.appendChild(childInMenu)
    const submenuEl = createMenuElement()
    const childInSubmenu = document.createElement('span')
    submenuEl.appendChild(childInSubmenu)

    const onEmitChoose = vi.fn<(id: string) => void>()
    const onEmitDismiss = vi.fn<() => void>()
    const scope = effectScope()
    const nav = scope.run(() =>
      useMenuNavigation({
        items: () => items,
        trigger: () => 'keyboard',
        current: () => null,
        clock: () => 1000,
        menuRef: shallowRef(menuEl),
        getFallbackPosition: () => ({ x: 10, y: 20 }),
        getSubmenuPosition: (el) => (el ? { x: 50, y: 60 } : null),
        onEmitChoose,
        onEmitDismiss,
        hasFrom: () => true,
      }),
    )!

    expect(nav.submenu.trigger.value).toBe('pointer')
    expect(nav.submenu.active.value).toBeNull()

    // Disabled or empty children do not open submenu
    nav.submenu.onChoose(disabledParent)
    nav.submenu.onChoose(disabledLeaf)
    expect(nav.submenu.active.value).toBeNull()
    expect(onEmitChoose).not.toHaveBeenCalled()

    // Choosing parent without registered row uses fallback position
    nav.submenu.onChoose(parent)
    expect(nav.submenu.active.value).toEqual({
      id: 'sub',
      items: parent.children,
      at: { x: 10, y: 20 },
      from: null,
    })
    expect(nav.submenu.trigger.value).toBe('pointer')

    // Register row without getElement, then with getElement
    nav.submenu.onRegisterRow('sub', document.createElement('div'))
    nav.submenu.onChoose(parent)
    expect(nav.submenu.active.value?.from).toBeNull()

    const rowEl = document.createElement('button')
    nav.submenu.onRegisterRow(
      'sub',
      Object.assign(document.createElement('button'), { getElement: () => rowEl }),
    )
    nav.focusInitial()
    expect(nav.here.value).toBe(0)
    nav.onKey(createKey('ArrowRight'))
    expect(nav.submenu.trigger.value).toBe('keyboard')
    expect(nav.submenu.active.value?.from).toBe(rowEl)
    expect(nav.submenu.active.value?.at).toEqual({ x: 50, y: 60 })

    // Choosing disabledParent or emptyParent via keyboard does not change active submenu
    nav.submenu.onClose()
    expect(nav.submenu.active.value).toBeNull()
    nav.here.value = 1
    nav.onKey(createKey('ArrowRight'))
    expect(nav.submenu.active.value).toBeNull()
    nav.onKey(createKey('ArrowDown'))
    expect(nav.here.value).toBe(2)
    nav.onKey(createKey('ArrowRight'))
    expect(nav.submenu.active.value).toBeNull()

    // Submenu choose closes submenu and emits choose + dismiss
    nav.submenu.onChoose(parent)
    expect(nav.submenu.active.value).not.toBeNull()
    nav.submenu.onSubmenuChoose('c')
    expect(nav.submenu.active.value).toBeNull()
    expect(onEmitChoose).toHaveBeenCalledWith('c')
    expect(onEmitDismiss).toHaveBeenCalledTimes(1)

    // Leaf choose emits choose + dismiss
    nav.submenu.onChoose(leaf)
    expect(onEmitChoose).toHaveBeenCalledWith('leaf')
    expect(onEmitDismiss).toHaveBeenCalledTimes(2)

    // Reset navigation resets here to -1 and closes submenu
    nav.submenu.onChoose(parent)
    nav.onResetNavigation()
    expect(nav.here.value).toBe(-1)
    expect(nav.submenu.active.value).toBeNull()

    // Exercise aim callbacks when getSubmenuBox/getSubmenuElement are omitted
    nav.submenu.onChoose(parent)
    nav.submenu.onMenuLeave(new MouseEvent('mouseleave', { clientX: 5, clientY: 5 }))

    const removeSpy = vi.spyOn(window, 'removeEventListener')
    scope.stop()
    expect(removeSpy).toHaveBeenCalledWith('mousemove', expect.any(Function))
    removeSpy.mockRestore()
  })

  it('forwards getSubmenuBox and getSubmenuElement to submenu aim', () => {
    vi.useFakeTimers()
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => undefined)
    let now = 1000
    const parent: MenuItem = { id: 'sub', text: 'Sub', children: [{ id: 'c', text: 'Child' }] }
    const menuEl = createMenuElement()
    const submenuEl = createMenuElement()
    const childInSubmenu = document.createElement('span')
    submenuEl.appendChild(childInSubmenu)

    const nav = useMenuNavigation({
      items: () => [parent],
      trigger: () => 'pointer',
      current: () => null,
      clock: () => now,
      menuRef: shallowRef(menuEl),
      getFallbackPosition: () => ({ x: 10, y: 20 }),
      getSubmenuPosition: () => ({ x: 10, y: 20 }),
      getSubmenuBox: () => ({ left: 100, right: 200, top: 50, bottom: 150 }),
      getSubmenuElement: () => submenuEl,
      onEmitChoose: vi.fn(),
      onEmitDismiss: vi.fn(),
      hasFrom: () => false,
    })
    expect(warnSpy).not.toHaveBeenCalled()
    warnSpy.mockRestore()

    // Hovering schedules open; calling onClose cancels the pending open
    nav.submenu.onRowHover(parent, new MouseEvent('mouseenter', { clientX: 10, clientY: 10 }))
    nav.submenu.onClose()
    now += 200
    vi.advanceTimersByTime(200)
    expect(nav.submenu.active.value).toBeNull()

    // Opening submenu binds window listener so moving outside closes after grace
    nav.submenu.onChoose(parent)
    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 5, clientY: 5 }))
    now += 400
    vi.advanceTimersByTime(400)
    expect(nav.submenu.active.value).toBeNull()

    nav.submenu.onChoose(parent)
    // Leaving into the submenu box keeps the submenu open without scheduling close
    nav.submenu.onMenuLeave(new MouseEvent('mouseleave', { clientX: 150, clientY: 100 }))
    now += 400
    vi.advanceTimersByTime(400)
    expect(nav.submenu.active.value?.id).toBe('sub')

    // Leaving into an event target inside submenuEl also keeps it open
    const evt = new MouseEvent('mouseleave', { clientX: 10, clientY: 10 })
    Object.defineProperty(evt, 'target', { value: childInSubmenu })
    nav.submenu.onMenuLeave(evt)
    now += 400
    vi.advanceTimersByTime(400)
    expect(nav.submenu.active.value?.id).toBe('sub')

    // Window pointermove inside menuEl does not schedule close
    const moveInside = new MouseEvent('mousemove', { clientX: 10, clientY: 10 })
    Object.defineProperty(moveInside, 'target', { value: menuEl })
    window.dispatchEvent(moveInside)
    now += 400
    vi.advanceTimersByTime(400)
    expect(nav.submenu.active.value?.id).toBe('sub')
    nav.submenu.onClose()
    vi.useRealTimers()
  })
})
