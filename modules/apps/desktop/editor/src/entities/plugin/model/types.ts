/**
 * Types and interfaces for the plugin subsystem.
 */
import type {
  AgentProviderRegistration,
  CommandRegistration,
  Disposable,
  FileSelectEvent,
  PluginContext,
  PluginLifecycle,
  ReaderRegistration,
  TabKindRegistration,
  VaultChangeEvent,
} from '@numen/plugin-sdk'
import type { AnyTabKind, SourceReader } from '@/entities/tab/@x/plugin'

export type {
  AgentProviderRegistration,
  CommandRegistration,
  Disposable,
  FileSelectEvent,
  PluginContext,
  PluginLifecycle,
  ReaderRegistration,
  TabKindRegistration,
  VaultChangeEvent,
}

export interface PluginPaletteCommand {
  readonly id: string
  readonly title: string
  readonly shortcut?: string
  isOffered(): boolean
}

export type TabActionHandler = (at: string, ...args: unknown[]) => unknown

export interface PluginHostDeps {
  readonly registerKind: (kind: AnyTabKind) => () => void
  readonly registerExtensionReader: (extension: string, read: SourceReader) => () => void
  readonly registerEditor?: (
    type: string,
    open: (path: string, title?: string, options?: unknown) => void,
  ) => (() => void) | void
  readonly openTab: (kind: string, at?: string) => Promise<string>
  readonly openTabBeside?: (kind: string, at?: string) => Promise<string>
  readonly closeTab?: (id: string) => void
  readonly getTabState?: (id: string) => unknown
  readonly getSetting?: (pluginId: string, key: string) => unknown
  readonly onSettingChange?: (
    pluginId: string,
    key: string,
    handler: (value: unknown) => void | Promise<void>,
  ) => () => void
  readonly closeTabsForPlugin?: (pluginId: string) => void
  readonly onError?: (pluginId: string, error: unknown) => void
  readonly tabActionHandlers?: Record<string, TabActionHandler>
  readonly onTabAction?: (action: string, at: string, ...args: unknown[]) => unknown

  readonly onCustomCardAdded?: (
    at: string,
    values: readonly { field: string; text: string }[],
  ) => void
  readonly readNote?:
    ((path: string) => Promise<{ body?: string; error?: string } | string | undefined>) | undefined
  readonly writeNote?: ((path: string, body: string) => Promise<void>) | undefined
}

export interface PluginContextHost {
  registerReader(pluginId: string, registration: ReaderRegistration): Disposable
  registerTabKind(pluginId: string, registration: TabKindRegistration): Disposable
  registerCommand(pluginId: string, registration: CommandRegistration): Disposable
  registerAgentProvider(pluginId: string, registration: AgentProviderRegistration): Disposable
  getSetting<T = unknown>(pluginId: string, key: string): T | undefined
  onSettingChange<T = unknown>(
    pluginId: string,
    key: string,
    handler: (value: T) => void | Promise<void>,
  ): Disposable
  onVaultChange(
    pluginId: string,
    handler: (event: VaultChangeEvent) => void | Promise<void>): Disposable
  onFileSelect(
    pluginId: string,
    handler: (event: FileSelectEvent) => void | Promise<void>): Disposable
  emitVaultChange(pluginId: string, event: VaultChangeEvent): void
  emitFileSelect(pluginId: string, event: FileSelectEvent): void
  openTab?(kind: string, at?: string): Promise<string>
  closeTab?(id: string): void
}
