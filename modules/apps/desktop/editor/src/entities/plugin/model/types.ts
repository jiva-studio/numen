/**
 * Types and interfaces for the plugin subsystem.
 */
import type {
  AgentProviderRegistration,
  CommandRegistration,
  Disposable,
  PluginContext,
  PluginLifecycle,
  ReaderRegistration,
  TabKindRegistration,
} from '@numen/plugin-sdk'
import type { AnyTabKind, SourceReader } from '@/entities/tab/@x/plugin'

export type {
  AgentProviderRegistration,
  CommandRegistration,
  Disposable,
  PluginContext,
  PluginLifecycle,
  ReaderRegistration,
  TabKindRegistration,
}

export interface PluginPaletteCommand {
  readonly id: string
  readonly title: string
  readonly shortcut?: string
  isOffered(): boolean
}

export interface PluginHostDeps {
  readonly registerKind: (kind: AnyTabKind) => () => void
  readonly registerExtensionReader: (extension: string, read: SourceReader) => () => void
  readonly openTab: (kind: string, at?: string) => Promise<string>
  readonly getSetting?: (pluginId: string, key: string) => unknown
  readonly onSettingChange?: (
    pluginId: string,
    key: string,
    handler: (value: unknown) => void,
  ) => () => void
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
    handler: (value: T) => void,
  ): Disposable
}
