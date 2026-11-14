/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { h, type Component, type FunctionalComponent } from 'vue'
import type { SlotCommand, SlotItem, SlotSubmenu, SlotView } from '@numen/plugin-sdk'
import {
  createMockNoticeHost,
  createMockHostServices,
  createMockPluginContext,
  type MockHostServices,
  type MockPluginContext,
} from '@numen/plugin-sdk/testing'
import { activate, STATUS_BAR_VIEW } from './index'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  id: string
  targets: readonly string[]
  contributes: { slots: readonly { slot: string; view?: string }[] }
}

const PLUGIN = 'org.example.words'

const createCommand = (command: string, title: string, extra: Partial<SlotCommand> = {}) =>
  ({
    kind: 'command',
    key: `${PLUGIN}/command/${command}`,
    pluginId: PLUGIN,
    command: `${PLUGIN}:${command}`,
    title,
    shortcut: '',
    hasItems: false,
    ...extra,
  }) satisfies SlotCommand

const createSeparator = (key: string): SlotItem => ({ kind: 'separator', key })

const createView = (view: string, text: string): SlotView => ({
  kind: 'view',
  key: `${PLUGIN}/view/${view}`,
  pluginId: PLUGIN,
  view,
  component: (() => h('span', { 'data-word-count': '' }, text)) satisfies FunctionalComponent,
})

const createSubmenu = (
  submenu: string,
  title: string,
  entries: readonly SlotItem[],
): SlotSubmenu => ({
  kind: 'submenu',
  key: `${PLUGIN}/submenu/${submenu}`,
  pluginId: PLUGIN,
  submenu,
  title,
  entries,
})

const icon: FunctionalComponent = () => h('svg', { 'data-icon': 'words' })

const mounted: VueWrapper[] = []

afterEach(() => {
  for (const one of mounted.splice(0)) one.unmount()
})

const activateBar = (): { context: MockPluginContext; host: MockHostServices; view: Component } => {
  const context = createMockPluginContext({ pluginId: manifest.id })
  const host = createMockHostServices()
  activate(context, host)
  const [registration] = context.getRegisteredSlotViews()
  return { context, host, view: registration!.component as Component }
}

const mountBar = (view: Component): VueWrapper => {
  const wrapper = mount(view, { attachTo: document.body })
  mounted.push(wrapper)
  return wrapper
}

describe('the manifest', () => {
  it('places the bar in the status bar slot of the window', () => {
    expect(manifest.id).toBe('md.numen.views.statusBar')
    expect(manifest.contributes.slots).toStrictEqual([{ slot: 'statusBar', view: STATUS_BAR_VIEW }])
  })

  it('registers the bar under the id the manifest places', () => {
    const { context } = activateBar()

    expect(context.getRegisteredSlotViews().map((one) => one.id)).toStrictEqual([STATUS_BAR_VIEW])
  })
})

describe('the bar', () => {
  it('is a region with a start and an end and nothing in them while no entry is placed', () => {
    const { view } = activateBar()

    const wrapper = mountBar(view)

    const bar = wrapper.find('footer[role="region"]')
    expect(bar.attributes('aria-label')).toBe('Status bar')
    expect(wrapper.find('[data-status-bar-slot="statusBar.left"]').element.children).toHaveLength(0)
    expect(wrapper.find('[data-status-bar-slot="statusBar.right"]').element.children).toHaveLength(
      0,
    )
  })

  it('draws the entries of the start and of the end in their parts', () => {
    const { host, view } = activateBar()
    host.ui.slots.setEntries('statusBar.left', [createCommand('count', 'Count words')])
    host.ui.slots.setEntries('statusBar.right', [createCommand('lang', 'Language')])

    const wrapper = mountBar(view)

    expect(wrapper.find('[data-status-bar-slot="statusBar.left"]').text()).toBe('Count words')
    expect(wrapper.find('[data-status-bar-slot="statusBar.right"]').text()).toBe('Language')
  })

  it('follows the changes of the entries', async () => {
    const { host, view } = activateBar()
    const wrapper = mountBar(view)

    host.ui.slots.setEntries('statusBar.left', [createCommand('count', 'Count words')])
    await flushPromises()

    expect(wrapper.find('[data-status-bar-slot="statusBar.left"]').text()).toBe('Count words')
  })
})

