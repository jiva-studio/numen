/**
 * The commands of the title bar: the entries the plugins place in `titleBar`, drawn as buttons
 * while a vault is open, and run through the runner the palette and the shortcuts use.
 */
import { computed, type Component, type ComputedRef, type Ref } from 'vue'
import { SLOT, type SlotHost, type SlotItem, type TitleBarCommand } from '@numen/plugin-host'
import type { PaletteIcon } from '@numen/ui'
import { findShortcutKeys, type CommandTarget } from '@/features/command-palette'

/** The text of each modifier key in a shortcut hover text. */
const KEY_TEXT: Readonly<Record<PaletteIcon, string>> = {
  control: 'Ctrl',
  shift: 'Shift',
  command: '⌘',
  option: 'Alt',
  return: 'Enter',
}

export interface TitleBarCommandsDeps {
  /** The slots of the window; only `titleBar.commands` is read. */
  readonly slots: Pick<SlotHost, 'getEntries'>
  /** True while a vault is open in this window. */
  readonly isVaultOpen: () => boolean
  /** The runner the palette and the shortcuts use. */
  readonly runCommand: (id: string, target: CommandTarget) => void
  readonly getTarget: () => CommandTarget
  /** `navigator.userAgent`, for the keys of a shortcut. */
  readonly agent: string
}

export interface TitleBarCommands {
  /** The commands in the order the slot gives; empty while no vault is open. */
  readonly commands: ComputedRef<readonly TitleBarCommand[]>
  /** The press of a command by its id (the full command id). */
  runCommand(id: string): void
}

export function useTitleBarCommands(deps: TitleBarCommandsDeps): TitleBarCommands {
  const commands = computed(() =>
    deps.isVaultOpen()
      ? toTitleBarCommands(deps.slots.getEntries(SLOT.titleBarCommands), deps.agent)
      : [],
  )

  const runCommand = (id: string): void => {
    if (!commands.value.some((one) => one.id === id)) return
    deps.runCommand(id, deps.getTarget())
  }

  return { commands, runCommand }
}

/** `mod+shift+e` as the person reads it; undefined where the palette would show none. */
export function formatShortcut(shortcut: string | undefined, agent: string): string | undefined {
  const keys = shortcut ? findShortcutKeys(shortcut, agent) : undefined
  if (!keys) return undefined
  return [...keys.icons.map((icon) => KEY_TEXT[icon]), keys.letter].join('+')
}

/** The buttons of the slot: its commands, each in the group the separators before it count. */
function toTitleBarCommands(items: readonly SlotItem[], agent: string): TitleBarCommand[] {
  let group = 0
  const commands: TitleBarCommand[] = []
  for (const item of items) {
    if (item.kind === 'separator') group += 1
    if (item.kind !== 'command') continue
    const shortcut = formatShortcut(item.shortcut, agent)
    commands.push({
      id: item.command,
      title: item.title,
      icon: item.icon as Component,
      group: String(group),
      ...(shortcut === undefined ? {} : { shortcut }),
    })
  }
  return commands
}

/** The title bar commands of a window over its slots and its runner, shown while a vault is open. */
export function useWindowTitleBar(
  window: Pick<TitleBarCommandsDeps, 'slots' | 'runCommand' | 'getTarget'> & {
    readonly vaultName: Readonly<Ref<string>>
  },
): TitleBarCommands {
  return useTitleBarCommands({
    slots: window.slots,
    isVaultOpen: () => window.vaultName.value !== '',
    runCommand: window.runCommand,
    getTarget: window.getTarget,
    agent: navigator.userAgent,
  })
}
