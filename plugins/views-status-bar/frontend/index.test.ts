/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, describe, expect, it } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { h, type Component, type FunctionalComponent } from 'vue'
import type { SlotCommand, SlotItem, SlotSubmenu, SlotView } from '@numen/plugin-sdk'
import { createMockPluginContext, type MockPluginContext } from '@numen/plugin-sdk/testing'
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

const activateBar = (): { context: MockPluginContext; view: Component } => {
  const context = createMockPluginContext({ pluginId: manifest.id })
  activate(context)
  const [registration] = context.getRegisteredSlotViews()
  return { context, view: registration!.component as Component }
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
    const { context, view } = activateBar()
    context.slots.setEntries('statusBar.left', [createCommand('count', 'Count words')])
    context.slots.setEntries('statusBar.right', [createCommand('lang', 'Language')])

    const wrapper = mountBar(view)

    expect(wrapper.find('[data-status-bar-slot="statusBar.left"]').text()).toBe('Count words')
    expect(wrapper.find('[data-status-bar-slot="statusBar.right"]').text()).toBe('Language')
  })

  it('follows the changes of the entries', async () => {
    const { context, view } = activateBar()
    const wrapper = mountBar(view)

    context.slots.setEntries('statusBar.left', [createCommand('count', 'Count words')])
    await flushPromises()

    expect(wrapper.find('[data-status-bar-slot="statusBar.left"]').text()).toBe('Count words')
  })

  it('draws an empty bar where the window offers no slots', () => {
    const context = createMockPluginContext({ pluginId: manifest.id })
    const { slots: _slots, ...bare } = context
    activate(bare)
    const [registration] = context.getRegisteredSlotViews()

    const wrapper = mountBar(registration!.component as Component)

    expect(wrapper.find('footer[role="region"]').exists()).toBe(true)
    expect(wrapper.findAll('[data-status-bar-entry]')).toHaveLength(0)
  })
})

describe('a command', () => {
  it('is a button that runs the command with no target', async () => {
    const { context, view } = activateBar()
    context.slots.setEntries('statusBar.left', [createCommand('count', 'Count words')])
    const wrapper = mountBar(view)

    await wrapper.find('button[data-status-bar-entry]').trigger('click')

    expect(context.commands.getRuns()).toStrictEqual([
      { command: `${PLUGIN}:count`, target: undefined, argument: undefined },
    ])
  })

  it('shows the icon of the command with its title and shortcut as the hover text', () => {
    const { context, view } = activateBar()
    context.slots.setEntries('statusBar.left', [
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
    const { context, view } = activateBar()
    context.slots.setEntries('statusBar.left', [
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
    const { context, view } = activateBar()
    context.slots.setEntries('statusBar.right', [createView('words', '1 024 words')])

    const wrapper = mountBar(view)

    expect(wrapper.find('[data-status-bar-slot="statusBar.right"] [data-word-count]').text()).toBe(
      '1 024 words',
    )
  })
})

describe('a submenu', () => {
  const submenu = createSubmenu('words.more', 'Words', [
    createCommand('count', 'Count words'),
    createSeparator('sep-1'),
    createSubmenu('words.lang', 'Language', [createCommand('english', 'English')]),
  ])

  it('is a button that opens the list of its entries', async () => {
    const { context, view } = activateBar()
    context.slots.setEntries('statusBar.right', [submenu])
    const wrapper = mountBar(view)
    expect(wrapper.find('[role="menu"]').exists()).toBe(false)

    const trigger = wrapper.find('button[data-status-bar-entry]')
    expect(trigger.text()).toBe('Words')
    await trigger.trigger('click')

    const menu = wrapper.find('[role="menu"]')
    expect(menu.attributes('aria-label')).toBe('Words')
    expect(menu.findAll(':scope > [role="menuitem"]').map((one) => one.text())).toStrictEqual([
      'Count words',
      'Language',
    ])
    expect(menu.findAll(':scope > [role="separator"]')).toHaveLength(1)
  })

  it('runs a command of the list and closes it', async () => {
    const { context, view } = activateBar()
    context.slots.setEntries('statusBar.right', [submenu])
    const wrapper = mountBar(view)
    await wrapper.find('button[data-status-bar-entry]').trigger('click')

    await wrapper.find('[role="menuitem"]').trigger('click')

    expect(context.commands.getRuns()).toStrictEqual([
      { command: `${PLUGIN}:count`, target: undefined, argument: undefined },
    ])
    expect(wrapper.find('[role="menu"]').exists()).toBe(false)
  })

  it('opens a nested submenu from its row', async () => {
    const { context, view } = activateBar()
    context.slots.setEntries('statusBar.right', [submenu])
    const wrapper = mountBar(view)
    await wrapper.find('button[data-status-bar-entry]').trigger('click')

    const row = wrapper.findAll('[role="menuitem"]').find((one) => one.text() === 'Language')!
    expect(row.attributes('aria-haspopup')).toBe('menu')
    await row.trigger('click')

    const nested = wrapper.find('[role="menu"][aria-label="Language"]')
    expect(nested.text()).toBe('English')
    await nested.find('[role="menuitem"]').trigger('click')
    expect(context.commands.getRuns()).toStrictEqual([
      { command: `${PLUGIN}:english`, target: undefined, argument: undefined },
    ])
  })

  it('closes on Escape', async () => {
    const { context, view } = activateBar()
    context.slots.setEntries('statusBar.right', [submenu])
    const wrapper = mountBar(view)
    await wrapper.find('button[data-status-bar-entry]').trigger('click')

    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()

    expect(wrapper.find('[role="menu"]').exists()).toBe(false)
  })

  it('draws a view among its entries as a row that holds it', async () => {
    const { context, view } = activateBar()
    context.slots.setEntries('statusBar.right', [
      createSubmenu('words.more', 'Words', [createView('words', '1 024 words')]),
    ])
    const wrapper = mountBar(view)

    await wrapper.find('button[data-status-bar-entry]').trigger('click')

    expect(wrapper.find('[role="menu"] [data-word-count]').text()).toBe('1 024 words')
  })
})