describe('a command', () => {
  it('is a button that runs the command with no target', async () => {
    const { host, view } = activateBar()
    host.ui.slots.setEntries('statusBar.left', [createCommand('count', 'Count words')])
    const wrapper = mountBar(view)

    await wrapper.find('button[data-status-bar-entry]').trigger('click')

    expect(host.ui.commands.getRuns()).toStrictEqual([
      { command: `${PLUGIN}:count`, target: undefined, itemId: undefined },
    ])
  })

  it('shows the error of a run that fails as one notice', async () => {
    const notices = createMockNoticeHost()
    const context = createMockPluginContext({ pluginId: manifest.id })
    const host = createMockHostServices({ notices })
    activate(context, host)
    const view = context.getRegisteredSlotViews()[0]!.component as Component
    host.ui.slots.setEntries('statusBar.left', [createCommand('count', 'Count words')])
    vi.spyOn(host.ui.commands, 'run').mockRejectedValueOnce(new Error('the words are gone'))
    const wrapper = mountBar(view)

    await wrapper.find('button[data-status-bar-entry]').trigger('click')
    await flushPromises()

    expect(notices.getShown()).toStrictEqual([{ text: 'the words are gone', severity: 'error' }])
  })

  it('shows the icon of the command with its title and shortcut as the hover text', () => {
    const { host, view } = activateBar()
    host.ui.slots.setEntries('statusBar.left', [
      createCommand('count', 'Count words', { icon, shortcut: 'mod+shift+w' }),
    ])

    const wrapper = mountBar(view)

    const button = wrapper.find('button[data-status-bar-entry]')
    expect(button.find('[data-icon="words"]').exists()).toBe(true)
    expect(button.attributes('title')).toBe('Count words (mod+shift+w)')
    expect(button.attributes('aria-label')).toBe('Count words')
  })
})

describe('a separator', () => {
  it('is a gap between two groups', () => {
    const { host, view } = activateBar()
    host.ui.slots.setEntries('statusBar.left', [
      createCommand('count', 'Count words'),
      createSeparator('sep-1'),
      createCommand('lang', 'Language'),
    ])

    const wrapper = mountBar(view)

    const children = [...wrapper.find('[data-status-bar-slot="statusBar.left"]').element.children]
    expect(children.map((one) => one.hasAttribute('data-status-bar-gap'))).toStrictEqual([
      false,
      true,
      false,
    ])
  })
})

describe('a view', () => {
  it('is drawn in place', () => {
    const { host, view } = activateBar()
    host.ui.slots.setEntries('statusBar.right', [createView('words', '1 024 words')])

    const wrapper = mountBar(view)

    expect(wrapper.find('[data-status-bar-slot="statusBar.right"] [data-word-count]').text()).toBe(
      '1 024 words',
    )
  })
})

const getMenus = (): HTMLElement[] => [
  ...document.body.querySelectorAll<HTMLElement>('[role="menu"]'),
]
const getRows = (menu: HTMLElement | undefined): HTMLElement[] => [
  ...(menu?.querySelectorAll<HTMLElement>(':scope > [role="menuitem"]') ?? []),
]

describe('a submenu', () => {
  const submenu = createSubmenu('words.more', 'Words', [
    createCommand('count', 'Count words'),
    createSeparator('sep-1'),
    createSubmenu('words.lang', 'Language', [createCommand('english', 'English')]),
  ])

  it('is a button that opens the list of its entries', async () => {
    const { host, view } = activateBar()
    host.ui.slots.setEntries('statusBar.right', [submenu])
    const wrapper = mountBar(view)
    expect(getMenus()).toHaveLength(0)

    const trigger = wrapper.find('button[data-status-bar-entry]')
    expect(trigger.text()).toBe('Words')
    await trigger.trigger('click')
    await flushPromises()

    const [menu] = getMenus()
    expect(menu?.getAttribute('aria-label')).toBe('Words')
    expect(getRows(menu).map((one) => one.textContent?.trim())).toStrictEqual([
      'Count words',
      'Language ›',
    ])
    expect(menu?.querySelectorAll(':scope > [role="separator"]')).toHaveLength(1)
  })

  it('runs a command of the list and closes it', async () => {
    const { host, view } = activateBar()
    host.ui.slots.setEntries('statusBar.right', [submenu])
    const wrapper = mountBar(view)
    await wrapper.find('button[data-status-bar-entry]').trigger('click')
    await flushPromises()

    getRows(getMenus()[0])[0]!.click()
    await flushPromises()

    expect(host.ui.commands.getRuns()).toStrictEqual([
      { command: `${PLUGIN}:count`, target: undefined, itemId: undefined },
    ])
    expect(getMenus()).toHaveLength(0)
  })

  it('opens a nested submenu from its row', async () => {
    const { host, view } = activateBar()
    host.ui.slots.setEntries('statusBar.right', [submenu])
    const wrapper = mountBar(view)
    await wrapper.find('button[data-status-bar-entry]').trigger('click')
    await flushPromises()

    const row = getRows(getMenus()[0]).find((one) => one.textContent?.includes('Language'))!
    expect(row.getAttribute('aria-haspopup')).toBe('menu')
    row.click()
    await flushPromises()

    const nested = getMenus()[1]
    expect(nested?.textContent?.trim()).toBe('English')
    getRows(nested)[0]!.click()
    await flushPromises()
    expect(host.ui.commands.getRuns()).toStrictEqual([
      { command: `${PLUGIN}:english`, target: undefined, itemId: undefined },
    ])
  })

  it('closes on Escape', async () => {
    const { host, view } = activateBar()
    host.ui.slots.setEntries('statusBar.right', [submenu])
    const wrapper = mountBar(view)
    await wrapper.find('button[data-status-bar-entry]').trigger('click')
    await flushPromises()

    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()

    expect(getMenus()).toHaveLength(0)
  })

  it('draws a view among its entries as a row that holds it', async () => {
    const { host, view } = activateBar()
    host.ui.slots.setEntries('statusBar.right', [
      createSubmenu('words.more', 'Words', [createView('words', '1 024 words')]),
    ])
    const wrapper = mountBar(view)

    await wrapper.find('button[data-status-bar-entry]').trigger('click')
    await flushPromises()

    expect(document.body.querySelector('[role="menu"] [data-word-count]')?.textContent).toBe(
      '1 024 words',
    )
  })
})

