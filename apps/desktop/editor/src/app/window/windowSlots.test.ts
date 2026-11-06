/**
 * The slots of the editor window as the plugin host serves them: the buttons of the title bar in the
 * order of the slot, the settings button of the application last, and nothing while no vault is open.
 * Every plugin here is invented.
 */
import { describe, expect, it, vi } from 'vitest'
import { computed, ref } from 'vue'
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

function setup(isOpen = true) {
  const open = ref(isOpen)
  const slots = createWindowSlots({
    declaredCommands: { slots: computed(() => SLOTS), commands: computed(() => COMMANDS) },
    hostHolder: { current: { getPaletteCommands: () => [] } as unknown as PluginHost },
    writeIssue: vi.fn(),
  })
  slots.vaultOpen.current = () => open.value
  const registry = createSlotRegistry({
    getEntries: slots.hostDeps.getSlotEntries,
    findCommand: slots.hostDeps.findSlotCommand,
    findView: () => undefined,
  })
  return { registry, open }
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

  it('are none, the settings button included, while no vault is open, and come back once one is', () => {
    const { registry, open } = setup(false)

    expect(registry.getEntries(SLOT.titleBarCommands)).toStrictEqual([])
    open.value = true
    expect(registry.getEntries(SLOT.titleBarCommands).at(-1)).toMatchObject({
      command: 'app:settings.open',
    })
  })
})
