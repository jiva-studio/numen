/**
 * The window a plugin runs in, and the vaults it shows: what a bar plugin draws and controls. The
 * platform is `plugin.identity.app.platform`, and the name of the application
 * `plugin.identity.app.name`.
 */
import type { Disposable } from './types'

export interface VaultSummary {
  readonly id: string
  readonly name: string
}

// VaultsState is the vaults the window can show and the one it shows.
export interface VaultsState {
  readonly vaults: readonly VaultSummary[]
  readonly current: VaultSummary | null
}

export interface VaultsHost {
  getState(): VaultsState
  // Calls the handler with the state at once and again on every change.
  watch(handler: (state: VaultsState) => void): Disposable
  // Shows the vault of the id in the window.
  switchTo(id: string): Promise<void>
}

// WindowState is the state of the window a bar draws its buttons by.
export interface WindowState {
  readonly isMaximized: boolean
  readonly isFullscreen: boolean
}

// WindowHost is the window the plugin runs in.
export interface WindowHost {
  getState(): WindowState
  // Calls the handler with the state at once and again on every change.
  watch(handler: (state: WindowState) => void): Disposable
  minimize(): void
  toggleMaximize(): void
  close(): void
}
