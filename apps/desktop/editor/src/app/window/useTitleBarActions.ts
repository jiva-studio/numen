/**
 * The actions of the title bar: the entries the plugins place in `titleBar`, drawn as buttons
 * while a vault is open, and run through the runner the palette and the shortcuts use.
 */
import { computed, type Component, type ComputedRef } from 'vue'
import type { Menus } from '@numen/plugin-host'
import type { TitleBarAction } from '@numen/shell'
import type { CommandTarget } from '@/features/command-palette'

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

export function useTitleBarActions(_deps: TitleBarActionsDeps): TitleBarActions {
  return { actions: computed(() => []), run: () => undefined }
}

/** `mod+shift+e` as the person reads it; undefined where the palette would show none. */
export function formatShortcut(_shortcut: string | undefined, _agent: string): string | undefined {
  return undefined
}
