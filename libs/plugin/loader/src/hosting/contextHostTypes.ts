/**
 * The parts a plugin context is made from, by the responsibility of each part.
 */
import type {
  AppInfo,
  BackgroundTaskHost,
  BufferHost,
  CommandRegistration,
  CommandsHost,
  ConnectionHost,
  Disposable,
  DropHost,
  DropRegistration,
  EditorExtensionHost,
  EditorExtensionRegistry,
  EditorSettings,
  FileSelectEvent,
  HealthHost,
  JsonValue,
  LogHost,
  MountedView,
  NoticeHost,
  PickerHost,
  DialogHost,
  ProcessHost,
  ReaderRegistration,
  SettingsViewRegistration,
  SlotHost,
  SlotViewRegistration,
  TabKindRegistration,
  TaskHost,
  VaultChangeEvent,
  VaultFiles,
  VaultNotes,
  VaultRelations,
  VaultsHost,
  VaultTexts,
  WindowHost,
  WorkspaceHost,
} from '@numen/plugin-sdk'
import type { DeclaredViewRegistration, MountAccess } from '../views'

/** The registrations a plugin makes into the registry of the host. */
export interface ContextRegistrationHost {
  readonly registerReader: (pluginId: string, registration: ReaderRegistration) => Disposable
  readonly registerSettingsView: (
    pluginId: string,
    registration: SettingsViewRegistration,
  ) => Disposable
  readonly registerTabKind: (pluginId: string, registration: TabKindRegistration) => Disposable
  readonly registerCommand: (pluginId: string, registration: CommandRegistration) => Disposable
}

/** The views and drop handlers a plugin registers, and the views it mounts. */
export interface ContextViewHost {
  readonly registerSlotView: (
    pluginId: string,
    slot: string,
    registration: SlotViewRegistration,
  ) => Disposable
  readonly registerView: (
    pluginId: string,
    registration: DeclaredViewRegistration,
    permissions?: readonly string[],
  ) => Disposable
  mountView(
    capability: string,
    element: HTMLElement,
    context: object,
    access: MountAccess,
  ): MountedView
  readonly registerDropHandler: (pluginId: string, registration: DropRegistration) => Disposable
  readonly drops: DropHost
  readonly registerEditorExtension: EditorExtensionRegistry['registerEditorExtension']
  readonly editorExtensions: EditorExtensionHost
}

/** The settings and licenses a plugin reads and follows. */
interface ContextLicenseHost {
  readonly hasLicense: (pluginId: string, licenseId: string) => boolean
  readonly getLicenseKey: (pluginId: string, licenseId: string) => string | undefined
  readonly watchLicense: (
    pluginId: string,
    licenseId: string,
    handler: (isActive: boolean) => void | Promise<void>,
  ) => Disposable
}

export interface ContextSettingHost extends ContextLicenseHost {
  readonly getSetting: <T extends JsonValue = JsonValue>(
    pluginId: string,
    key: string,
  ) => T | undefined
  readonly watchSetting: <T extends JsonValue = JsonValue>(
    pluginId: string,
    key: string,
    handler: (value: T) => void | Promise<void>,
  ) => Disposable
  readonly getEditorSettings: () => EditorSettings | undefined
  readonly watchEditorSettings: (
    pluginId: string,
    handler: (settings: EditorSettings) => void | Promise<void>,
  ) => Disposable
}

/** The events of the vault and of the selection that plugins listen to and emit. */
export interface ContextEventHost {
  readonly onVaultChange: (
    pluginId: string,
    handler: (event: VaultChangeEvent) => void | Promise<void>,
  ) => Disposable
  readonly onFileSelect: (
    pluginId: string,
    handler: (event: FileSelectEvent) => void | Promise<void>,
  ) => Disposable
  readonly emitVaultChange: (pluginId: string, event: VaultChangeEvent) => void
  readonly emitFileSelect: (pluginId: string, event: FileSelectEvent) => void
}

/** The tabs a plugin opens, navigates and closes. A window without tabs answers no id. */
interface ContextTabHost {
  readonly openTab: (kind: string, at?: string) => Promise<string> | undefined
  readonly navigateTab: (kind: string, path: string) => Promise<string> | undefined
  readonly closeTab: (id: string) => void
}

/** The services that read and write the vault; a service the window lacks answers undefined. */
interface ContextVaultHost {
  readonly getNotes: () => VaultNotes | undefined
  readonly getRelations: () => VaultRelations | undefined
  readonly getFiles: () => VaultFiles | undefined
  readonly getBuffers: () => BufferHost | undefined
  readonly getVaults: () => VaultsHost | undefined
  readonly getTexts: (pluginId: string) => VaultTexts | undefined
}

/**
 * The window a plugin draws in: its application, its target, how it opens web links, its workspace,
 * commands, frame, notices and slots. A service the window lacks answers undefined.
 */
export interface ContextUiHost {
  readonly app: AppInfo
  readonly target: string
  readonly openExternal: (url: string) => Promise<void>
  readonly getWorkspace: () => WorkspaceHost | undefined
  readonly getCommands: () => CommandsHost | undefined
  readonly commandAccess: {
    readonly isListed: (command: string) => boolean
    readonly getPermission?: (command: string) => string | undefined
  }
  readonly getWindow: () => WindowHost | undefined
  readonly getNotices: (pluginId: string) => NoticeHost
  readonly getPicker: () => PickerHost | undefined
  readonly getDialogs: () => DialogHost | undefined
  readonly getSlots: () => SlotHost
}

/** The background work of the window and the health items of a plugin. A service the window lacks answers undefined. */
export interface ContextWorkHost {
  readonly getTasks: (pluginId: string) => TaskHost | undefined
  readonly getConnections: (pluginId: string) => ConnectionHost | undefined
  readonly getBackgroundTasks: () => BackgroundTaskHost | undefined
  readonly getProcesses: () => ProcessHost | undefined
  readonly getLogs: (pluginId: string) => LogHost | undefined
  readonly getHealth: (pluginId: string) => HealthHost | undefined
}

/** What a plugin context is made from, by the responsibility of each part. */
export interface PluginContextHost {
  readonly registration: ContextRegistrationHost
  readonly views: ContextViewHost
  readonly settings: ContextSettingHost
  readonly events: ContextEventHost
  readonly tabs: ContextTabHost
  readonly vault: ContextVaultHost
  readonly ui: ContextUiHost
  readonly work: ContextWorkHost
}
