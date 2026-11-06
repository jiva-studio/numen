import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import TitleBar from './TitleBar.vue'
import { TITLE_MIN_WIDTH } from '../lib/layout'
import type { TitleBarViewProps, TitleBarCommand } from '@numen/plugin-sdk'

const command = (id: string, group?: string, shortcut?: string): TitleBarCommand => ({
  id,
  title: `Command ${id}`,
  icon: { render: () => null },
  ...(group === undefined ? {} : { group }),
  ...(shortcut === undefined ? {} : { shortcut }),
})

interface Options {
  vaultName?: string
  platform?: TitleBarViewProps['platform']
  buttons?: 'windows' | 'mac'
  isMaximized?: boolean
  commands?: readonly TitleBarCommand[]
}

const mountBar = (options: Options = {}): VueWrapper =>
  mount(TitleBar, {
    attachTo: document.body,
    props: {
      appName: 'numen',
      vaultName: 'Orchard notes',
      platform: 'windows' as TitleBarViewProps['platform'],
      isMaximized: false,
      commands: [],
      ...options,
    },
  })

const button = (wrapper: VueWrapper, id: string) => wrapper.find(`[data-title-bar-command="${id}"]`)
const shownIds = (wrapper: VueWrapper): string[] =>
  wrapper
    .findAll('[data-title-bar-command]')
    .map((one) => one.attributes('data-title-bar-command')!)
const control = (wrapper: VueWrapper, name: string) => wrapper.find(`[data-control="${name}"]`)

afterEach(() => {
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
})

describe('the title', () => {
  it('names the application and the vault', () => {
    expect(mountBar().find('[data-title]').text()).toBe('numen — Orchard notes')
  })

  it('names only the application when no vault is open', () => {
    expect(mountBar({ vaultName: '' }).find('[data-title]').text()).toBe('numen')
  })

  it('changes at once when the vault changes', async () => {
    const wrapper = mountBar()

    await wrapper.setProps({ vaultName: 'Greenhouse log' })

    expect(wrapper.find('[data-title]').text()).toBe('numen — Greenhouse log')
  })

  it('carries the whole title as its tooltip for when the name is cut', () => {
    const wrapper = mountBar({ vaultName: 'A very long name of a vault' })

    expect(wrapper.find('[data-title]').attributes('title')).toBe(
      'numen — A very long name of a vault',
    )
  })
})

describe('the commands', () => {
  it('draw in the order given', () => {
    const wrapper = mountBar({ commands: [command('a'), command('b'), command('c')] })

    expect(shownIds(wrapper)).toStrictEqual(['a', 'b', 'c'])
  })

  it('mark the first command of each group after the first group', () => {
    const wrapper = mountBar({
      commands: [command('a', 'navigate'), command('b', 'navigate'), command('c', 'new')],
    })

    expect(button(wrapper, 'a').attributes('data-group-start')).toBeUndefined()
    expect(button(wrapper, 'b').attributes('data-group-start')).toBeUndefined()
    expect(button(wrapper, 'c').attributes('data-group-start')).toBeDefined()
  })

  it('show the title and the shortcut as the tooltip', () => {
    const wrapper = mountBar({ commands: [command('a', undefined, 'Ctrl+Shift+E')] })

    expect(button(wrapper, 'a').attributes('title')).toBe('Command a (Ctrl+Shift+E)')
  })

  it('show only the title as the tooltip when there is no shortcut', () => {
    const wrapper = mountBar({ commands: [command('a')] })

    expect(button(wrapper, 'a').attributes('title')).toBe('Command a')
  })

  it('emit run-command with the id when pressed', async () => {
    const wrapper = mountBar({ commands: [command('a'), command('b')] })

    await button(wrapper, 'b').trigger('click')

    expect(wrapper.emitted('run-command')).toStrictEqual([['b']])
  })

  it('are left out of the tab order', () => {
    const wrapper = mountBar({ commands: [command('a')] })

    expect(button(wrapper, 'a').attributes('tabindex')).toBe('-1')
  })

  it('draw no menu button when there are none', () => {
    expect(mountBar().find('[data-title-bar-more]').exists()).toBe(false)
  })

  it('draw no menu button while the room is not measured', () => {
    const wrapper = mountBar({ commands: [command('a'), command('b'), command('c')] })

    expect(wrapper.find('[data-title-bar-more]').exists()).toBe(false)
  })
})

describe('the buttons of plugins', () => {
  const PRESSED_MARKS = ['aria-pressed', 'aria-checked', 'data-pressed', 'data-active']

  it('draw no pressed state before or after a press', async () => {
    const commands = [
      command('org.example.tide:chart.open', 'navigate', 'Ctrl+Shift+E'),
      command('org.example.reef:map.open', 'navigate'),
      command('app:note.create', 'new'),
    ]
    const wrapper = mountBar({ commands })
    const before = wrapper.find('[data-title-bar]').html()

    await button(wrapper, 'org.example.reef:map.open').trigger('click')
    await wrapper.setProps({ commands: [...commands] })

    for (const id of shownIds(wrapper)) {
      for (const mark of PRESSED_MARKS) expect(button(wrapper, id).attributes(mark)).toBeUndefined()
    }
    expect(wrapper.emitted('run-command')).toStrictEqual([['org.example.reef:map.open']])
    expect(wrapper.find('[data-title-bar]').html()).toBe(before)
  })
})

