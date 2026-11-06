import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick, type Component } from 'vue'
import { createMockPluginContext, type MockPluginContext } from '@numen/plugin-sdk/testing'
import {
  SLOT,
  TITLE_BAR_SLOT,
  type AppInfo,
  type PluginContext,
  type SlotViewRegistration,
} from '@numen/plugin-sdk'
import { activate, TITLE_BAR_VIEW } from './index'
import { command, separator, submenu, view } from './testing/entries'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  id: string
  targets: readonly string[]
  contributes: { views: readonly { id: string; slot: string }[] }
  settings: readonly { key: string; default: string; options: readonly { value: string }[] }[]
}

const ORCHARD = { id: 'orchard', name: 'Orchard notes' }
const GREENHOUSE = { id: 'greenhouse', name: 'Greenhouse log' }

const mounted: VueWrapper[] = []

interface Options {
  readonly platform?: AppInfo['platform']
  readonly settings?: Record<string, unknown>
}

const activateBar = ({ platform = 'windows', settings = {} }: Options = {}) => {
  const context = createMockPluginContext({
    pluginId: manifest.id,
    initialSettings: settings,
    app: { name: 'numen', version: '2026.11.1', build: 7, platform },
  })
  activate(context)
  const [registration] = context.getRegisteredSlotViews()
  return { context, view: registration!.component as Component }
}

const mountBar = (options: Options = {}) => {
  const { context, view: component } = activateBar(options)
  const bar = mount(component, { attachTo: document.body })
  mounted.push(bar)
  return { context, bar }
}

const getButton = (bar: VueWrapper, id: string) => bar.find(`[data-title-bar-command="${id}"]`)
const listButtonIds = (bar: VueWrapper, slot: string) =>
  bar
    .findAll(`[data-title-bar-slot="${slot}"] [data-title-bar-command]`)
    .map((one) => one.attributes('data-title-bar-command'))
const getControl = (bar: VueWrapper, name: string) => bar.find(`[data-control="${name}"]`)
const getDrag = (element: Element) =>
  (element as HTMLElement).style.getPropertyValue('--wails-draggable')
const getMenuRows = () =>
  [...document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')].map((one) =>
    one.textContent?.trim(),
  )

const setVaults = async (context: MockPluginContext, current = ORCHARD) => {
  context.vaults.setState({ vaults: [ORCHARD, GREENHOUSE], current })
  await nextTick()
}

afterEach(() => {
  mounted.splice(0).forEach((one) => one.unmount())
  document.body.innerHTML = ''
})

describe('the plugin', () => {
  it('registers the bar under the id the manifest declares in the title bar slot', () => {
    const { view: component } = activateBar()

    expect(component).toBeDefined()
    expect(manifest.contributes.views).toStrictEqual([{ id: TITLE_BAR_VIEW, slot: TITLE_BAR_SLOT }])
  })

  it('runs in the editor and in the flashcards window', () => {
    expect(manifest.id).toBe('md.numen.views.titleBar')
    expect(manifest.targets).toStrictEqual(['desktop.editor', 'desktop.flashcards'])
  })

  it('declares the setting of the window buttons, at the right by default', () => {
    const setting = manifest.settings.find((one) => one.key === 'buttons')

    expect(setting?.default).toBe('windows')
    expect(setting?.options.map((one) => one.value)).toStrictEqual(['windows', 'mac'])
  })
})

