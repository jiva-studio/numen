/**
 * Types and interfaces for the plugin subsystem.
 */
import type {
  AgentHost,
  AgentProviderRegistration,
  BufferHost,
  CommandRegistration,
  ContextMenuRegistration,
  ContextMenuTarget,
  Disposable,
  DropHost,
  DropRegistration,
  FileSelectEvent,
  MountedView,
  NoticeHost,
  PluginContext,
  PluginLifecycle,
  ReaderRegistration,
  SettingsViewRegistration,
  TabKindRegistration,
  TaskHost,
  VaultChangeEvent,
  ViewRegistration,
  VaultFiles,
  VaultNotes,
  VaultRelations,
  VaultTexts,
  WorkspaceHost,
} from '@numen/plugin-sdk'
import type { AnyTabKind, SourceReader } from '@/shared/model/tabKinds'
import type { Component } from 'vue'

export type {
  AgentProviderRegistration,
  CommandRegistration,
  ContextMenuRegistration,
  ContextMenuTarget,
  Disposable,
  FileSelectEvent,
  PluginContext,
  PluginLifecycle,
  ReaderRegistration,
  SettingsViewRegistration,
  TabKindRegistration,
  TaskHost,
  VaultChangeEvent,
}

export interface PluginPaletteCommand {
  readonly id: string
  readonly title: string
  readonly shortcut?: string
  /** 'note' where it is offered only while a note is in front. */
  readonly scope?: 'note'
  /** The icon it is drawn with. */
  readonly icon?: unknown
  /** Whether it presents a note. */
  readonly isNoteDestination?: boolean
}

export type TabActionHandler = (targetPath: string, ...args: unknown[]) => unknown

export interface PluginHostDeps {
  readonly registerKind: (kind: AnyTabKind) => () => void
  readonly registerExtensionReader: (
    extension: string,
    readItem: SourceReader,
    icon?: Component,
  ) => () => void
  /** The extensions of the files the reader of the plugin opens, as the manifest and the core choose. */
  readonly getReaderExtensions?: (pluginId: string, readerId: string) => readonly string[]
  readonly registerEditor?: (
    type: string,
    open: (path: string, title?: string, options?: unknown) => void,
  ) => (() => void) | void
  readonly openTab: (kind: string, at?: string) => Promise<string>
  readonly openTabAdjacent?: (kind: string, at?: string) => Promise<string>
  readonly navigateTab?: (kind: string, path: string) => Promise<string>
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
  readonly onTabAction?: (action: string, targetPath: string, ...args: unknown[]) => unknown

  readonly readNote?:
    ((path: string) => Promise<{ body?: string; error?: string } | string | undefined>) | undefined
  readonly writeNote?: ((path: string, body: string) => Promise<void>) | undefined
  /** The window a plugin's tabs are drawn in, once the window has made it. */
  readonly getWorkspace?: () => WorkspaceHost | undefined
  /** The notes of the vault, once the window has made them. */
  readonly getNotes?: () => VaultNotes | undefined
  /** The relationships between notes, once the window has made them. */
  readonly getRelations?: () => VaultRelations | undefined
  /** The files of the vault, once the window has made them. */
  readonly getFiles?: () => VaultFiles | undefined
  /** The agent the window queries, once the window has created it. */
  readonly getAgent?: () => AgentHost | undefined
  /** The open files of the window, once the window has made them. */
  readonly getBuffers?: () => BufferHost | undefined
  /** The text versions of the shown vault, as the views of one plugin read and edit them. */
  readonly getTexts?: (pluginId: string) => VaultTexts
  /** The notices one plugin shows in the window's corner. */
  readonly getNotices?: (pluginId: string) => NoticeHost
  /** The tasks the plugins report, as one plugin watches them. */
  readonly getTasks?: (pluginId: string) => TaskHost | undefined
  /** Everything a plugin showed, taken away with the plugin. */
  readonly clearNotices?: (pluginId: string) => void
}

export interface PluginContextHost {
  registerReader(pluginId: string, registration: ReaderRegistration): Disposable
  registerSettingsView(pluginId: string, registration: SettingsViewRegistration): Disposable
  registerTabKind(pluginId: string, registration: TabKindRegistration): Disposable
  registerCommand(pluginId: string, registration: CommandRegistration): Disposable
  registerContextMenu(pluginId: string, registration: ContextMenuRegistration): Disposable
  registerAgentProvider(pluginId: string, registration: AgentProviderRegistration): Disposable
  registerView(pluginId: string, registration: ViewRegistration): Disposable
  mountView(capability: string, element: HTMLElement, context: unknown): MountedView
  registerDropHandler(pluginId: string, registration: DropRegistration): Disposable
  readonly drops: DropHost
  getSetting<T = unknown>(pluginId: string, key: string): T | undefined
  onSettingChange<T = unknown>(
    pluginId: string,
    key: string,
    handler: (value: T) => void | Promise<void>,
  ): Disposable
  onVaultChange(
    pluginId: string,
    handler: (event: VaultChangeEvent) => void | Promise<void>,
  ): Disposable
  onFileSelect(
    pluginId: string,
    handler: (event: FileSelectEvent) => void | Promise<void>,
  ): Disposable
  emitVaultChange(pluginId: string, event: VaultChangeEvent): void
  emitFileSelect(pluginId: string, event: FileSelectEvent): void
  openTab?(kind: string, at?: string): Promise<string>
  navigateTab?(kind: string, path: string): Promise<string> | undefined
  closeTab?(id: string): void
  getWorkspace?(): WorkspaceHost | undefined
  getNotes?(): VaultNotes | undefined
  getRelations?(): VaultRelations | undefined
  getFiles?(): VaultFiles | undefined
  getAgent?(): AgentHost | undefined
  getBuffers?(): BufferHost | undefined
  getNotices?(pluginId: string): NoticeHost | undefined
  getTasks?(pluginId: string): TaskHost | undefined
  getTexts?(pluginId: string): VaultTexts | undefined
}
