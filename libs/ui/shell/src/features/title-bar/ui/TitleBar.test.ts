import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import TitleBar from './TitleBar.vue'
import { NATIVE_MAC_INSET, TITLE_MIN_WIDTH } from '../lib/layout'
import type { ControlsStyle, TitleBarAction } from '../lib/types'

const action = (id: string, group?: string, shortcut?: string): TitleBarAction => ({
  id,
  title: `Action ${id}`,
  icon: { render: () => null },
  ...(group === undefined ? {} : { group }),
  ...(shortcut === undefined ? {} : { shortcut }),
})

interface Options {
  vaultName?: string
  controls?: ControlsStyle
  isMaximised?: boolean
  actions?: readonly TitleBarAction[]
}

const mountBar = (options: Options = {}): VueWrapper =>
  mount(TitleBar, {
    attachTo: document.body,
    props: {
      appName: 'numen',
      vaultName: 'Orchard notes',
      controls: 'windows' as ControlsStyle,
      isMaximised: false,
      actions: [],
      ...options,
    },
  })

const button = (wrapper: VueWrapper, id: string) => wrapper.find(`[data-title-bar-action="${id}"]`)
const shownIds = (wrapper: VueWrapper): string[] =>
  wrapper.findAll('[data-title-bar-action]').map((one) => one.attributes('data-title-bar-action')!)
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

describe('the actions', () => {
  it('draw in the order given', () => {
    const wrapper = mountBar({ actions: [action('a'), action('b'), action('c')] })

    expect(shownIds(wrapper)).toStrictEqual(['a', 'b', 'c'])
  })

  it('mark the first action of each group after the first group', () => {
    const wrapper = mountBar({
      actions: [action('a', 'navigate'), action('b', 'navigate'), action('c', 'new')],
    })

    expect(button(wrapper, 'a').attributes('data-group-start')).toBeUndefined()
    expect(button(wrapper, 'b').attributes('data-group-start')).toBeUndefined()
    expect(button(wrapper, 'c').attributes('data-group-start')).toBeDefined()
  })

  it('show the title and the shortcut as the tooltip', () => {
    const wrapper = mountBar({ actions: [action('a', undefined, 'Ctrl+Shift+E')] })

    expect(button(wrapper, 'a').attributes('title')).toBe('Action a (Ctrl+Shift+E)')
  })

  it('show only the title as the tooltip when there is no shortcut', () => {
    const wrapper = mountBar({ actions: [action('a')] })

    expect(button(wrapper, 'a').attributes('title')).toBe('Action a')
  })

  it('emit run with the id when pressed', async () => {
    const wrapper = mountBar({ actions: [action('a'), action('b')] })

    await button(wrapper, 'b').trigger('click')

    expect(wrapper.emitted('run')).toStrictEqual([['b']])
  })

  it('do not take the keyboard focus', () => {
    const wrapper = mountBar({ actions: [action('a')] })

    expect(button(wrapper, 'a').attributes('tabindex')).toBe('-1')
  })

  it('draw no menu button when there are none', () => {
    expect(mountBar().find('[data-title-bar-more]').exists()).toBe(false)
  })

  it('draw no menu button while the room is not measured', () => {
    const wrapper = mountBar({ actions: [action('a'), action('b'), action('c')] })

    expect(wrapper.find('[data-title-bar-more]').exists()).toBe(false)
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
    const wrapper = mountBar({ actions: [action('a')] })

    expect(
      (button(wrapper, 'a').element as HTMLElement).style.getPropertyValue('--wails-draggable'),
    ).toBe('no-drag')
    expect(
      (control(wrapper, 'close').element as HTMLElement).style.getPropertyValue(
        '--wails-draggable',
      ),
    ).toBe('no-drag')
  })

  it('toggles maximise on a double click on the empty part', async () => {
    const wrapper = mountBar()

    await wrapper.find('[data-title-bar]').trigger('dblclick')

    expect(wrapper.emitted('toggle-maximise')).toHaveLength(1)
  })

  it('does not toggle maximise on a double click on an action', async () => {
    const wrapper = mountBar({ actions: [action('a')] })

    await button(wrapper, 'a').trigger('dblclick')

    expect(wrapper.emitted('toggle-maximise')).toBeUndefined()
  })

  it('does not toggle maximise on a double click on a window control', async () => {
    const wrapper = mountBar()

    await control(wrapper, 'close').trigger('dblclick')

    expect(wrapper.emitted('toggle-maximise')).toBeUndefined()
  })

  it('does not emit toggle-maximise when an action is pressed', async () => {
    const wrapper = mountBar({ actions: [action('a')] })

    await button(wrapper, 'a').trigger('click')

    expect(wrapper.emitted('toggle-maximise')).toBeUndefined()
  })
})

