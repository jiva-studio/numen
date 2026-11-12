import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import { type AppInfo, type CommandItem, type SlotItem } from '@numen/plugin-sdk'
import TitleBar from './TitleBar.vue'
import { getControlsRoom } from '../lib/frame'
import { TITLE_MIN_WIDTH } from '../lib/layout'
import { command, separator, submenu, view } from '../testing/entries'

interface Options {
  vaultName?: string
  platform?: AppInfo['platform']
  buttons?: 'windows' | 'mac' | 'none'
  isMaximized?: boolean
  isFullscreen?: boolean
  commands?: readonly SlotItem[]
  status?: readonly SlotItem[]
  listItems?: (command: string) => Promise<readonly CommandItem[]>
  reportError?: (error: unknown) => void
}

const WINDOWS_TRAIL = getControlsRoom('windows', false).trail

const mounted: VueWrapper[] = []

const mountBar = (options: Options = {}): VueWrapper => {
  const { vaultName = 'Orchard notes', ...rest } = options
  const vault = { id: 'orchard', name: vaultName }
  const wrapper = mount(TitleBar, {
    attachTo: document.body,
    props: {
      appName: 'numen',
      vaults: { vaults: [vault], current: vaultName === '' ? null : vault },
      platform: 'windows' as AppInfo['platform'],
      isMaximized: false,
      ...rest,
    },
  })
  mounted.push(wrapper)
  return wrapper
}

const id = (name: string) => `org.example.tide:${name}`
const button = (wrapper: VueWrapper, name: string) =>
  wrapper.find(`[data-title-bar-command="${id(name)}"]`)
const shownIds = (wrapper: VueWrapper): string[] =>
  wrapper
    .findAll('[data-title-bar-command]')
    .map((one) => one.attributes('data-title-bar-command')!.replace('org.example.tide:', ''))
const control = (wrapper: VueWrapper, name: string) => wrapper.find(`[data-control="${name}"]`)

