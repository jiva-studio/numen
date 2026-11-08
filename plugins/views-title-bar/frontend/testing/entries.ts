/** Invented slot entries for the tests and the stories of the bar. Not shipped. */
import { h, markRaw, type FunctionalComponent } from 'vue'
import type { SlotCommand, SlotItem, SlotSeparator, SlotSubmenu, SlotView } from '@numen/plugin-sdk'

const TIDE = 'org.example.tide'
const REEF = 'org.example.reef'

/** An icon that draws an empty mark, which a test finds by `data-icon`. */
const MARK: FunctionalComponent = markRaw(() => h('svg', { 'data-icon': 'mark' }))

export const command = (id: string, extra: Partial<SlotCommand> = {}): SlotCommand => ({
  kind: 'command',
  key: `${TIDE}/command/${id}`,
  pluginId: TIDE,
  command: `${TIDE}:${id}`,
  title: `Open ${id}`,
  icon: MARK,
  shortcut: '',
  hasItems: false,
  ...extra,
})

const createLabelView = (text: string): FunctionalComponent =>
  markRaw(() => h('span', { 'data-label-view': text }, text))

export const view = (id: string, component: unknown = createLabelView(id)): SlotView => ({
  kind: 'view',
  key: `${REEF}/view/${id}`,
  pluginId: REEF,
  view: id,
  component,
})

export const separator = (index: number): SlotSeparator => ({
  kind: 'separator',
  key: `separator/${index}`,
})

export const submenu = (
  id: string,
  entries: readonly SlotItem[],
  extra: Partial<SlotSubmenu> = {},
): SlotSubmenu => ({
  kind: 'submenu',
  key: `${TIDE}/submenu/${id}`,
  pluginId: TIDE,
  submenu: id,
  title: 'New',
  entries,
  ...extra,
})
