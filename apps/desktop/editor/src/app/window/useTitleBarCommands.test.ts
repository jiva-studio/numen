/**
 * The commands of the title bar: what an entry of its slot becomes, when the bar shows any, and
 * what a press runs.
 */
import { describe, expect, it, vi } from 'vitest'
import { markRaw, shallowRef, type Component } from 'vue'
import { createSlotRegistry } from '@numen/plugin-host'
import type { SlotItem } from '@numen/plugin-host'
import type { CommandTarget } from '@/features/command-palette'
import type { DeclaredCommand, DeclaredSlot } from '@/entities/plugin'
import { createSlotCommandFinder, readSlotEntries } from '../commands/pluginSlots'
import { formatShortcut, useTitleBarCommands } from './useTitleBarCommands'

const LINUX = 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/605.1.15'
const MAC = 'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/605.1.15'
const TIDE = 'org.example.tide'
const REEF = 'org.example.reef'

const target = {
  tab: 'tab-1',
  kind: 'note',
  path: 'Kelp.md',
  title: 'Kelp',
  file: 'Kelp.md',
  vault: { id: 'v', name: 'Brine notes' },
  isReady: true,
} as unknown as CommandTarget

const icons = new Map<string, Component>()
const getIcon = (name: string): Component => {
  if (!icons.has(name)) icons.set(name, markRaw({ render: () => null, name }))
  return icons.get(name)!
}

interface EntryLook {
  readonly pluginId?: string
  readonly title?: string
  readonly icon?: string
  readonly shortcut?: string
}

const entry = (command: string, look: EntryLook = {}): SlotItem => ({
  kind: 'command',
  key: `${look.pluginId ?? TIDE}/command/${command}`,
  pluginId: look.pluginId ?? TIDE,
  command,
  title: look.title ?? `Title of ${command}`,
  icon: getIcon(look.icon ?? 'waves'),
  shortcut: look.shortcut ?? '',
  hasItems: false,
})

/** The separator between two groups of the slot. */
const gap: SlotItem = { kind: 'separator', key: 'titleBar.commands/separator/navigate' }

function setup(
  options: {
    entries?: readonly SlotItem[]
    isVaultOpen?: boolean
    agent?: string
  } = {},
) {
  const state = {
    entries: options.entries ?? [],
    isVaultOpen: options.isVaultOpen ?? true,
  }
  const entries = shallowRef(state.entries)
  const isVaultOpen = shallowRef(state.isVaultOpen)
  const getEntries = vi.fn((_slot: string) => entries.value)
  const runCommand = vi.fn<(id: string, target: CommandTarget) => void>()
  const getTarget = vi.fn(() => target)
  const bar = useTitleBarCommands({
    slots: { getEntries },
    isVaultOpen: () => isVaultOpen.value,
    runCommand,
    getTarget,
    agent: options.agent ?? LINUX,
  })
  return { bar, entries, isVaultOpen, getEntries, runCommand, getTarget }
}

