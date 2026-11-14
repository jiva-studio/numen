import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { nextTick, type Component } from 'vue'
import {
  createMockNoticeHost,
  createMockHostServices,
  createMockPluginContext,
  type MockHostServices,
} from '@numen/plugin-sdk/testing'
import {
  SLOT,
  type AppInfo,
  type JsonValue,
  type PluginContext,
  type SlotViewRegistration,
} from '@numen/plugin-sdk'
import { activate, TITLE_BAR_VIEW } from './index'
import { command, commandWithoutIcon, separator, submenu, view } from './testing/entries'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  id: string
  targets: readonly string[]
  contributes: { slots: readonly { slot: string; view: string }[] }
  settings: readonly {
    key: string
    default: string
    options: readonly { value: string; when?: string }[]
  }[]
}

const ORCHARD = { id: 'orchard', name: 'Orchard notes' }
const GREENHOUSE = { id: 'greenhouse', name: 'Greenhouse log' }

const mounted: VueWrapper[] = []

interface Options {
  readonly platform?: AppInfo['platform']
  readonly settings?: Record<string, JsonValue>
}

const activateBar = ({ platform = 'windows', settings = {} }: Options = {}) => {
  const context = createMockPluginContext({
    pluginId: manifest.id,
    initialSettings: settings,
    app: { name: 'numen', version: '2026.11.1', build: 7, platform },
  })
  const host = createMockHostServices()
  activate(context, host)
  const [registration] = context.getRegisteredSlotViews()
  return { context, host, view: registration!.component as Component }
}

const mountBar = (options: Options = {}) => {
  const { context, host, view: component } = activateBar(options)
  const bar = mount(component, { attachTo: document.body })
  mounted.push(bar)
  return { context, host, bar }
}

const getButton = (bar: VueWrapper, id: string) => bar.find(`[data-title-bar-command="${id}"]`)
const listButtonIds = (bar: VueWrapper, slot: string) =>
  bar
    .findAll(`[data-title-bar-slot="${slot}"] [data-title-bar-command]`)
    .map((one) => one.attributes('data-title-bar-command'))
const getControl = (bar: VueWrapper, name: string) => bar.find(`[data-control="${name}"]`)
const getMenuRows = () =>
  [...document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')].map((one) =>
    one.textContent?.trim(),
  )

const setVaults = async (host: MockHostServices, current = ORCHARD) => {
  host.vault.vaults.setState({ vaults: [ORCHARD, GREENHOUSE], current })
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
    expect(manifest.contributes.slots).toStrictEqual([
      { slot: SLOT.titleBar, view: TITLE_BAR_VIEW },
    ])
  })

  it('runs in the editor and in the flashcards window', () => {
    expect(manifest.id).toBe('md.numen.views.titleBar')
    expect(manifest.targets).toStrictEqual(['desktop.editor', 'desktop.flashcards'])
  })

  it('declares the setting of the window buttons, at the right by default', () => {
    const setting = manifest.settings.find((one) => one.key === 'buttons')

    expect(setting?.default).toBe('windows')
    expect(setting?.options.map((one) => one.value)).toStrictEqual(['mac', 'windows', 'none'])
  })

  it('offers Right and None except on macOS, and Left everywhere', () => {
    const options = manifest.settings.find((one) => one.key === 'buttons')?.options ?? []

    expect(options.map((one) => [one.value, one.when ?? ''])).toStrictEqual([
      ['mac', ''],
      ['windows', "app.platform != 'darwin'"],
      ['none', "app.platform != 'darwin'"],
    ])
  })
})