describe('a command with items', () => {
  const LANGUAGE = createCommand('lang', 'Language', { hasItems: true })

  it('is a button that opens a dropdown of the items of the command', async () => {
    const { host, view } = activateBar()
    host.ui.commands.setItems(LANGUAGE.command, [
      { id: 'en', title: 'English' },
      { id: 'sa', title: 'Sanskrit' },
    ])
    host.ui.slots.setEntries('statusBar.right', [LANGUAGE])
    const wrapper = mountBar(view)

    await wrapper.find('button[data-status-bar-entry]').trigger('click')
    await flushPromises()

    expect(getRows(getMenus()[0]).map((one) => one.textContent?.trim())).toStrictEqual([
      'English',
      'Sanskrit',
    ])
  })

  it('opens no dropdown where the command lists no item', async () => {
    const { host, view } = activateBar()
    host.ui.commands.setItems(LANGUAGE.command, [])
    host.ui.slots.setEntries('statusBar.right', [LANGUAGE])
    const wrapper = mountBar(view)

    await wrapper.find('button[data-status-bar-entry]').trigger('click')
    await flushPromises()

    expect(getMenus().flatMap((menu) => getRows(menu))).toStrictEqual([])
  })

  it('opens on the next click after a click whose command listed no item', async () => {
    const { host, view } = activateBar()
    host.ui.commands.setItems(LANGUAGE.command, [])
    host.ui.slots.setEntries('statusBar.right', [LANGUAGE])
    const wrapper = mountBar(view)
    await wrapper.find('button[data-status-bar-entry]').trigger('click')
    await flushPromises()

    host.ui.commands.setItems(LANGUAGE.command, [{ id: 'sa', title: 'Sanskrit' }])
    await wrapper.find('button[data-status-bar-entry]').trigger('click')
    await flushPromises()

    expect(getRows(getMenus()[0]).map((one) => one.textContent?.trim())).toStrictEqual(['Sanskrit'])
  })

  it('shows the error of a list that failed to read', async () => {
    const notices = createMockNoticeHost()
    const context = createMockPluginContext({ pluginId: manifest.id })
    const host = createMockHostServices({ notices })
    vi.spyOn(host.ui.commands, 'listItems').mockRejectedValue(new Error('the languages are gone'))
    activate(context, host)
    host.ui.slots.setEntries('statusBar.right', [LANGUAGE])
    const wrapper = mountBar(context.getRegisteredSlotViews()[0]!.component as Component)

    await wrapper.find('button[data-status-bar-entry]').trigger('click')
    await flushPromises()

    expect(notices.getShown()).toStrictEqual([
      { text: 'the languages are gone', severity: 'error' },
    ])
  })

  it('runs the command with the item picked as its itemId', async () => {
    const { host, view } = activateBar()
    host.ui.commands.setItems(LANGUAGE.command, [{ id: 'sa', title: 'Sanskrit' }])
    host.ui.slots.setEntries('statusBar.right', [LANGUAGE])
    const wrapper = mountBar(view)
    await wrapper.find('button[data-status-bar-entry]').trigger('click')
    await flushPromises()

    getRows(getMenus()[0])[0]!.click()
    await flushPromises()

    expect(host.ui.commands.getRuns()).toStrictEqual([
      { command: LANGUAGE.command, target: undefined, itemId: 'sa' },
    ])
  })
})