describe('the commands of the slots', () => {
  it('draws the commands of titleBar.commands as buttons in the order of the slot', async () => {
    const { context, bar } = mountBar()

    context.slots.setEntries(SLOT.titleBarCommands, [command('chart.open'), command('map.open')])
    await nextTick()

    expect(listButtonIds(bar, SLOT.titleBarCommands)).toStrictEqual([
      'org.example.tide:chart.open',
      'org.example.tide:map.open',
    ])
  })

  it('draws the icon of the command and names the button by its title', async () => {
    const { context, bar } = mountBar()

    context.slots.setEntries(SLOT.titleBarCommands, [command('chart.open')])
    await nextTick()

    const button = getButton(bar, 'org.example.tide:chart.open')
    expect(button.find('[data-icon="mark"]').exists()).toBe(true)
    expect(button.attributes('aria-label')).toBe('Open chart.open')
  })

  it('writes the title as the button of a command without an icon', async () => {
    const { context, bar } = mountBar()

    context.slots.setEntries(SLOT.titleBarCommands, [command('chart.open', { icon: undefined })])
    await nextTick()

    expect(getButton(bar, 'org.example.tide:chart.open').text()).toBe('Open chart.open')
  })

  it('shows the title and the shortcut in the tooltip, written for the platform', async () => {
    const { context, bar } = mountBar({ platform: 'darwin' })

    context.slots.setEntries(SLOT.titleBarCommands, [
      command('chart.open', { shortcut: 'mod+shift+e' }),
      command('map.open'),
    ])
    await nextTick()

    expect(getButton(bar, 'org.example.tide:chart.open').attributes('title')).toBe(
      'Open chart.open (⌘+Shift+E)',
    )
    expect(getButton(bar, 'org.example.tide:map.open').attributes('title')).toBe('Open map.open')
  })

  it('runs the command of a button through the slots, with no target', async () => {
    const { context, bar } = mountBar()
    context.slots.setEntries(SLOT.titleBarCommands, [command('chart.open')])
    await nextTick()

    await getButton(bar, 'org.example.tide:chart.open').trigger('click')

    expect(context.slots.getRuns()).toStrictEqual([
      { command: 'org.example.tide:chart.open', target: undefined },
    ])
  })

  it('follows a change of the entries of the slot', async () => {
    const { context, bar } = mountBar()
    context.slots.setEntries(SLOT.titleBarCommands, [command('chart.open'), command('map.open')])
    await nextTick()

    context.slots.setEntries(SLOT.titleBarCommands, [command('map.open')])
    await nextTick()

    expect(listButtonIds(bar, SLOT.titleBarCommands)).toStrictEqual(['org.example.tide:map.open'])
  })

  it('draws a separator as a gap between two groups', async () => {
    const { context, bar } = mountBar()

    context.slots.setEntries(SLOT.titleBarCommands, [
      command('chart.open'),
      separator(1),
      command('map.open'),
    ])
    await nextTick()

    const row = bar.find(`[data-title-bar-slot="${SLOT.titleBarCommands}"]`).element
    expect([...row.children].map((one) => one.hasAttribute('data-title-bar-gap'))).toStrictEqual([
      false,
      true,
      false,
    ])
  })

  it('draws a command placed in titleBar.status as a button there', async () => {
    const { context, bar } = mountBar()

    context.slots.setEntries(SLOT.titleBarStatus, [command('log.open')])
    await nextTick()

    expect(listButtonIds(bar, SLOT.titleBarStatus)).toStrictEqual(['org.example.tide:log.open'])
  })
})

describe('the views of the slots', () => {
  it('draws a view of titleBar.status in its place among the entries', async () => {
    const { context, bar } = mountBar()

    context.slots.setEntries(SLOT.titleBarStatus, [
      view('tasks'),
      command('log.open'),
      view('stats'),
    ])
    await nextTick()

    const row = bar.find(`[data-title-bar-slot="${SLOT.titleBarStatus}"]`)
    expect(row.findAll('[data-label-view]').map((one) => one.text())).toStrictEqual([
      'tasks',
      'stats',
    ])
    expect(row.element.children[1]?.hasAttribute('data-title-bar-command')).toBe(true)
  })

  it('draws a view placed in titleBar.commands', async () => {
    const { context, bar } = mountBar()

    context.slots.setEntries(SLOT.titleBarCommands, [view('due')])
    await nextTick()

    expect(
      bar.find(`[data-title-bar-slot="${SLOT.titleBarCommands}"] [data-label-view="due"]`).exists(),
    ).toBe(true)
  })
})