describe('the commands of the slots', () => {
  it('draws the commands of titleBar.commands as buttons in the order of the slot', async () => {
    const { host, bar } = mountBar()

    host.ui.slots.setEntries(SLOT.titleBarCommands, [command('chart.open'), command('map.open')])
    await nextTick()

    expect(listButtonIds(bar, SLOT.titleBarCommands)).toStrictEqual([
      'org.example.tide:chart.open',
      'org.example.tide:map.open',
    ])
  })

  it('draws the icon of the command and names the button by its title', async () => {
    const { host, bar } = mountBar()

    host.ui.slots.setEntries(SLOT.titleBarCommands, [command('chart.open')])
    await nextTick()

    const button = getButton(bar, 'org.example.tide:chart.open')
    expect(button.find('[data-icon="mark"]').exists()).toBe(true)
    expect(button.attributes('aria-label')).toBe('Open chart.open')
  })

  it('writes the title as the button of a command without an icon', async () => {
    const { host, bar } = mountBar()

    host.ui.slots.setEntries(SLOT.titleBarCommands, [commandWithoutIcon('chart.open')])
    await nextTick()

    expect(getButton(bar, 'org.example.tide:chart.open').text()).toBe('Open chart.open')
  })

  it('shows the title and the shortcut in the tooltip, written for the platform', async () => {
    const { host, bar } = mountBar({ platform: 'darwin' })

    host.ui.slots.setEntries(SLOT.titleBarCommands, [
      command('chart.open', { shortcut: 'mod+shift+e' }),
      command('map.open'),
    ])
    await nextTick()

    expect(getButton(bar, 'org.example.tide:chart.open').attributes('title')).toBe(
      'Open chart.open (⌘+Shift+E)',
    )
    expect(getButton(bar, 'org.example.tide:map.open').attributes('title')).toBe('Open map.open')
  })

  it('runs the command of a button through the commands of the context, with no target', async () => {
    const { host, bar } = mountBar()
    host.ui.slots.setEntries(SLOT.titleBarCommands, [command('chart.open')])
    await nextTick()

    await getButton(bar, 'org.example.tide:chart.open').trigger('click')

    expect(host.ui.commands.getRuns()).toStrictEqual([
      { command: 'org.example.tide:chart.open', target: undefined, itemId: undefined },
    ])
  })

  it('shows the error of a run that fails as one notice', async () => {
    const notices = createMockNoticeHost()
    const context = createMockPluginContext({ pluginId: manifest.id })
    const host = createMockHostServices({ notices })
    activate(context, host)
    const bar = mount(context.getRegisteredSlotViews()[0]!.component as Component, {
      attachTo: document.body,
    })
    mounted.push(bar)
    host.ui.slots.setEntries(SLOT.titleBarCommands, [command('chart.open')])
    await nextTick()
    vi.spyOn(host.ui.commands, 'run').mockRejectedValueOnce(new Error('the chart is gone'))

    await getButton(bar, 'org.example.tide:chart.open').trigger('click')
    await flushPromises()

    expect(notices.getShown()).toStrictEqual([{ text: 'the chart is gone', severity: 'error' }])
  })

  it('follows a change of the entries of the slot', async () => {
    const { host, bar } = mountBar()
    host.ui.slots.setEntries(SLOT.titleBarCommands, [command('chart.open'), command('map.open')])
    await nextTick()

    host.ui.slots.setEntries(SLOT.titleBarCommands, [command('map.open')])
    await nextTick()

    expect(listButtonIds(bar, SLOT.titleBarCommands)).toStrictEqual(['org.example.tide:map.open'])
  })

  it('draws a separator as a gap between two groups', async () => {
    const { host, bar } = mountBar()

    host.ui.slots.setEntries(SLOT.titleBarCommands, [
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
    const { host, bar } = mountBar()

    host.ui.slots.setEntries(SLOT.titleBarStatus, [command('log.open')])
    await nextTick()

    expect(listButtonIds(bar, SLOT.titleBarStatus)).toStrictEqual(['org.example.tide:log.open'])
  })
})

describe('the views of the slots', () => {
  it('draws a view of titleBar.status in its place among the entries', async () => {
    const { host, bar } = mountBar()

    host.ui.slots.setEntries(SLOT.titleBarStatus, [
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
    const { host, bar } = mountBar()

    host.ui.slots.setEntries(SLOT.titleBarCommands, [view('due')])
    await nextTick()

    expect(
      bar.find(`[data-title-bar-slot="${SLOT.titleBarCommands}"] [data-label-view="due"]`).exists(),
    ).toBe(true)
  })
})

describe('a submenu in a slot', () => {
  const placeSubmenu = async (host: MockHostServices) => {
    host.ui.slots.setEntries(SLOT.titleBarCommands, [
      submenu('tide.new', [command('chart.create'), separator(1), command('log.create')]),
    ])
    await nextTick()
  }

  it('is a button that says it opens a menu', async () => {
    const { host, bar } = mountBar()
    await placeSubmenu(host)

    const button = bar.find('[data-title-bar-submenu="tide.new"]')
    expect(button.attributes('aria-haspopup')).toBe('menu')
    expect(button.attributes('aria-expanded')).toBe('false')
    expect(button.attributes('aria-label')).toBe('New')
  })

  it('opens the menu of its entries and runs the one picked', async () => {
    const { host, bar } = mountBar()
    await placeSubmenu(host)

    await bar.find('[data-title-bar-submenu="tide.new"]').trigger('click')
    expect(getMenuRows()).toStrictEqual(['Open chart.create', 'Open log.create'])
    expect(bar.find('[data-title-bar-submenu="tide.new"]').attributes('aria-expanded')).toBe('true')
    document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')[1]!.click()
    await nextTick()

    expect(host.ui.commands.getRuns()).toStrictEqual([
      { command: 'org.example.tide:log.create', target: undefined, itemId: undefined },
    ])
    expect(document.body.querySelector('[role="menu"]')).toBeNull()
  })
})

describe('the title', () => {
  it('names the application from the context and the vault the window shows', async () => {
    const { host, bar } = mountBar()

    await setVaults(host)

    expect(bar.find('[data-title]').text()).toBe('numen — Orchard notes')
  })

  it('names only the application while the window shows no vault', async () => {
    const { host, bar } = mountBar()

    host.vault.vaults.setState({ vaults: [ORCHARD, GREENHOUSE], current: null })
    await nextTick()

    expect(bar.find('[data-title]').text()).toBe('numen')
    expect(bar.find('[data-title]').element.tagName).toBe('SPAN')
  })

  it('switches to the vault chosen from the list the title opens', async () => {
    const { host, bar } = mountBar()
    await setVaults(host)

    await bar.find('[data-title]').trigger('click')
    await nextTick()
    document.body.querySelectorAll<HTMLElement>('[role="option"]')[1]!.click()
    await nextTick()

    expect(host.vault.vaults.getSwitches()).toStrictEqual(['greenhouse'])
  })

  it('runs the command of the window that creates a vault from the last row of the list', async () => {
    const { host, bar } = mountBar()
    host.ui.commands.setIds(['app:vault.create'])
    await setVaults(host)

    await bar.find('[data-title]').trigger('click')
    await nextTick()
    const options = [...document.body.querySelectorAll<HTMLElement>('[role="option"]')]
    expect(options.at(-1)?.textContent?.trim()).toBe('Create vault')
    options.at(-1)!.click()
    await nextTick()

    expect(host.ui.commands.getRuns()).toStrictEqual([
      { command: 'app:vault.create', target: undefined, itemId: undefined },
    ])
    expect(host.vault.vaults.getSwitches()).toStrictEqual([])
  })

  it('draws the row that creates a vault only while the window runs that command', async () => {
    const { host, bar } = mountBar()
    await setVaults(host)
    const labels = () =>
      [...document.body.querySelectorAll('[role="option"]')].map((one) => one.textContent?.trim())

    await bar.find('[data-title]').trigger('click')
    await nextTick()
    expect(labels()).not.toContain('Create vault')

    host.ui.commands.setIds(['app:vault.create'])
    await nextTick()
    expect(labels()).toContain('Create vault')
  })

  it('switches to nothing when the vault the window shows is chosen', async () => {
    const { host, bar } = mountBar()
    await setVaults(host)

    await bar.find('[data-title]').trigger('click')
    await nextTick()
    document.body.querySelectorAll<HTMLElement>('[role="option"]')[0]!.click()
    await nextTick()

    expect(host.vault.vaults.getSwitches()).toStrictEqual([])
  })

  it('follows a switch of the vault', async () => {
    const { host, bar } = mountBar()
    await setVaults(host)

    await setVaults(host, GREENHOUSE)

    expect(bar.find('[data-title]').text()).toBe('numen — Greenhouse log')
  })
})

describe('the window buttons', () => {
  it('minimize, maximize and close the window through the context', async () => {
    const { host, bar } = mountBar()

    await getControl(bar, 'minimize').trigger('click')
    await getControl(bar, 'toggle-maximize').trigger('click')
    await getControl(bar, 'close').trigger('click')

    expect(host.ui.window.getActions()).toStrictEqual(['minimize', 'toggleMaximize', 'close'])
  })

  it('toggle maximize on a double click on the free space', async () => {
    const { host, bar } = mountBar()

    await bar.find('[data-title-bar]').trigger('dblclick')

    expect(host.ui.window.getActions()).toStrictEqual(['toggleMaximize'])
  })

  it('draw the state of the window and follow its change', async () => {
    const { host, bar } = mountBar()
    expect(getControl(bar, 'toggle-maximize').attributes('title')).toBe('Maximize')

    host.ui.window.setState({ isMaximized: true, isFullscreen: false })
    await nextTick()

    expect(getControl(bar, 'toggle-maximize').attributes('title')).toBe('Restore')
    expect(getControl(bar, 'toggle-maximize').attributes('data-maximized')).toBe('true')
  })

  it('keep room for the macOS buttons, and none in full screen', async () => {
    const { host, bar } = mountBar({ platform: 'darwin' })
    expect(bar.find('[data-mac-buttons-inset]').exists()).toBe(true)

    host.ui.window.setState({ isMaximized: false, isFullscreen: true })
    await nextTick()

    expect(bar.find('[data-mac-buttons-inset]').exists()).toBe(false)
  })
})

describe('the window buttons by the setting', () => {
  it('stand at the right on Windows, and at the left when the setting says so', () => {
    const macBar = mountBar({ platform: 'windows', settings: { buttons: 'mac' } }).bar
    const right = mountBar({ platform: 'windows', settings: { buttons: 'windows' } }).bar

    expect(macBar.find('[data-title-bar]').attributes('data-controls')).toBe('mac')
    expect(right.find('[data-title-bar]').attributes('data-controls')).toBe('windows')
  })

  it('draw no control and keep no room when the setting says none', async () => {
    for (const platform of ['linux', 'windows'] as const) {
      const { bar } = mountBar({ platform, settings: { buttons: 'none' } })

      expect(bar.find('[data-title-bar]').attributes('data-controls')).toBe('none')
      expect(bar.findAll('[data-control]')).toHaveLength(0)
      expect(bar.find('[data-mac-buttons-inset]').exists()).toBe(false)
    }
  })

  it('keep the system buttons on macOS whatever the setting says', () => {
    for (const buttons of ['windows', 'mac', 'none']) {
      const { bar } = mountBar({ platform: 'darwin', settings: { buttons } })

      expect(bar.find('[data-title-bar]').attributes('data-controls')).toBe('native-mac')
      expect(bar.find('[data-mac-buttons-inset]').exists()).toBe(true)
    }
  })
})

describe('the setting hideInFullscreen', () => {
  const setFullscreen = async (host: MockHostServices, isFullscreen: boolean) => {
    host.ui.window.setState({ isMaximized: false, isFullscreen })
    await nextTick()
  }

  it('is declared as a boolean that is off by default', () => {
    const setting = manifest.settings.find((one) => one.key === 'hideInFullscreen')

    expect(setting).toMatchObject({ type: 'boolean', default: false })
  })

  it('draws the bar in full screen while the setting is off', async () => {
    const { host, bar } = mountBar()

    await setFullscreen(host, true)

    expect(bar.find('[data-title-bar]').exists()).toBe(true)
  })

  it('draws no bar in full screen while the setting is on, and draws it again after', async () => {
    const { host, bar } = mountBar({ settings: { hideInFullscreen: true } })
    expect(bar.find('[data-title-bar]').exists()).toBe(true)

    await setFullscreen(host, true)
    expect(bar.find('[data-title-bar]').exists()).toBe(false)
    expect(bar.html()).not.toContain('title-bar')

    await setFullscreen(host, false)
    expect(bar.find('[data-title-bar]').exists()).toBe(true)
  })

  it('draws no bar in a window that opens in full screen with the setting on', () => {
    const { host, view: component } = activateBar({ settings: { hideInFullscreen: true } })
    host.ui.window.setState({ isMaximized: false, isFullscreen: true })

    const bar = mount(component, { attachTo: document.body })
    mounted.push(bar)

    expect(bar.find('[data-title-bar]').exists()).toBe(false)
  })

  it('follows a change of the setting in full screen', async () => {
    const { context, host, bar } = mountBar()
    await setFullscreen(host, true)

    context.setSetting('hideInFullscreen', true)
    await nextTick()
    expect(bar.find('[data-title-bar]').exists()).toBe(false)

    context.setSetting('hideInFullscreen', false)
    await nextTick()
    expect(bar.find('[data-title-bar]').exists()).toBe(true)
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
})

describe('the bar in a window that offers no slots, vaults or window', () => {
  it('draws the title of the application and no entry', () => {
    const registerSlotView = vi.fn(() => ({ dispose: () => undefined }))
    const mock = createMockPluginContext()
    const host = createMockHostServices()
    const { window: _window, ...ui } = host.ui
    const { vaults: _vaults, ...vault } = host.vault
    const context: PluginContext = {
      ...mock,
      register: { ...mock.register, slotView: registerSlotView },
    }
    activate(context, { ...host, ui, vault })
    const [[registration]] = registerSlotView.mock.calls as unknown as [[SlotViewRegistration]]
    const bar = mount(registration.component as Component)
    mounted.push(bar)

    expect(bar.find('[data-title]').text()).toBe('numen')
    expect(bar.findAll('[data-title-bar-entry]')).toHaveLength(0)
  })
})

describe('removing the bar', () => {
  it('stops watching the slots, the vaults and the window', async () => {
    const { host, view: component } = activateBar()
    const disposed: string[] = []
    const track = <T extends { dispose: () => void }>(name: string, one: T): T => ({
      ...one,
      dispose: () => {
        disposed.push(name)
        one.dispose()
      },
    })
    const { slots, window } = host.ui
    const { vaults } = host.vault
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
