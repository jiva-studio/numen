export { createPluginContext } from './model/context'
export { createPluginHost, createPluginPaneWrapper, type PluginHost } from './model/host'
export {
  clearPluginStyles,
  getPluginStyle,
  registerPluginStyle,
  unregisterPluginStyle,
} from './model/styles'
export type {
  AgentProviderRegistration,
  CommandRegistration,
  Disposable,
  PluginContext,
  PluginContextHost,
  PluginHostDeps,
  PluginLifecycle,
  PluginPaletteCommand,
  ReaderRegistration,
  TabKindRegistration,
} from './model/types'
export {
  canRunOn,
  listDeclaredCommands,
  runDeclaredCommand,
  type DeclaredCommand,
} from './api/declaredCommands'