describe('a submenu in a slot', () => {
  const placeSubmenu = async (context: MockPluginContext) => {
    context.slots.setEntries(SLOT.titleBarCommands, [
      submenu('tide.new', [command('chart.create'), separator(1), command('log.create')]),
    ])
    await nextTick()
  }

  it('is a button that says it opens a menu', async () => {
    const { context, bar } = mountBar()
    await placeSubmenu(context)

    const button = bar.find('[data-title-bar-submenu="tide.new"]')
    expect(button.attributes('aria-haspopup')).toBe('menu')
    expect(button.attributes('aria-expanded')).toBe('false')
    expect(button.attributes('aria-label')).toBe('New')
  })

  it('opens the menu of its entries and runs the one picked', async () => {
    const { context, bar } = mountBar()
    await placeSubmenu(context)

    await bar.find('[data-title-bar-submenu="tide.new"]').trigger('click')
    expect(getMenuRows()).toStrictEqual(['Open chart.create', 'Open log.create'])
    expect(bar.find('[data-title-bar-submenu="tide.new"]').attributes('aria-expanded')).toBe('true')
    document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')[1]!.click()
    await nextTick()

    expect(context.slots.getRuns()).toStrictEqual([
      { command: 'org.example.tide:log.create', target: undefined },
    ])
    expect(document.body.querySelector('[role="menu"]')).toBeNull()
  })
})

describe('dragging the window', () => {
  it('drags the window by the free space of the bar', () => {
    const { bar } = mountBar()

    expect(getDrag(bar.find('[data-title-bar]').element)).toBe('drag')
  })

  it('marks every entry of both slots as no-drag', async () => {
    const { context, bar } = mountBar()
    context.slots.setEntries(SLOT.titleBarCommands, [
      command('chart.open'),
      separator(1),
      submenu('tide.new', [command('chart.create')]),
      view('due'),
    ])
    context.slots.setEntries(SLOT.titleBarStatus, [view('tasks'), command('log.open')])
    await nextTick()

    const entries = bar.findAll('[data-title-bar-entry]')
    expect(entries).toHaveLength(6)
    for (const entry of entries) expect(getDrag(entry.element)).toBe('no-drag')
  })
})

describe('the title', () => {
  it('names the application from the context and the vault the window shows', async () => {
    const { context, bar } = mountBar()

    await setVaults(context)

    expect(bar.find('[data-title]').text()).toBe('numen — Orchard notes')
  })

  it('names only the application while the window shows no vault', async () => {
    const { context, bar } = mountBar()

    context.vaults.setState({ vaults: [ORCHARD, GREENHOUSE], current: null })
    await nextTick()

    expect(bar.find('[data-title]').text()).toBe('numen')
    expect(bar.find('[data-title]').element.tagName).toBe('SPAN')
  })

  it('switches to the vault chosen from the list the title opens', async () => {
    const { context, bar } = mountBar()
    await setVaults(context)

    await bar.find('[data-title]').trigger('click')
    await nextTick()
    document.body.querySelectorAll<HTMLElement>('[role="option"]')[1]!.click()
    await nextTick()

    expect(context.vaults.getSwitches()).toStrictEqual(['greenhouse'])
  })

  it('switches to nothing when the vault the window shows is chosen', async () => {
    const { context, bar } = mountBar()
    await setVaults(context)

    await bar.find('[data-title]').trigger('click')
    await nextTick()
    document.body.querySelectorAll<HTMLElement>('[role="option"]')[0]!.click()
    await nextTick()

    expect(context.vaults.getSwitches()).toStrictEqual([])
  })

  it('follows a switch of the vault', async () => {
    const { context, bar } = mountBar()
    await setVaults(context)

    await setVaults(context, GREENHOUSE)

    expect(bar.find('[data-title]').text()).toBe('numen — Greenhouse log')
  })
})