describe('the window drag area', () => {
  it('lets the empty part drag the window', () => {
    expect(
      (mountBar().find('[data-title-bar]').element as HTMLElement).style.getPropertyValue(
        '--wails-draggable',
      ),
    ).toBe('drag')
  })

  it('keeps the buttons out of the drag area', () => {
    const wrapper = mountBar({ commands: [command('a')] })

    expect(
      (button(wrapper, 'a').element as HTMLElement).style.getPropertyValue('--wails-draggable'),
    ).toBe('no-drag')
    expect(
      (control(wrapper, 'close').element as HTMLElement).style.getPropertyValue(
        '--wails-draggable',
      ),
    ).toBe('no-drag')
  })

  it('toggles maximize on a double click on the empty part', async () => {
    const wrapper = mountBar()

    await wrapper.find('[data-title-bar]').trigger('dblclick')

    expect(wrapper.emitted('toggle-maximize-window')).toHaveLength(1)
  })

  it('does not toggle maximize on a double click on an command', async () => {
    const wrapper = mountBar({ commands: [command('a')] })

    await button(wrapper, 'a').trigger('dblclick')

    expect(wrapper.emitted('toggle-maximize-window')).toBeUndefined()
  })

  it('does not toggle maximize on a double click on a window control', async () => {
    const wrapper = mountBar()

    await control(wrapper, 'close').trigger('dblclick')

    expect(wrapper.emitted('toggle-maximize-window')).toBeUndefined()
  })

  it('does not emit toggle-maximize when an command is pressed', async () => {
    const wrapper = mountBar({ commands: [command('a')] })

    await button(wrapper, 'a').trigger('click')

    expect(wrapper.emitted('toggle-maximize-window')).toBeUndefined()
  })
})

