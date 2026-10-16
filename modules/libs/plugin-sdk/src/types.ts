// Disposable is a handle released on teardown.
export interface Disposable {
  dispose(): void
}

// Span describes a run of text by start and end boundaries.
export interface Span {
  readonly from: number
  readonly to: number
}

// Result is the discriminated outcome of a fallible operation.
export type Result<T, E = string> = { ok: true; value: T } | { ok: false; error: E }

// SettingOption declares one choice in an enumerated setting.
export interface SettingOption {
  value: string
  label: string
}

// SettingDeclaration declares a plugin configuration option.
export interface SettingDeclaration {
  key: string
  type: 'string' | 'boolean' | 'number' | 'range' | 'enum' | 'select'
  title: string
  description?: string
  default: unknown
  options?: SettingOption[]
  min?: number
  max?: number
  step?: number
}

// ReaderRegistration registers a document viewer component for file extensions.
export interface ReaderRegistration {
  id: string
  fileExtensions: string[]
  component: unknown
  title?: string
  source?: 'note' | 'book' | 'recording' | 'url' | 'other' | string
  capabilities?: readonly string[]
  metadata?: Record<string, unknown>
}

// ReaderPaneProps describes the props passed to a reader component by the host pane wrapper.
export interface ReaderPaneProps {
  readonly path: string
  readonly file?: string
  readonly pluginId?: string
  readonly title?: string
  readonly state?: unknown
}

// TabKindRegistration registers a custom workspace tab.
export interface TabKindRegistration {
  kind: string
  component: unknown
  title?: string
  source?: 'note' | 'book' | 'recording' | 'url' | 'other' | string
  actions?: Record<string, (at: string, ...args: unknown[]) => unknown>
  onClose?: (state: unknown, id: string) => boolean
}

// CommandRegistration registers a palette or keybound action.
export interface CommandRegistration {
  id: string
  title: string
  handler: () => void | Promise<void>
  shortcut?: string
  canShowInPalette?: boolean
}

// AgentProviderRegistration registers an AI agent provider.
export interface AgentProviderRegistration {
  id: string
  name: string
  createAgent: (options?: unknown) => unknown
  capabilities?: string[]
}

// PluginLifecycle declares the activation and teardown hooks.
export interface PluginLifecycle {
  activate(context: PluginContext): void | Promise<void>
  deactivate?(): void | Promise<void>
}

// VaultChangeEvent describes a change in vault root or files.
export interface VaultChangeEvent {
  readonly vaultPath?: string
  readonly paths?: readonly string[]
  readonly renamed?: readonly { readonly from: string; readonly to: string }[]
}

// FileSelectEvent describes a file selection or open request.
export interface FileSelectEvent {
  readonly path: string
  readonly title?: string
}

// PluginContext provides registration and host services to an activated plugin.
export interface PluginContext {
  readonly pluginId: string
  readonly subscriptions: Disposable[]
  registerReader(registration: ReaderRegistration): Disposable
  registerTabKind(registration: TabKindRegistration): Disposable
  registerCommand(registration: CommandRegistration): Disposable
  registerAgentProvider(registration: AgentProviderRegistration): Disposable
  getSetting<T = unknown>(key: string): T | undefined
  onSettingChange<T = unknown>(key: string, handler: (value: T) => void): Disposable
  onVaultChange(handler: (event: VaultChangeEvent) => void | Promise<void>): Disposable
  onFileSelect(handler: (event: FileSelectEvent) => void | Promise<void>): Disposable
  emitVaultChange(event: VaultChangeEvent): void
  emitFileSelect(event: FileSelectEvent): void
  openTab?(kind: string, at?: string): Promise<string> | void
  closeTab?(id: string): void
  hasPermission?(permission: string): boolean
}

// PluginManifest describes the metadata in manifest.json.
export interface PluginManifest {
  id: string
  name: string
  version: string
  description?: string
  capabilities?: string[]
  targets?: string[]
  activationEvents?: string[]
  permissions?: string[]
  lifecycle?: {
    mode: 'on-demand' | 'persistent' | 'manual'
    idleTimeoutSeconds?: number
    keepAliveWhileVisible?: boolean
  }
  backend?: {
    type: 'process' | 'wasm'
    entrypoint: string
    serviceName?: string
  }
  frontend?: {
    entrypoint: string
    styles?: string
  }
  settings?: SettingDeclaration[]
}
