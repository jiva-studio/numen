/**
 * What the bar draws, read from the host services: the entries of its two slots, the ids of
 * the commands the window runs, the vaults of the window and the state of the window, each followed
 * until the bar is removed. A window that offers none of them gives empty slots, no commands, no
 * vaults and a window that is not maximized.
 */
import { onScopeDispose, shallowRef } from 'vue'
import {
  SLOT,
  type Disposable,
  type HostServices,
  type SlotItem,
  type VaultsState,
  type WindowState,
} from '@numen/plugin-sdk'

const NO_VAULTS: VaultsState = { vaults: [], current: null }
const RESTORED: WindowState = { isMaximized: false, isFullscreen: false }

export function useTitleBarHosts(host: Pick<HostServices, 'ui' | 'vault'>) {
  const commands = shallowRef<readonly SlotItem[]>([])
  const status = shallowRef<readonly SlotItem[]>([])
  const vaults = shallowRef<VaultsState>(NO_VAULTS)
  const window = shallowRef<WindowState>(RESTORED)
  const commandIds = shallowRef<readonly string[]>([])

  const watches: Disposable[] = []
  const { slots } = host.ui
  watches.push(slots.watchEntries(SLOT.titleBarCommands, (next) => (commands.value = next)))
  watches.push(slots.watchEntries(SLOT.titleBarStatus, (next) => (status.value = next)))
  if (host.ui.commands) watches.push(host.ui.commands.watch((next) => (commandIds.value = next)))
  if (host.vault.vaults) watches.push(host.vault.vaults.watch((next) => (vaults.value = next)))
  if (host.ui.window) watches.push(host.ui.window.watch((next) => (window.value = next)))
  onScopeDispose(() => {
    for (const one of watches.splice(0)) one.dispose()
  })

  return { commands, status, vaults, window, commandIds }
}
