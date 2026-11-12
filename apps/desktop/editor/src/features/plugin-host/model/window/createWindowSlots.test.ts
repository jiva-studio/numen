/**
 * The slots of the editor window as the plugin host serves them: the buttons of the title bar in the
 * order of the slot and the settings button of the application last. Every plugin here is invented.
 */
import { describe, expect, it, vi } from 'vitest'
import { computed } from 'vue'
import { createWhenReader, SLOT, createSlotRegistry, type SlotItem } from '@numen/plugin-loader'
import type { DeclaredCommand, DeclaredSlot } from '@/entities/plugin'
import type { Command } from '@/entities/command'
import { createAppCommands } from '@/testing/appCommands'
import { createWindowSlots } from './createWindowSlots'

const TIDE = 'org.example.tide'
const HARBOR = 'org.example.harbor'

const command = (pluginId: string, id: string, title: string): DeclaredCommand => ({
  pluginId,
  id,
  title,
  arguments: [],
})

const COMMANDS = [
  command(TIDE, 'tide.open', 'Open tide'),
  command(TIDE, 'tide.add', 'Add tide'),
  { ...command(TIDE, 'tide.vault', 'Open vault view'), when: 'window.isVaultOpen' },
  command(HARBOR, 'harbor.open', 'Open harbor'),
  { ...command(HARBOR, 'harbor.moor', 'Moor link'), when: "target.noteType == 'link'" },
  command(HARBOR, 'harbor.dock', 'Dock'),
  {
    ...command(HARBOR, 'harbor.read', 'Read aloud'),
    when: 'version.isLive',
    hasItems: true as const,
  },
]

const MOVE: Command = {
  id: 'app:tab.moveToWindow',
  text: 'Move to window',
  group: 'window',
  input: 'items',
  isAvailable: () => true,
  listItems: () => [],
}

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

function setup(slotEntries: readonly DeclaredSlot[] = SLOTS) {
  const state = { isVaultOpen: true }
  const facts = {
    getAppCommands: () => [...createAppCommands(), MOVE],
    getValues: () => ({ 'window.isVaultOpen': state.isVaultOpen }),
  }
  const host = {
    commands: {
      getPaletteCommands: () => [],
      hasCommandItems: (id: string) => id === `${HARBOR}:harbor.dock`,
    },
  }
  const slots = createWindowSlots({
    declaredCommands: {
      slots: computed(() => slotEntries),
      commands: computed(() => COMMANDS),
      contributions: { handlers: computed(() => []) },
      whenContext: {
        getContentTypes: () => ({ md: 'text/markdown' }),
        getConfig: () => ({}),
        whenReader: createWhenReader(),
      },
    },
    facts,
    writeIssue: vi.fn(),
  })
  const findSlotCommand = slots.hostDeps.createSlotCommandFinder({
    getPluginCommands: host.commands.getPaletteCommands,
    hasPluginItems: host.commands.hasCommandItems,
  })
  const registry = createSlotRegistry({
    getEntries: () => slots.hostDeps.getSlotEntries(new Map()),
    findCommand: findSlotCommand,
    findView: () => undefined,
    getWhenValues: slots.hostDeps.getSlotWhenValues,
    whenReader: slots.hostDeps.whenReader,
  })
  return { registry, hostDeps: slots.hostDeps, state, findSlotCommand }
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

  it('carry the title of their command, and no items', () => {
    const { registry } = setup()

    expect(registry.getEntries(SLOT.titleBarCommands)[0]).toMatchObject({ hasItems: false })
  })

  it('carry the title of their command', () => {
    const { registry } = setup()

    const [first] = registry.getEntries(SLOT.titleBarCommands)

    expect(first).toMatchObject({ kind: 'command', title: 'Open tide' })
  })
})

describe('the buttons of the title bar while no vault is open', () => {
  const declaredSlots: DeclaredSlot[] = [
    { pluginId: TIDE, slot: SLOT.titleBarCommands, command: 'tide.vault', group: 'navigate' },
  ]
  const setVault = (state: ReturnType<typeof setup>['state'], isOpen: boolean) => {
    state.isVaultOpen = isOpen
  }

  it('are none, the settings button too', () => {
    const { registry, state } = setup(declaredSlots)
    setVault(state, false)

    expect(describeItems(registry.getEntries(SLOT.titleBarCommands))).toStrictEqual([])
  })

  it('appear when a vault is open', () => {
    const { registry, state } = setup(declaredSlots)
    setVault(state, true)

    expect(describeItems(registry.getEntries(SLOT.titleBarCommands))).toStrictEqual([
      `${TIDE}:tide.vault`,
      '|',
      'app:settings.open',
    ])
  })

  it('leave the command of the settings available to the palette and the shortcut', () => {
    const { findSlotCommand, state } = setup(declaredSlots)
    setVault(state, false)

    expect(findSlotCommand('app:settings.open', undefined)).toMatchObject({
      title: expect.any(String),
    })
  })
})

describe('the entries of a menu over a target', () => {
  const LINK = {
    path: 'links/Pier.md',
    title: 'Pier',
    isFolder: false,
    others: [],
    tab: '',
    noteType: 'link',
  }
  const menu: DeclaredSlot[] = [
    { pluginId: HARBOR, slot: SLOT.fileTreeContext, command: 'harbor.moor' },
    { pluginId: HARBOR, slot: SLOT.fileTreeContext, command: 'harbor.dock' },
    { pluginId: HARBOR, slot: SLOT.fileTreeContext, command: 'harbor.read' },
  ]

  it('read the note type of the target, which the place gives with it', () => {
    const { registry } = setup(menu)

    expect(describeItems(registry.getEntries(SLOT.fileTreeContext, LINK))).toContain(
      `${HARBOR}:harbor.moor`,
    )
    expect(
      describeItems(registry.getEntries(SLOT.fileTreeContext, { ...LINK, noteType: 'note' })),
    ).not.toContain(`${HARBOR}:harbor.moor`)
  })

  it('mark a command of a plugin that lists items', () => {
    const { registry } = setup(menu)

    expect(registry.getEntries(SLOT.fileTreeContext, LINK)).toContainEqual(
      expect.objectContaining({ command: `${HARBOR}:harbor.dock`, hasItems: true }),
    )
  })

  it('mark a command a backend runs that the core lists with items', () => {
    const { registry } = setup(menu)

    expect(registry.getEntries(SLOT.fileTreeContext, LINK)).toContainEqual(
      expect.objectContaining({ command: `${HARBOR}:harbor.read`, hasItems: true }),
    )
  })

  it('draw the commands the window makes as it runs, and mark those that list items', () => {
    const { registry } = setup(menu)

    expect(registry.getEntries(SLOT.tabStripContext, { ...LINK, tab: 'reef' })).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          command: 'app:tab.moveToWindow',
          title: 'Move to window',
          hasItems: true,
        }),
      ]),
    )
  })
})

describe('the when values of a target', () => {
  it('carry the note type and the properties the place read', () => {
    const { hostDeps } = setup()
    const target = {
      path: 'links/talk.md',
      title: 'talk',
      isFolder: false,
      others: [],
      tab: '',
      noteType: 'link',
      properties: { media: 'plugin://md.importer/a.mp4' },
    }

    expect(hostDeps.getSlotWhenValues(target)).toMatchObject({
      'target.noteType': 'link',
      'target.properties': { media: 'plugin://md.importer/a.mp4' },
    })
  })
})