describe('the commands of the title bar', () => {
  const tree = entry(`${TIDE}:tree.open`, {
    title: 'Open tide chart',
    icon: 'waves',
    shortcut: 'mod+shift+e',
  })
  const graph = entry(`${REEF}:graph.open`, { pluginId: REEF, title: 'Open reef map', icon: 'map' })
  const note = entry('app:note.create', { title: 'Create note', icon: 'file-plus' })

  it('are the commands of the title bar slot in the order the slot gives them', () => {
    const { bar, getEntries } = setup({ entries: [tree, graph, gap, note] })

    expect(bar.commands.value.map((one) => one.id)).toStrictEqual([
      `${TIDE}:tree.open`,
      `${REEF}:graph.open`,
      'app:note.create',
    ])
    expect(getEntries).toHaveBeenCalledWith('titleBar.commands')
  })

  it('carry the title and the icon of the command', () => {
    const { bar } = setup({ entries: [tree, graph] })

    const [first, second] = bar.commands.value
    expect(first?.title).toBe('Open tide chart')
    expect(second?.title).toBe('Open reef map')
    expect(first?.icon).toBe(getIcon('waves'))
    expect(second?.icon).toBe(getIcon('map'))
  })

  it('carry the group the separators before them count', () => {
    const { bar } = setup({ entries: [tree, graph, gap, note] })

    expect(bar.commands.value.map((one) => one.group)).toStrictEqual(['0', '0', '1'])
  })

  it('sort, filter and merge nothing of what the slot gives', () => {
    const { bar } = setup({
      entries: [note, graph, tree, entry('app:vault.create')],
    })

    expect(bar.commands.value.map((one) => one.id)).toStrictEqual([
      'app:note.create',
      `${REEF}:graph.open`,
      `${TIDE}:tree.open`,
      'app:vault.create',
    ])
  })

  it('are empty where the slot has no entries', () => {
    const { bar } = setup({ entries: [] })

    expect(bar.commands.value).toStrictEqual([])
  })

  it('are empty while no vault is open, whatever the slot gives', () => {
    const { bar, isVaultOpen } = setup({ entries: [tree, note], isVaultOpen: false })

    expect(bar.commands.value).toStrictEqual([])

    isVaultOpen.value = true

    expect(bar.commands.value.map((one) => one.id)).toStrictEqual([
      `${TIDE}:tree.open`,
      'app:note.create',
    ])

    isVaultOpen.value = false

    expect(bar.commands.value).toStrictEqual([])
  })

  it('follow the slot when its entries change', () => {
    const { bar, entries } = setup({ entries: [tree] })
    expect(bar.commands.value).toHaveLength(1)

    entries.value = [tree, graph]

    expect(bar.commands.value.map((one) => one.id)).toStrictEqual([
      `${TIDE}:tree.open`,
      `${REEF}:graph.open`,
    ])
  })
})

describe('the hover text shortcut of a command', () => {
  const withShortcut = entry(`${TIDE}:tree.open`, { shortcut: 'mod+shift+e' })
  const noShortcut = entry(`${TIDE}:chart.open`)
  const undrawable = entry(`${TIDE}:chart.sync`, { shortcut: 'mod+alt+x' })

  it('is the keys the palette draws for the command on this platform, joined with a plus', () => {
    const { bar } = setup({ entries: [withShortcut], agent: LINUX })

    expect(bar.commands.value[0]?.shortcut).toBe('Ctrl+Shift+E')
  })

  it('is written for the Command key on macOS', () => {
    const { bar } = setup({ entries: [withShortcut], agent: MAC })

    expect(bar.commands.value[0]?.shortcut).toMatch(/^(⌘|Cmd|Command)\+Shift\+E$/)
  })

  it('is absent for a command with no shortcut', () => {
    const { bar } = setup({ entries: [noShortcut] })

    expect(bar.commands.value[0]).toBeDefined()
    expect(bar.commands.value[0] && 'shortcut' in bar.commands.value[0]).toBe(false)
  })

  it('is absent for a shortcut the palette cannot draw', () => {
    const { bar } = setup({ entries: [undrawable] })

    expect(bar.commands.value[0]).toBeDefined()
    expect(bar.commands.value[0]?.shortcut).toBeUndefined()
  })
})

describe('the shortcut as the person reads it', () => {
  it('joins the modifier, Shift and the letter with a plus', () => {
    expect(formatShortcut('mod+shift+e', LINUX)).toBe('Ctrl+Shift+E')
    expect(formatShortcut('mod+g', LINUX)).toBe('Ctrl+G')
  })

  it('is nothing where there is none or the palette draws none', () => {
    expect(formatShortcut(undefined, LINUX)).toBeUndefined()
    expect(formatShortcut('', LINUX)).toBeUndefined()
    expect(formatShortcut('mod+f5', LINUX)).toBeUndefined()
    expect(formatShortcut('shift+e', LINUX)).toBeUndefined()
  })
})

