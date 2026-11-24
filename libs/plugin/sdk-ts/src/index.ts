export type {
  Disposable,
  TextRange,
  ReaderRegistration,
  SettingsViewRegistration,
  SlotViewRegistration,
  AppInfo,
  ReaderHandle,
  ReaderTabState,
  TabTarget,
  TabPlacement,
  TabKindIdentity,
  TabKindPresentation,
  TabKindLifecycle,
  TabKindInput,
  TabKindFiles,
  TabKindRegistration,
  CommandNote,
  CommandRegistration,
  PluginLifecycle,
  VaultChangeEvent,
  FileSelectEvent,
  Subscriptions,
  OpenExternalErrorCode,
  PluginIdentity,
  PluginRegistrations,
  EditorSettings,
  PluginSettings,
  PluginLicenses,
  PluginEvents,
  PluginTabs,
  PluginVault,
  PluginUi,
  PluginWork,
  PluginContext,
  HostServices,
  JsonValue,
  JsonObject,
  WikilinkChoice,
  EditorHandle,
  NoteTabState,
} from './types'
export { definePlugin } from './plugin'
export { getFileName, getFileStem, getRenamedPath, isNotePath, NOTE_TYPE } from './workspace'
export type {
  NoteRef,
  NoteType,
  NoteHeading,
  NoteRelation,
  LinkDirection,
  LinkMarker,
  RelatedNote,
  NoteGraph,
  VaultNotes,
  ConnectionRole,
  LinkRole,
  ConnectionSide,
  Connection,
  RelationJoin,
  RelatedNoteRequest,
  Backlink,
  RelationError,
  VaultRelations,
  TextLink,
  FilePlacement,
  OpenFileOptions,
  MessageWriter,
  WorkspaceWindow,
  WorkspaceDragged,
  WorkspaceFiles,
  WorkspaceMessages,
  WorkspaceIcons,
  WorkspaceHost,
  VaultEntry,
  VaultFiles,
  PathRename,
} from './workspace'
export { CONVERSATION, VAULT_SUMMARY, NOTE_EDITOR } from './views'
export type {
  ViewRegistration,
  ViewContexts,
  ViewCapability,
  MountedView,
  ConversationContext,
  VaultSummaryContext,
  NoteEditorContext,
} from './views'
export { isUnsavedBuffersError, UnsavedBuffersError } from './buffers'
export type { UnsavedBuffer, FileBuffers, BufferHost, FileConflict } from './buffers'
export type {
  Selector,
  TextSegment,
  TextVersionId,
  TextVersion,
  LiveText,
  TextErrorCode,
  VaultTexts,
} from './texts'
export type {
  ConnectionErrorCode,
  ConnectionError,
  ConnectionPreferredRow,
  ConnectionCallOptions,
  ConnectionRowField,
  ConnectionRow,
  ConnectionRowOption,
  ConnectionRowOptionsRequest,
  ConnectionHost,
} from './connections'
export type {
  DialogOptions,
  DialogHandle,
  DialogHost,
  SaveTextOptions,
  SavedText,
} from './dialogs'
export type { Severity, NoticeTask, NoticeHost } from './notices'
export type {
  PickerItem,
  PickerOptions,
  PromptTextOptions,
  ChoiceOption,
  SelectOptionOptions,
  PickerHost,
} from './picker'
export type { PluginTaskStatus, TaskHost } from './tasks'
export type { PluginProcess, LeaseHolder, ProcessHost } from './processes'
export type { LogLevel, LogEntry, LogWrite, LogHost } from './logs'
export type {
  BackgroundTaskState,
  BackgroundTaskProgress,
  BackgroundTaskAllowed,
  BackgroundTask,
  BackgroundTaskHost,
} from './backgroundTasks'
export { SETTINGS_OPEN_COMMAND } from './health'
export type { HealthItem, HealthHost } from './health'
export { isSharedCommand, SHARED_COMMANDS, APP_KEYBINDINGS } from './commands'
export type {
  CommandTarget,
  CommandItem,
  CommandAnswers,
  CommandWarns,
  CommandsHost,
  SharedCommandArgument,
  SharedCommand,
  AppKeybinding,
} from './commands'
export {
  SLOT,
  WHEN_VARIABLES,
  createSlotMenu,
  useSlotMenu,
  MENU_HOLD_MS,
  useSlotLoading,
  createSlotLoading,
  provideSlotLoading,
  SLOT_LOADING_CAP_MS,
  SLOT_LOADING_KEY,
} from './slots'
export type {
  SlotName,
  SlotPlace,
  SlotEntryKey,
  SlotCommand,
  SlotView,
  SlotSubmenu,
  SlotSeparator,
  SlotItem,
  SlotHost,
  CommandItemLists,
  SlotMenuWords,
  SlotMenuPick,
  SlotMenu,
  SlotMenuRequest,
  SlotMenuOptions,
  SlotMenuChoice,
  SlotMenuState,
  SlotLoadingHost,
  SlotLoadingClock,
  SlotLoadingDeps,
  SlotLoading,
  SlotLoadingEntry,
  PlexBoardViewProps,
} from './slots'
export type { VaultFlagsHost } from './vaultFlags'
export type { VaultSummary, VaultsState, VaultsHost, WindowState, WindowHost } from './window'
export type { DroppedData, DropRegistration, DropHost, DropRegistry } from './drops'
export { editorNotePath } from './editorExtensions'
export type {
  EditorExtensionRegistration,
  EditorExtensionEntry,
  EditorExtensionHost,
  EditorExtensionRegistry,
} from './editorExtensions'
export { createApi, UNAVAILABLE } from './api'
export type { Api, ApiError } from './api'
export {
  asFailure,
  asValue,
  formatErrorMessage,
  type ErrorCode,
  type Failure,
  type Result,
} from '@numen/result'
export {
  formatFingerprint,
  getAssetUrl,
  getBytesQuery,
  getRelationErrorWords,
} from './errors'
export { isWebAddress } from './webAddress'
export { resolveAddress, type AddressSource } from './media'