describe('the window buttons', () => {
  it('minimize, maximize and close the window through the context', async () => {
    const { context, bar } = mountBar()

    await getControl(bar, 'minimize').trigger('click')
    await getControl(bar, 'toggle-maximize').trigger('click')
    await getControl(bar, 'close').trigger('click')

    expect(context.window.getActions()).toStrictEqual(['minimize', 'toggleMaximize', 'close'])
  })

  it('toggle maximize on a double click on the free space', async () => {
    const { context, bar } = mountBar()

    await bar.find('[data-title-bar]').trigger('dblclick')

    expect(context.window.getActions()).toStrictEqual(['toggleMaximize'])
  })

  it('draw the state of the window and follow its change', async () => {
    const { context, bar } = mountBar()
    expect(getControl(bar, 'toggle-maximize').attributes('title')).toBe('Maximize')

    context.window.setState({ isMaximized: true, isFullscreen: false })
    await nextTick()

    expect(getControl(bar, 'toggle-maximize').attributes('title')).toBe('Restore')
    expect(getControl(bar, 'toggle-maximize').attributes('data-maximized')).toBe('true')
  })

  it('keep room for the macOS buttons, and none in full screen', async () => {
    const { context, bar } = mountBar({ platform: 'darwin' })
    expect(bar.find('[data-mac-buttons-inset]').exists()).toBe(true)

    context.window.setState({ isMaximized: false, isFullscreen: true })
    await nextTick()

    expect(bar.find('[data-mac-buttons-inset]').exists()).toBe(false)
  })
})

describe('the window buttons on Linux', () => {
  it('stand at the right while the setting says nothing', () => {
    const { bar } = mountBar({ platform: 'linux' })

    expect(bar.find('[data-title-bar]').attributes('data-controls')).toBe('windows')
  })

  it('stand at the left when the setting says mac, and move back when it changes', async () => {
    const { context, bar } = mountBar({ platform: 'linux', settings: { buttons: 'mac' } })
    expect(bar.find('[data-title-bar]').attributes('data-controls')).toBe('mac')

    context.setSetting('buttons', 'windows')
    await nextTick()

    expect(bar.find('[data-title-bar]').attributes('data-controls')).toBe('windows')
  })

  it('stay with the system on macOS and at the right on Windows, whatever the setting', () => {
    const mac = mountBar({ platform: 'darwin', settings: { buttons: 'mac' } }).bar
    const windows = mountBar({ platform: 'windows', settings: { buttons: 'mac' } }).bar

    expect(mac.find('[data-title-bar]').attributes('data-controls')).toBe('native-mac')
    expect(windows.find('[data-title-bar]').attributes('data-controls')).toBe('windows')
  })
})

describe('the bar in a window that offers no slots, vaults or window', () => {
  it('draws the title of the application and no entry', () => {
    const registerSlotView = vi.fn(() => ({ dispose: () => undefined }))
    const context: PluginContext = { ...createMockPluginContext(), registerSlotView }
    for (const key of ['slots', 'vaults', 'window']) Reflect.deleteProperty(context, key)
    activate(context)
    const [[registration]] = registerSlotView.mock.calls as unknown as [[SlotViewRegistration]]
    const bar = mount(registration.component as Component)
    mounted.push(bar)

    expect(bar.find('[data-title]').text()).toBe('numen')
    expect(bar.findAll('[data-title-bar-entry]')).toHaveLength(0)
  })
})

describe('removing the bar', () => {
  it('stops watching the slots, the vaults and the window', async () => {
    const { context, view: component } = activateBar()
    const disposed: string[] = []
    const track = <T extends { dispose: () => void }>(name: string, one: T): T => ({
      ...one,
      dispose: () => {
        disposed.push(name)
        one.dispose()
      },
    })
    const { slots, vaults, window } = context
    const watchEntries = slots.watchEntries
    const watchVaults = vaults.watch
    const watchWindow = window.watch
    slots.watchEntries = (slot, handler) => track(slot, watchEntries(slot, handler))
    vaults.watch = (handler) => track('vaults', watchVaults(handler))
    window.watch = (handler) => track('window', watchWindow(handler))
    const bar = mount(component)

    bar.unmount()
    await nextTick()

    expect(disposed.sort()).toStrictEqual(
      [SLOT.titleBarCommands, SLOT.titleBarStatus, 'vaults', 'window'].sort(),
    )
  })
})
