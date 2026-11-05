/**
 * The actions of the title bar: the entries the plugins place in `titleBar`, drawn as buttons
 * while a vault is open, and run through the runner the palette and the shortcuts use.
 */
import { computed, type Component, type ComputedRef, type Ref } from 'vue'
import { MENU_LOCATION, type Menus, type PlacedCommand } from '@numen/plugin-host'
import type { PaletteIcon } from '@numen/ui'
import type { TitleBarAction } from '@numen/shell'
import { getNamedIcon } from '@/shared/icons'
import { findShortcutKeys, type CommandTarget } from '@/features/command-palette'

/** The text of each modifier key in a shortcut hover text. */
const KEY_TEXT: Readonly<Record<PaletteIcon, string>> = {
  control: 'Ctrl',
  shift: 'Shift',
  command: '⌘',
  option: 'Alt',
  return: 'Enter',
}

export interface TitleBarActionsDeps {
  /** The places plugins put their commands in; only `titleBar` is read. */
  readonly menus: Pick<Menus, 'getEntries'>
  /** True while a vault is open in this window. */
  readonly isVaultOpen: () => boolean
  /** The runner the palette and the shortcuts use. */
  readonly runCommand: (id: string, target: CommandTarget) => void
  readonly getTarget: () => CommandTarget
  /** Lucide name to component. */
  readonly getIcon: (name: string) => Component
  /** `navigator.userAgent`, for the keys of a shortcut. */
  readonly agent: string
}

export interface TitleBarActions {
  /** The actions in the order the menus give; empty while no vault is open. */
  readonly actions: ComputedRef<readonly TitleBarAction[]>
  /** The press of an action by its id (the full command id). */
  run(id: string): void
}

export function useTitleBarActions(deps: TitleBarActionsDeps): TitleBarActions {
  const toAction = (entry: PlacedCommand): TitleBarAction => {
    const shortcut = formatShortcut(entry.shortcut, deps.agent)
    return {
      id: entry.command,
      title: entry.title,
      icon: deps.getIcon(entry.icon ?? ''),
      ...(entry.group === '' ? {} : { group: entry.group }),
      ...(shortcut === undefined ? {} : { shortcut }),
    }
  }

  const actions = computed(() =>
    deps.isVaultOpen() ? deps.menus.getEntries(MENU_LOCATION.titleBar).map(toAction) : [],
  )

  const run = (id: string): void => {
    if (!actions.value.some((one) => one.id === id)) return
    deps.runCommand(id, deps.getTarget())
  }

  return { actions, run }
}

/** `mod+shift+e` as the person reads it; undefined where the palette would show none. */
export function formatShortcut(shortcut: string | undefined, agent: string): string | undefined {
  const keys = shortcut ? findShortcutKeys(shortcut, agent) : undefined
  if (!keys) return undefined
  return [...keys.icons.map((icon) => KEY_TEXT[icon]), keys.letter].join('+')
}

/** The title bar actions of a window over its menus and its runner, shown while a vault is open. */
export function useWindowTitleBar(
  window: Pick<TitleBarActionsDeps, 'menus' | 'runCommand' | 'getTarget'> & {
    readonly vaultName: Readonly<Ref<string>>
  },
): TitleBarActions {
  return useTitleBarActions({
    menus: window.menus,
    isVaultOpen: () => window.vaultName.value !== '',
    runCommand: window.runCommand,
    getTarget: window.getTarget,
    getIcon: getNamedIcon,
    agent: navigator.userAgent,
  })
}
