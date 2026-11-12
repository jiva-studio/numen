export { createPluginHost, createPluginPaneWrapper, type PluginHost } from './host'
export { requireMountAccess } from './guards/viewPermissions'
export { createPluginContext } from './context'
export { createIdleContextHost, type IdleContextHostParts } from './idleContextHost'
export { disposeSubscriptions } from './guards/subscriptions'
export { createCustomTabKind } from './customTabKind'
export type {
  NoteBuffer,
  PluginContextHost,
  PluginHostDeps,
  PluginPaletteCommand,
  CommandDeclaration,
  HandlerDeclaration,
  BackendHandlerRun,
  HostCommandReads,
  SlotCommandFinder,
} from './types'
export type { AnyTabKind, OpenTab, SourceReader, TabKind, TabTarget } from './tabKinds'
export { createWindowHost, type WindowControls } from './windowHost'
