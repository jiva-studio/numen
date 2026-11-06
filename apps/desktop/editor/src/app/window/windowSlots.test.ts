/**
 * The slots of the editor window as the plugin host serves them: the buttons of the title bar in the
 * order of the slot and the settings button of the application last. Every plugin here is invented.
 */
import { describe, expect, it, vi } from 'vitest'
import { computed } from 'vue'
import { SLOT, createSlotRegistry, type PluginHost, type SlotItem } from '@numen/plugin-host'
import type { DeclaredCommand, DeclaredSlot } from '@/entities/plugin'
import { createWindowSlots } from './windowSlots'

const TIDE = 'org.example.tide'
const HARBOR = 'org.example.harbor'

const command = (pluginId: string, id: string, title: string): DeclaredCommand => ({
  pluginId,
  id,
  title,
  contentTypes: [],
  extensions: [],
  arguments: [],
})

const COMMANDS = [
  command(TIDE, 'tide.open', 'Open tide'),
  command(TIDE, 'tide.add', 'Add tide'),
  command(HARBOR, 'harbor.open', 'Open harbor'),
]

const SLOTS: DeclaredSlot[] = [
  { pluginId: TIDE, slot: SLOT.titleBarCommands, command: 'tide.add', group: 'new' },
  {
    pluginId: HARBOR,
    slot: SLOT.titleBarCommands,
    command: 'harbor.open',
    group: 'navigate',
    order: 20,
  },
  {
    pluginId: TIDE,
    slot: SLOT.titleBarCommands,
    command: 'tide.open',
    group: 'navigate',
    order: 10,
  },
]

function setup() {
  const slots = createWindowSlots({
    declaredCommands: { slots: computed(() => SLOTS), commands: computed(() => COMMANDS) },
    hostHolder: { current: { getPaletteCommands: () => [] } as unknown as PluginHost },
    writeIssue: vi.fn(),
  })
  const registry = createSlotRegistry({
    getEntries: slots.hostDeps.getSlotEntries,
    findCommand: slots.hostDeps.findSlotCommand,
    findView: () => undefined,
  })
  return { registry }
}

const describeItems = (items: readonly SlotItem[]) =>
  items.map((one) =>
    one.kind === 'command' ? one.command : one.kind === 'separator' ? '|' : one.kind,
  )

describe('the buttons of the title bar', () => {
  it('sort by group and order, with the settings button of the application last', () => {
    const { registry } = setup()

    expect(describeItems(registry.getEntries(SLOT.titleBarCommands))).toStrictEqual([
      `${TIDE}:tide.open`,
      `${HARBOR}:harbor.open`,
      '|',
      `${TIDE}:tide.add`,
      '|',
      'app:settings.open',
    ])
  })

  it('carry the title of their command', () => {
    const { registry } = setup()

    const [first] = registry.getEntries(SLOT.titleBarCommands)

    expect(first).toMatchObject({ kind: 'command', title: 'Open tide' })
  })
})
