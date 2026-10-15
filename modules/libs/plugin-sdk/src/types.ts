// Disposable is a handle released on teardown.
export interface Disposable {
  dispose(): void
}

// Result is the discriminated outcome of a fallible operation.
export type Result<T, E = string> =
  | { ok: true; value: T }
  | { ok: false; error: E }

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
}

// TabKindRegistration registers a custom workspace tab.
export interface TabKindRegistration {
  kind: string
  component: unknown
  title?: string
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