describe('the window controls', () => {
  it('draw minimize, maximize and close in the windows style', () => {
    const wrapper = mountBar({ platform: 'windows' })

    expect(
      wrapper.findAll('[data-control]').map((one) => one.attributes('data-control')),
    ).toStrictEqual(['minimize', 'toggle-maximize', 'close'])
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

  it('draw the controls of the windows style after the title', () => {
    const wrapper = mountBar({ platform: 'windows' })
    const title = wrapper.find('[data-title]').element
    const close = control(wrapper, 'close').element

    expect(title.compareDocumentPosition(close) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('name the toggle control by what it does next', () => {
    expect(control(mountBar({ isMaximized: false }), 'toggle-maximize').attributes('title')).toBe(
      'Maximize',
    )
    expect(control(mountBar({ isMaximized: true }), 'toggle-maximize').attributes('title')).toBe(
      'Restore',
    )
  })

  it('mark the maximize button for Snap Layouts on Windows', () => {
    const maximize = control(mountBar(), 'toggle-maximize').element as HTMLElement

    expect(maximize.style.getPropertyValue('--wails-non-client-region').trim()).toBe('maximize')
  })

  it('show the restore icon while the window is maximized', () => {
    const wrapper = mountBar({ isMaximized: true })

    expect(control(wrapper, 'toggle-maximize').attributes('data-maximized')).toBe('true')
    expect(
      mountBar({ isMaximized: false })
        .find('[data-control="toggle-maximize"]')
        .attributes('data-maximized'),
    ).toBe('false')
  })

  it('draw the three circles of the mac style before the commands', () => {
    const wrapper = mountBar({ platform: 'linux', buttons: 'mac', commands: [command('a')] })
    const close = control(wrapper, 'close').element
    const first = button(wrapper, 'a').element

    expect(wrapper.findAll('[data-control]')).toHaveLength(3)
    expect(wrapper.find('[data-title-bar]').attributes('data-controls')).toBe('mac')
    expect(close.compareDocumentPosition(first) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('emit the same events in the mac style', async () => {
    const wrapper = mountBar({ platform: 'linux', buttons: 'mac' })

    await control(wrapper, 'close').trigger('click')
    await control(wrapper, 'minimize').trigger('click')
    await control(wrapper, 'toggle-maximize').trigger('click')

    expect(wrapper.emitted('close-window')).toHaveLength(1)
    expect(wrapper.emitted('minimize-window')).toHaveLength(1)
    expect(wrapper.emitted('toggle-maximize-window')).toHaveLength(1)
  })

  it('draw no control in the native-mac style and keep room for the system buttons', () => {
    const wrapper = mountBar({ platform: 'darwin' })

    expect(wrapper.findAll('[data-control]')).toHaveLength(0)
    expect(wrapper.find('[data-mac-buttons-inset]').exists()).toBe(true)
  })

  it('keep no room for the system buttons in full screen', () => {
    const wrapper = mount(TitleBar, {
      props: { appName: 'numen', platform: 'darwin', isFullscreen: true },
    })

    expect(wrapper.find('[data-mac-buttons-inset]').exists()).toBe(false)
  })

  it('keep no inset in the other styles', () => {
    expect(mountBar({ platform: 'windows' }).find('[data-mac-buttons-inset]').exists()).toBe(false)
    expect(
      mountBar({ platform: 'linux', buttons: 'mac' }).find('[data-mac-buttons-inset]').exists(),
    ).toBe(false)
  })
})

describe('the commands that do not fit', () => {
  const BUTTON_WIDTH = 32
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
      return this.hasAttribute('data-title-bar-command') ? BUTTON_WIDTH : 0
    })
  })

  const resize = async (wrapper: VueWrapper, fits: number): Promise<void> => {
    slotWidth = TITLE_MIN_WIDTH + fits * BUTTON_WIDTH
    const entries = [{ contentRect: { width: slotWidth, height: 32 } } as ResizeObserverEntry]
    for (const tell of callbacks) tell(entries, {} as ResizeObserver)
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()
  }

  const mountOverflowing = async (fits: number): Promise<VueWrapper> => {
    slotWidth = TITLE_MIN_WIDTH + fits * BUTTON_WIDTH
    const wrapper = mountBar({
      commands: [
        command('a', 'navigate'),
        command('b', 'navigate'),
        command('c', 'new'),
        command('d', 'new'),
        command('e', 'tools'),
      ],
    })
    await resize(wrapper, fits)
    return wrapper
  }

  const menuItems = (): string[] =>
    [...document.body.querySelectorAll('[role="menuitem"]')].map((one) => one.textContent!.trim())

  it('show every command and no menu button when they all fit', async () => {
    const wrapper = await mountOverflowing(5)

    expect(shownIds(wrapper)).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
    expect(wrapper.find('[data-title-bar-more]').exists()).toBe(false)
  })

  it('show the first commands and a menu button when some do not fit', async () => {
    const wrapper = await mountOverflowing(3)

    expect(shownIds(wrapper)).toStrictEqual(['a', 'b'])
    expect(wrapper.find('[data-title-bar-more]').exists()).toBe(true)
  })

  it('list the commands that do not fit in the menu, in order, with their titles', async () => {
    const wrapper = await mountOverflowing(3)

    await wrapper.find('[data-title-bar-more]').trigger('click')

    expect(menuItems()).toStrictEqual(['Command c', 'Command d', 'Command e'])
  })

  it('separate the groups of the menu', async () => {
    const wrapper = await mountOverflowing(3)

    await wrapper.find('[data-title-bar-more]').trigger('click')

    expect(document.body.querySelectorAll('[role="separator"], .menu__rule').length).toBe(1)
  })

  it('show only the menu button when no command fits', async () => {
    const wrapper = await mountOverflowing(1)

    expect(shownIds(wrapper)).toStrictEqual([])
    await wrapper.find('[data-title-bar-more]').trigger('click')
    expect(menuItems()).toHaveLength(5)
  })

  it('emit run-command with the id of the entry picked in the menu', async () => {
    const wrapper = await mountOverflowing(3)

    await wrapper.find('[data-title-bar-more]').trigger('click')
    const entries = [...document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')]
    entries[1]!.click()
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('run-command')).toStrictEqual([['d']])
  })

  it('close the menu after an entry is picked', async () => {
    const wrapper = await mountOverflowing(3)

    await wrapper.find('[data-title-bar-more]').trigger('click')
    document.body.querySelector<HTMLElement>('[role="menuitem"]')!.click()
    await wrapper.vm.$nextTick()

    expect(document.body.querySelector('[role="menu"]')).toBeNull()
  })

  it('move commands back to the bar when the window grows', async () => {
    const wrapper = await mountOverflowing(3)

    await resize(wrapper, 5)

    expect(shownIds(wrapper)).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
    expect(wrapper.find('[data-title-bar-more]').exists()).toBe(false)
  })

  it('keep every command in the bar while the room is not measured', async () => {
    slotWidth = 0
    const wrapper = mountBar({
      commands: ['a', 'b', 'c', 'd', 'e'].map((id) => command(id)),
    })
    await wrapper.vm.$nextTick()

    expect(shownIds(wrapper)).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
    expect(wrapper.find('[data-title-bar-more]').exists()).toBe(false)
  })

  it('round the room down to a whole number of buttons', async () => {
    const wrapper = await mountOverflowing(3.5)

    expect(shownIds(wrapper)).toStrictEqual(['a', 'b'])
  })

  it('close the menu when the window grows enough for every command', async () => {
    const wrapper = await mountOverflowing(3)
    await wrapper.find('[data-title-bar-more]').trigger('click')

    await resize(wrapper, 5)

    expect(document.body.querySelector('[role="menu"]')).toBeNull()
  })

  it('move commands to the menu when the window shrinks', async () => {
    const wrapper = await mountOverflowing(5)

    await resize(wrapper, 2)

    expect(shownIds(wrapper)).toStrictEqual(['a'])
    expect(wrapper.find('[data-title-bar-more]').exists()).toBe(true)
  })
})
