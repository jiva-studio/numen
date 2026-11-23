export { createPluginHost, type PluginHost } from './host'
export { requireMountAccess } from './guards/viewPermissions'
export { createCustomTabKind, pickCustomTabParts } from './customTabKind'
export type {
  NoteBuffer,
  PluginContextHost,
  PluginHostDeps,
  HostCommands,
  HostSlots,
  HostKinds,
  HostTabs,
  HostVault,
  HostWork,
  PluginPaletteCommand,
  CommandDeclaration,
  HandlerDeclaration,
} from './types'
export type { AnyTabKind, OpenTab, SourceReader, TabKind } from './tabKinds'
export { createWindowHost, type WindowControls } from './windowHost'
export {
  NO_COMMANDS,
  NO_KINDS,
  NO_PLUGINS,
  NO_SLOTS,
  NO_TABS,
  NO_VAULT,
  NO_WINDOW,
  NO_WORK,
} from './nullDeps'