describe('the window controls', () => {
  it('draw minimise, maximise and close in the windows style', () => {
    const wrapper = mountBar({ controls: 'windows' })

    expect(
      wrapper.findAll('[data-control]').map((one) => one.attributes('data-control')),
    ).toStrictEqual(['minimise', 'toggle-maximise', 'close'])
  })

  it('emit the event of the control that is pressed', async () => {
    const wrapper = mountBar({ controls: 'windows' })

    await control(wrapper, 'minimise').trigger('click')
    await control(wrapper, 'toggle-maximise').trigger('click')
    await control(wrapper, 'close').trigger('click')

    expect(wrapper.emitted('minimise')).toHaveLength(1)
    expect(wrapper.emitted('toggle-maximise')).toHaveLength(1)
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('draw the controls of the windows style after the title', () => {
    const wrapper = mountBar({ controls: 'windows' })
    const title = wrapper.find('[data-title]').element
    const close = control(wrapper, 'close').element

    expect(title.compareDocumentPosition(close) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('show the restore icon instead of maximise while the window is maximised', () => {
    const wrapper = mountBar({ isMaximised: true })

    expect(control(wrapper, 'toggle-maximise').attributes('data-maximised')).toBe('true')
    expect(
      mountBar({ isMaximised: false })
        .find('[data-control="toggle-maximise"]')
        .attributes('data-maximised'),
    ).toBe('false')
  })

  it('draw the three circles of the mac style before the actions', () => {
    const wrapper = mountBar({ controls: 'mac', actions: [action('a')] })
    const close = control(wrapper, 'close').element
    const first = button(wrapper, 'a').element

    expect(wrapper.findAll('[data-control]')).toHaveLength(3)
    expect(wrapper.find('[data-title-bar]').attributes('data-controls')).toBe('mac')
    expect(close.compareDocumentPosition(first) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('emit the same events in the mac style', async () => {
    const wrapper = mountBar({ controls: 'mac' })

    await control(wrapper, 'close').trigger('click')
    await control(wrapper, 'minimise').trigger('click')
    await control(wrapper, 'toggle-maximise').trigger('click')

    expect(wrapper.emitted('close')).toHaveLength(1)
    expect(wrapper.emitted('minimise')).toHaveLength(1)
    expect(wrapper.emitted('toggle-maximise')).toHaveLength(1)
  })

  it('draw no control in the native-mac style and keep room for the system buttons', () => {
    const wrapper = mountBar({ controls: 'native-mac' })

    expect(wrapper.findAll('[data-control]')).toHaveLength(0)
    expect(wrapper.find('[data-inset]').attributes('style')).toContain(`${NATIVE_MAC_INSET}px`)
    expect(NATIVE_MAC_INSET).toBe(72)
  })

  it('keep no inset in the other styles', () => {
    expect(mountBar({ controls: 'windows' }).find('[data-inset]').exists()).toBe(false)
    expect(mountBar({ controls: 'mac' }).find('[data-inset]').exists()).toBe(false)
  })
})

describe('the actions that do not fit', () => {
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
      return this.hasAttribute('data-title-bar-action') ? BUTTON_WIDTH : 0
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
      actions: [
        action('a', 'navigate'),
        action('b', 'navigate'),
        action('c', 'new'),
        action('d', 'new'),
        action('e', 'tools'),
      ],
    })
    await resize(wrapper, fits)
    return wrapper
  }

  const menuItems = (): string[] =>
    [...document.body.querySelectorAll('[role="menuitem"]')].map((one) => one.textContent!.trim())

  it('show every action and no menu button when they all fit', async () => {
    const wrapper = await mountOverflowing(5)

    expect(shownIds(wrapper)).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
    expect(wrapper.find('[data-title-bar-more]').exists()).toBe(false)
  })

  it('show the first actions and a menu button when some do not fit', async () => {
    const wrapper = await mountOverflowing(3)

    expect(shownIds(wrapper)).toStrictEqual(['a', 'b'])
    expect(wrapper.find('[data-title-bar-more]').exists()).toBe(true)
  })

  it('list the actions that do not fit in the menu, in order, with their titles', async () => {
    const wrapper = await mountOverflowing(3)

    await wrapper.find('[data-title-bar-more]').trigger('click')

    expect(menuItems()).toStrictEqual(['Action c', 'Action d', 'Action e'])
  })

  it('separate the groups of the menu', async () => {
    const wrapper = await mountOverflowing(3)

    await wrapper.find('[data-title-bar-more]').trigger('click')

    expect(document.body.querySelectorAll('[role="separator"], .menu__rule').length).toBe(1)
  })

  it('show only the menu button when no action fits', async () => {
    const wrapper = await mountOverflowing(1)

    expect(shownIds(wrapper)).toStrictEqual([])
    await wrapper.find('[data-title-bar-more]').trigger('click')
    expect(menuItems()).toHaveLength(5)
  })

  it('emit run with the id of the entry picked in the menu', async () => {
    const wrapper = await mountOverflowing(3)

    await wrapper.find('[data-title-bar-more]').trigger('click')
    const entries = [...document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')]
    entries[1]!.click()
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('run')).toStrictEqual([['d']])
  })

  it('close the menu after an entry is picked', async () => {
    const wrapper = await mountOverflowing(3)

    await wrapper.find('[data-title-bar-more]').trigger('click')
    document.body.querySelector<HTMLElement>('[role="menuitem"]')!.click()
    await wrapper.vm.$nextTick()

    expect(document.body.querySelector('[role="menu"]')).toBeNull()
  })

  it('move actions back to the bar when the window grows', async () => {
    const wrapper = await mountOverflowing(3)

    await resize(wrapper, 5)

    expect(shownIds(wrapper)).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
    expect(wrapper.find('[data-title-bar-more]').exists()).toBe(false)
  })

  it('move actions to the menu when the window shrinks', async () => {
    const wrapper = await mountOverflowing(5)

    await resize(wrapper, 2)

    expect(shownIds(wrapper)).toStrictEqual(['a'])
    expect(wrapper.find('[data-title-bar-more]').exists()).toBe(true)
  })
})