afterEach(() => {
  mounted.splice(0).forEach((one) => one.unmount())
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('the title', () => {
  it('names the application and the vault', () => {
    expect(mountBar().find('[data-title]').text()).toBe('numen — Orchard notes')
  })

  it('names only the application when no vault is open', () => {
    expect(mountBar({ vaultName: '' }).find('[data-title]').text()).toBe('numen')
  })

  it('carries the whole title as its tooltip for when the name is cut', () => {
    const wrapper = mountBar({ vaultName: 'A very long name of a vault' })

    expect(wrapper.find('[data-title]').attributes('title')).toBe(
      'numen — A very long name of a vault',
    )
  })
})

describe('the place of the title', () => {
  const cells = (wrapper: VueWrapper) =>
    wrapper
      .find('[data-title-bar-start]')
      .findAll(
        ':scope > [data-title-bar-lead], :scope > [data-title-bar-middle], :scope > [data-title-bar-trail]',
      )
      .map((one) =>
        Object.keys(one.attributes()).find((name) => name.startsWith('data-title-bar-'))!,
      )

  it('is a middle cell between the cell of the buttons and the cell of the status', () => {
    const wrapper = mountBar({ commands: [command('a')], status: [view('tasks')] })

    expect(cells(wrapper)).toStrictEqual([
      'data-title-bar-lead',
      'data-title-bar-middle',
      'data-title-bar-trail',
    ])
    expect(wrapper.find('[data-title-bar-middle] [data-title]').exists()).toBe(true)
    expect(wrapper.find('[data-title-bar-lead] [data-title-bar-command]').exists()).toBe(true)
    expect(wrapper.find('[data-title-bar-trail] [data-label-view]').exists()).toBe(true)
  })

  it('is in the middle cell whether or not the buttons are drawn', async () => {
    const wrapper = mountBar({ commands: [command('a')] })
    const before = cells(wrapper)

    await wrapper.setProps({ commands: [] })

    expect(cells(wrapper)).toStrictEqual(before)
    expect(wrapper.find('[data-title-bar-middle] [data-title]').exists()).toBe(true)
  })
})

describe('the buttons of plugins', () => {
  const PRESSED_MARKS = ['aria-pressed', 'aria-checked', 'data-pressed', 'data-active']

  it('draw no pressed state before or after a press', async () => {
    const commands = [command('chart.open'), command('map.open'), separator(1), command('log.open')]
    const wrapper = mountBar({ commands })
    const before = wrapper.find('[data-title-bar]').html()

    await button(wrapper, 'map.open').trigger('click')
    await wrapper.setProps({ commands: [...commands] })

    for (const name of shownIds(wrapper)) {
      for (const mark of PRESSED_MARKS) {
        expect(button(wrapper, name).attributes(mark)).toBeUndefined()
      }
    }
    expect(wrapper.emitted('run-command')).toStrictEqual([[id('map.open')]])
    expect(wrapper.find('[data-title-bar]').html()).toBe(before)
  })

  it('do not toggle maximize on a double click', async () => {
    const wrapper = mountBar({ commands: [command('chart.open')] })

    await button(wrapper, 'chart.open').trigger('dblclick')

    expect(wrapper.emitted('toggle-maximize-window')).toBeUndefined()
  })
})

describe('the window controls', () => {
  it('draw minimize, maximize and close in the windows style, after the title', () => {
    const wrapper = mountBar({ platform: 'windows' })
    const title = wrapper.find('[data-title]').element
    const close = control(wrapper, 'close').element

    expect(
      wrapper.findAll('[data-control]').map((one) => one.attributes('data-control')),
    ).toStrictEqual(['minimize', 'toggle-maximize', 'close'])
    expect(title.compareDocumentPosition(close) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('emit the event of the control that is pressed', async () => {
    const wrapper = mountBar({ platform: 'windows' })

    await control(wrapper, 'minimize').trigger('click')
    await control(wrapper, 'toggle-maximize').trigger('click')
    await control(wrapper, 'close').trigger('click')

    expect(wrapper.emitted('minimize-window')).toHaveLength(1)
    expect(wrapper.emitted('toggle-maximize-window')).toHaveLength(1)
    expect(wrapper.emitted('close-window')).toHaveLength(1)
  })

  it('do not toggle maximize on a double click on a control', async () => {
    const wrapper = mountBar()

    await control(wrapper, 'close').trigger('dblclick')

    expect(wrapper.emitted('toggle-maximize-window')).toBeUndefined()
  })

  it('draw the three circles of the mac style before the commands', () => {
    const wrapper = mountBar({ platform: 'linux', buttons: 'mac', commands: [command('a')] })
    const close = control(wrapper, 'close').element
    const first = button(wrapper, 'a').element

    expect(wrapper.findAll('[data-control]')).toHaveLength(3)
    expect(close.compareDocumentPosition(first) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('draw no control and keep no room in the none style', () => {
    const wrapper = mountBar({ platform: 'linux', buttons: 'none' })

    expect(wrapper.findAll('[data-control]')).toHaveLength(0)
    expect(wrapper.find('[data-mac-buttons-inset]').exists()).toBe(false)
    expect(wrapper.find('[data-title-bar]').attributes('data-controls')).toBe('none')
  })

  it('draw no control in the native-mac style', () => {
    expect(mountBar({ platform: 'darwin' }).findAll('[data-control]')).toHaveLength(0)
  })

  it('keep no inset in the other styles', () => {
    expect(mountBar({ platform: 'windows' }).find('[data-mac-buttons-inset]').exists()).toBe(false)
    expect(
      mountBar({ platform: 'linux', buttons: 'mac' }).find('[data-mac-buttons-inset]').exists(),
    ).toBe(false)
  })
})

describe('the keyboard', () => {
  const COMMANDS = [command('a'), command('b'), separator(1), command('c')]
  const focused = () => document.activeElement?.getAttribute('data-title-bar-command') ?? ''
  const press = async (key: string) => {
    document.activeElement?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true }))
    await nextTick()
  }

  it('names the row of the bar as a toolbar', () => {
    const toolbar = mountBar({ commands: COMMANDS }).find('[role="toolbar"]')

    expect(toolbar.exists()).toBe(true)
    expect(toolbar.attributes('aria-label')).toBe('Title bar')
  })

  it('puts one button of the bar in the tab order, the first', async () => {
    const wrapper = mountBar({ commands: COMMANDS })
    await nextTick()

    expect(button(wrapper, 'a').attributes('tabindex')).toBe('0')
    expect(button(wrapper, 'b').attributes('tabindex')).toBe('-1')
    expect(wrapper.find('[data-title]').attributes('tabindex')).toBe('-1')
  })

  it('moves between the buttons with the arrow keys, Home and End, and wraps', async () => {
    const wrapper = mountBar({ commands: COMMANDS })
    await nextTick()
    ;(button(wrapper, 'a').element as HTMLElement).focus()

    await press('ArrowRight')
    expect(focused()).toBe(id('b'))
    await press('ArrowRight')
    expect(focused()).toBe(id('c'))
    await press('ArrowLeft')
    expect(focused()).toBe(id('b'))
    await press('End')
    expect(document.activeElement).toBe(wrapper.find('[data-title]').element)
    await press('ArrowRight')
    expect(focused()).toBe(id('a'))
    await press('Home')
    expect(focused()).toBe(id('a'))
  })

  it('keeps the button reached last in the tab order', async () => {
    const wrapper = mountBar({ commands: COMMANDS })
    await nextTick()
    ;(button(wrapper, 'a').element as HTMLElement).focus()

    await press('ArrowRight')

    expect(button(wrapper, 'a').attributes('tabindex')).toBe('-1')
    expect(button(wrapper, 'b').attributes('tabindex')).toBe('0')
  })
})

describe('the entries that do not fit', () => {
  const BUTTON_WIDTH = 32
  const VIEW_WIDTH = 64
  let callbacks: ResizeObserverCallback[] = []
  let slotWidth = 0

  beforeEach(() => {
    callbacks = []
    vi.stubGlobal(
      'ResizeObserver',
      class {
        constructor(tell: ResizeObserverCallback) {
          callbacks.push(tell)
        }
        observe = vi.fn()
        unobserve = vi.fn()
        disconnect = vi.fn()
      },
    )
    vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockImplementation(function (
      this: HTMLElement,
    ) {
      return this.hasAttribute('data-title-bar-start') ? slotWidth : 0
    })
    vi.spyOn(HTMLElement.prototype, 'clientHeight', 'get').mockReturnValue(32)
    vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockImplementation(function (
      this: HTMLElement,
    ) {
      if (this.hasAttribute('data-title-bar-view')) return VIEW_WIDTH
      if (this.hasAttribute('data-title-bar-gap')) return 0
      return this.hasAttribute('data-title-bar-entry') || this.hasAttribute('data-title-bar-more')
        ? BUTTON_WIDTH
        : 0
    })
  })

  const settle = async (wrapper: VueWrapper) => {
    for (let round = 0; round < 4; round += 1) await wrapper.vm.$nextTick()
  }

  const resize = async (wrapper: VueWrapper, room: number): Promise<void> => {
    slotWidth = TITLE_MIN_WIDTH + room + WINDOWS_TRAIL
    const entries = [{ contentRect: { width: slotWidth, height: 32 } } as ResizeObserverEntry]
    for (const tell of callbacks) tell(entries, {} as ResizeObserver)
    await settle(wrapper)
  }

  const FIVE = [
    command('a'),
    command('b'),
    separator(1),
    command('c'),
    command('d'),
    separator(2),
    command('e'),
  ]

  const mountOverflowingBar = async (room: number, options: Options = {}): Promise<VueWrapper> => {
    slotWidth = TITLE_MIN_WIDTH + room + WINDOWS_TRAIL
    const wrapper = mountBar({ commands: FIVE, ...options })
    await resize(wrapper, room)
    return wrapper
  }

  const menuItems = (): string[] =>
    [...document.body.querySelectorAll('[role="menuitem"]')].map((one) => one.textContent!.trim())
  const shownViews = (wrapper: VueWrapper): string[] =>
    wrapper.findAll('[data-label-view]').map((one) => one.text())

  it('shows every command and no menu button when they all fit', async () => {
    const wrapper = await mountOverflowingBar(5 * BUTTON_WIDTH)

    expect(shownIds(wrapper)).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
    expect(wrapper.find('[data-title-bar-more]').exists()).toBe(false)
  })

  it('shows the first commands and a menu button when some do not fit', async () => {
    const wrapper = await mountOverflowingBar(3 * BUTTON_WIDTH)

    expect(shownIds(wrapper)).toStrictEqual(['a', 'b'])
    expect(wrapper.find('[data-title-bar-more]').exists()).toBe(true)
  })

  it('names the menu button and says it opens a menu', async () => {
    const more = (await mountOverflowingBar(3 * BUTTON_WIDTH)).find('[data-title-bar-more]')

    expect(more.attributes('aria-label')).toBe('More commands')
    expect(more.attributes('aria-haspopup')).toBe('menu')
    expect(more.attributes('aria-expanded')).toBe('false')
  })

  it('lists the commands that do not fit in the menu, in order and in their groups', async () => {
    const wrapper = await mountOverflowingBar(3 * BUTTON_WIDTH)

    await wrapper.find('[data-title-bar-more]').trigger('click')

    expect(menuItems()).toStrictEqual(['Open c', 'Open d', 'Open e'])
    expect(document.body.querySelectorAll('[role="separator"]').length).toBe(1)
    expect(wrapper.find('[data-title-bar-more]').attributes('aria-expanded')).toBe('true')
  })

  it('emits run-command with the command of the entry picked in the menu, and closes it', async () => {
    const wrapper = await mountOverflowingBar(3 * BUTTON_WIDTH)

    await wrapper.find('[data-title-bar-more]').trigger('click')
    document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')[1]!.click()
    await nextTick()

    expect(wrapper.emitted('run-command')).toStrictEqual([[id('d')]])
    expect(document.body.querySelector('[role="menu"]')).toBeNull()
  })

  const clickMore = async (wrapper: VueWrapper, detail: number) => {
    const more = wrapper.find('[data-title-bar-more]').element
    more.dispatchEvent(new MouseEvent('click', { bubbles: true, detail }))
    await settle(wrapper)
  }

  it('puts the keyboard on the first entry when the menu is opened from the keyboard', async () => {
    const wrapper = await mountOverflowingBar(3 * BUTTON_WIDTH)

    await clickMore(wrapper, 0)

    expect(document.activeElement?.textContent?.trim()).toBe('Open c')
  })

  it('puts the keyboard on no entry when the menu is opened by a pointer', async () => {
    const wrapper = await mountOverflowingBar(3 * BUTTON_WIDTH)

    await clickMore(wrapper, 1)

    expect(document.activeElement?.getAttribute('role')).not.toBe('menuitem')
  })

  it('moves the commands back to the bar when the window grows, and closes the menu', async () => {
    const wrapper = await mountOverflowingBar(3 * BUTTON_WIDTH)
    await wrapper.find('[data-title-bar-more]').trigger('click')

    await resize(wrapper, 5 * BUTTON_WIDTH)

    expect(shownIds(wrapper)).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
    expect(wrapper.find('[data-title-bar-more]').exists()).toBe(false)
    expect(document.body.querySelector('[role="menu"]')).toBeNull()
  })

  it('keeps every entry in the bar while the room is not measured', async () => {
    slotWidth = 0
    const wrapper = mountBar({ commands: FIVE, status: [view('tasks')] })
    await settle(wrapper)

    expect(shownIds(wrapper)).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
    expect(shownViews(wrapper)).toStrictEqual(['tasks'])
  })

  it('moves the commands of titleBar.status into the menu before those of titleBar.commands', async () => {
    const wrapper = await mountOverflowingBar(3 * BUTTON_WIDTH + VIEW_WIDTH, {
      commands: [command('a'), command('b'), command('c')],
      status: [view('tasks'), command('f')],
    })

    expect(shownIds(wrapper)).toStrictEqual(['a', 'b'])
    expect(shownViews(wrapper)).toStrictEqual(['tasks'])
    await wrapper.find('[data-title-bar-more]').trigger('click')
    expect(menuItems()).toStrictEqual(['Open c', 'Open f'])
  })

  it('hides the views from the edge once every command is in the menu, and never lists one in it', async () => {
    const wrapper = await mountOverflowingBar(BUTTON_WIDTH + VIEW_WIDTH, {
      commands: [command('a'), view('due')],
      status: [view('tasks'), view('stats')],
    })

    expect(shownIds(wrapper)).toStrictEqual([])
    expect(shownViews(wrapper)).toStrictEqual(['due'])
    await wrapper.find('[data-title-bar-more]').trigger('click')
    expect(menuItems()).toStrictEqual(['Open a'])
  })

  it('draws the hidden views again when the window grows', async () => {
    const wrapper = await mountOverflowingBar(VIEW_WIDTH, {
      commands: [],
      status: [view('tasks'), view('stats')],
    })
    expect(shownViews(wrapper)).toStrictEqual(['tasks'])

    await resize(wrapper, 2 * VIEW_WIDTH)

    expect(shownViews(wrapper)).toStrictEqual(['tasks', 'stats'])
  })
})

describe('a command with items', () => {
  const MOVE = command('tab.move', { hasItems: true, title: 'Move to window' })
  const items = [
    { id: 'w1', title: 'Window 1' },
    { id: 'w2', title: 'Window 2' },
  ]
  const menuTexts = (): string[] =>
    [...document.body.querySelectorAll('[role="menuitem"]')].map((one) => one.textContent!.trim())
  const settle = async () => {
    for (let round = 0; round < 4; round += 1) await nextTick()
  }

  it('opens a dropdown of the items the command lists', async () => {
    const listItems = vi.fn(async () => items)
    const wrapper = mountBar({ commands: [MOVE], listItems })

    await wrapper.find(`[data-title-bar-command="${MOVE.command}"]`).trigger('click')
    await settle()

    expect(listItems).toHaveBeenCalledWith(MOVE.command)
    expect(menuTexts()).toStrictEqual(['Window 1', 'Window 2'])
  })

  it('opens no dropdown while the items are read, and one with a loading row once the hold has passed', async () => {
    vi.useFakeTimers()
    const wrapper = mountBar({ commands: [MOVE], listItems: () => new Promise(() => undefined) })

    await wrapper.find(`[data-title-bar-command="${MOVE.command}"]`).trigger('click')
    await settle()
    expect(menuTexts()).toStrictEqual([])
    await vi.advanceTimersByTimeAsync(150)
    await settle()
    vi.useRealTimers()

    expect(menuTexts()).toStrictEqual(['Loading…'])
  })

  it('opens no dropdown where the command lists no item', async () => {
    const wrapper = mountBar({ commands: [MOVE], listItems: async () => [] })

    await wrapper.find(`[data-title-bar-command="${MOVE.command}"]`).trigger('click')
    await settle()

    expect(menuTexts()).toStrictEqual([])
  })

  it('opens on the next click after a click whose command listed no item', async () => {
    const listItems = vi.fn<() => Promise<readonly CommandItem[]>>(async () => [])
    const wrapper = mountBar({ commands: [MOVE], listItems })
    await wrapper.find(`[data-title-bar-command="${MOVE.command}"]`).trigger('click')
    await settle()

    listItems.mockResolvedValue(items)
    await wrapper.find(`[data-title-bar-command="${MOVE.command}"]`).trigger('click')
    await settle()

    expect(menuTexts()).toStrictEqual(['Window 1', 'Window 2'])
  })

  it('reports a list that failed to read', async () => {
    const failure = new Error('the windows are gone')
    const reportError = vi.fn()
    const wrapper = mountBar({
      commands: [MOVE],
      listItems: () => Promise.reject(failure),
      reportError,
    })

    await wrapper.find(`[data-title-bar-command="${MOVE.command}"]`).trigger('click')
    await settle()

    expect(reportError).toHaveBeenCalledExactlyOnceWith(failure)
  })

  it('emits run-command with the command and the item picked as its itemId', async () => {
    const wrapper = mountBar({ commands: [MOVE], listItems: async () => items })

    await wrapper.find(`[data-title-bar-command="${MOVE.command}"]`).trigger('click')
    await settle()
    document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')[1]!.click()
    await nextTick()

    expect(wrapper.emitted('run-command')).toStrictEqual([[MOVE.command, 'w2']])
  })

  it('draws a command with items inside a submenu as a submenu of its items', async () => {
    const wrapper = mountBar({
      commands: [submenu('more', [MOVE], { title: 'More' })],
      listItems: async () => items,
    })

    await wrapper.find('[data-title-bar-submenu="more"]').trigger('click')
    await settle()

    expect(menuTexts()).toStrictEqual(['Move to window ›'])
    expect(document.body.querySelector('[role="menuitem"]')?.getAttribute('aria-haspopup')).toBe(
      'menu',
    )
  })
})