describe('the press of a command', () => {
  const tree = entry(`${TIDE}:tree.open`)
  const note = entry('app:note.create')

  it('runs the command with the target in front through the runner of the palette', () => {
    const { bar, runCommand, getTarget } = setup({ entries: [tree, note] })

    bar.runCommand('app:note.create')

    expect(runCommand).toHaveBeenCalledTimes(1)
    expect(runCommand).toHaveBeenCalledWith('app:note.create', target)
    expect(getTarget).toHaveBeenCalled()
  })

  it('passes the full command id of a plugin command as it is', () => {
    const { bar, runCommand } = setup({ entries: [tree] })

    bar.runCommand(`${TIDE}:tree.open`)

    expect(runCommand).toHaveBeenCalledWith(`${TIDE}:tree.open`, target)
  })

  it('does nothing for an id the bar no longer has', () => {
    const { bar, entries, runCommand } = setup({ entries: [tree, note] })
    expect(bar.commands.value).toHaveLength(2)

    entries.value = [note]
    bar.runCommand(`${TIDE}:tree.open`)

    expect(runCommand).not.toHaveBeenCalled()
  })

  it('does nothing for an id the bar never had', () => {
    const { bar, runCommand } = setup({ entries: [tree] })

    bar.runCommand('app:vault.erase')

    expect(runCommand).not.toHaveBeenCalled()
  })

  it('does nothing while no vault is open', () => {
    const { bar, runCommand } = setup({ entries: [tree], isVaultOpen: false })

    bar.runCommand(`${TIDE}:tree.open`)

    expect(runCommand).not.toHaveBeenCalled()
  })

  it('leaves a failure to the runner and does not catch it', () => {
    const { bar, runCommand } = setup({ entries: [tree] })
    runCommand.mockImplementation(() => {
      throw new Error('the chart store is locked')
    })

    expect(() => bar.runCommand(`${TIDE}:tree.open`)).toThrow('the chart store is locked')
  })
})

describe('the commands over the slots of the window when a plugin is disabled and enabled', () => {
  const declare = (pluginId: string, id: string, title: string): DeclaredCommand => ({
    pluginId,
    id,
    title,
    icon: 'waves',
    arguments: [],
  })
  const place = (pluginId: string, command: string, order: number): DeclaredSlot => ({
    pluginId,
    slot: 'titleBar.commands',
    command,
    group: 'navigate',
    order,
  })

  function setupWindow() {
    const tide = declare(TIDE, 'tide.open', 'Open tide chart')
    const reef = declare(REEF, 'reef.open', 'Open reef map')
    const both = {
      commands: [tide, reef],
      placements: [place(TIDE, 'tide.open', 10), place(REEF, 'reef.open', 20)],
    }
    const list = shallowRef<typeof both>(both)
    const slots = createSlotRegistry({
      getEntries: () => readSlotEntries(list.value.placements),
      findCommand: createSlotCommandFinder({
        getPluginCommands: () => [],
        getDeclaredCommands: () => list.value.commands,
        getEditorCommands: () => [],
        getWhenValues: () => ({}),
      }),
      findView: () => undefined,
    })
    const bar = useTitleBarCommands({
      slots,
      isVaultOpen: () => true,
      runCommand: vi.fn(),
      getTarget: () => target,
      agent: LINUX,
    })
    return { bar, list, both, tide, reef }
  }

  it('loses the buttons of a disabled plugin and keeps the order of the others', () => {
    const { bar, list } = setupWindow()
    expect(bar.commands.value.map((one) => one.id)).toStrictEqual([
      `${TIDE}:tide.open`,
      `${REEF}:reef.open`,
    ])

    list.value = {
      commands: [declare(REEF, 'reef.open', 'Open reef map')],
      placements: [place(REEF, 'reef.open', 20)],
    }

    expect(bar.commands.value.map((one) => one.id)).toStrictEqual([`${REEF}:reef.open`])
  })

  it('brings the buttons back at the same positions when the plugin is enabled again', () => {
    const { bar, list, both } = setupWindow()
    const before = bar.commands.value.map((one) => one.id)
    list.value = {
      commands: [both.commands[1]!],
      placements: [both.placements[1]!],
    }

    list.value = both

    expect(bar.commands.value.map((one) => one.id)).toStrictEqual(before)
  })
})
