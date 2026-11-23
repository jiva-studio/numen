export { createPluginHost } from './host'
export type { PluginHost } from './hostApi'
export { requireMountAccess } from './guards/viewPermissions'
export { createCustomTabKind, pickCustomTabParts } from './customTabKind'
export type {
  NoteBuffer,
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
export type { AnyTabKind, SourceReader, TabKind } from './tabKinds'
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
