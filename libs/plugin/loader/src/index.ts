export {
  DEFAULT_PLUGIN_ENABLED,
  createPluginLoader,
  isPluginEnabledValue,
  type PluginLoader,
  type PluginLoaderDeps,
} from './loader'
export {
  createPluginStatus,
  INTERFACE_WAIT_MS,
  PLUGIN_START_LIMIT_MS,
  waitWithinInterfaceLimit,
  type ContributionMatch,
  type PluginStatus,
} from './loading'
export type { LoaderHost, PluginParts } from './host'
export type { Manifest } from '@numen/core-client'
export {
  adoptPluginStyle,
  adoptSheet,
  clearPluginStyles,
  createSheet,
  getPluginSheet,
  getPluginStyle,
  loadPluginStyle,
  registerPluginStyle,
  unregisterPluginStyle,
  type StyleRoot,
} from './styles'
export { createDropRegistry } from './drops'
export {
  findSlotView,
  getDeclaredSlots,
  SlotViewHost,
  usePluginStyles,
  type DeclaredSlotView,
  type SlotViewManifest,
} from './slot-views'
export { createAppInfo, DEV_VERSION, type AppInfoSource } from './appInfo'
export { createExternalOpener, isWebAddress, type ExternalOpener } from './externalOpener'
export { resolveTextLinks, type LinkClient } from './textLinks'
export { createPluginConnections } from './pluginConnections'
export { createPluginTexts } from './pluginTexts'
export { APP_COMMAND_OWNER, formatCommandId, getCommandOwner } from './commandId'
export * from './contract'
export { createPluginNotices, type NoticeList, type NoticeListTask } from './pluginNotices'
export { createPluginTasks, type WindowTask } from './pluginTasks'
export type { HealthReports } from './pluginHealth'
export {
  createCustomTabKind,
  createPluginHost,
  createPluginPaneWrapper,
  createWindowHost,
  requireMountAccess,
  type AnyTabKind,
  type BackendHandlerRun,
  type CommandDeclaration,
  type HandlerDeclaration,
  type HostCommandReads,
  type NoteBuffer,
  type OpenTab,
  type PluginContextHost,
  type PluginHost,
  type PluginHostDeps,
  type HostWindow,
  type HostPlugins,
  type HostCommands,
  type HostSlots,
  type HostKinds,
  type HostTabs,
  type HostVault,
  type HostWork,
  type PluginPaletteCommand,
  type SlotCommandFinder,
  type SourceReader,
  type TabKind,
  type TabTarget,
  type WindowControls,
} from './hosting'
export {
  createSlotRegistry,
  readManifestSlotEntries,
  type DeclaredSlotEntry,
  type SlotCommandLook,
  type SlotRegistryDeps,
} from './slots'
export {
  createWhenReader,
  getTargetValues,
  getWhenValues,
  parseWhen,
  type WhenCondition,
  type WhenReader,
  type WhenSettings,
  type WhenValueParts,
  type WhenValues,
} from './when'
export {
  createKeybindingRegistry,
  parseKey,
  type ParsedKey,
  type Keybinding,
  type KeybindingRegistry,
  type KeybindingRegistryDeps,
  type KeybindingSource,
} from './keybindings'
