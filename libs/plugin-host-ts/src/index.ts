export {
  DEFAULT_PLUGIN_ENABLED,
  createPluginLoader,
  isPluginEnabledValue,
  type PluginLoader,
  type PluginLoaderDeps,
} from './loader'
export { INTERFACE_WAIT_MS, PLUGIN_START_LIMIT_MS } from './loading/limits'
export type { LoaderHost } from './host'
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
export { createViewRegistry, type ViewRegistry } from './views'
export { createDropRegistry } from './drops'
export {
  createSlotViewRegistry,
  findSlotView,
  getDeclaredSlots,
  type DeclaredSlotView,
  type SlotViewManifest,
  type SlotViewRegistry,
} from './slot-views/slotViews'
export { createAppInfo, DEV_VERSION, type AppInfoSource } from './appInfo'
export {
  createExternalOpener,
  isWebAddress,
  openExternalUrl,
  type ExternalOpener,
} from './externalOpener'
export { createAgentPort, type AgentClient } from './agentPort'
export { resolveTextLinks, type LinkClient } from './textLinks'
export { createPluginTexts } from './pluginTexts'
export { APP_COMMAND_OWNER, formatCommandId, getCommandOwner } from './commandId'
export * from './contract'
export { createPluginNotices, type NoticeList, type NoticeListTask } from './pluginNotices'
export { createPluginTasks, type WindowTask } from './pluginTasks'
export { createPluginHost, createPluginPaneWrapper, type PluginHost } from './hosting/host'
export { createPluginAgent } from './hosting/agentGuard'
export { createPluginContext } from './hosting/context'
export { createCustomTabKind } from './hosting/customTabKind'
export type {
  NoteBuffer,
  PluginContextHost,
  PluginHostDeps,
  PluginPaletteCommand,
  CommandDeclaration,
  HandlerDeclaration,
  BackendHandlerRun,
  TabActionHandler,
} from './hosting/types'
export type { AnyTabKind, OpenTab, SourceReader, TabKind, TabTarget } from './hosting/tabKinds'
export {
  createSlotRegistry,
  readManifestSlotEntries,
  type DeclaredSlotEntry,
  type SlotCommandLook,
  type SlotRegistryDeps,
} from './slots/slotRegistry'
export { parseWhen, type WhenCondition, type WhenSettings, type WhenValues } from './when/when'
export { getTargetValues } from './when/targetValues'
export { createWhenReader, type WhenReader } from './when/whenReader'
export {
  createKeybindingRegistry,
  type Keybinding,
  type KeybindingRegistry,
  type KeybindingRegistryDeps,
  type KeybindingSource,
} from './keybindings/keybindingRegistry'
export { parseKey, type ParsedKey } from './keybindings/parseKey'
export { getWhenValues, type WhenValueParts } from './when/whenValues'
export { createWindowHost, type WindowControls } from './hosting/windowHost'
export { SlotViewHost, usePluginStyles } from './slot-views/slotViewHost'
