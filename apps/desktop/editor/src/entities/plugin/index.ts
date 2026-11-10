export type { CatalogEntry } from './lib/catalogEntry'
export { useCatalogEntries } from './model/catalogEntries'
export {
  listDeclaredItemLists,
  listDeclaredCommands,
  runDeclaredCommand,
  runDeclaredHandler,
  type DeclaredCommand,
  type DeclaredCommandOutcome,
  type DeclaredKeybinding,
  type DeclaredList,
  type DeclaredSlot,
  type ItemListAnswer,
  type ItemListRequest,
} from './api/declaredCommands'
export { agentPort, getPluginTexts } from './api/hostClients'
export { readPluginReadme } from './api/readme'
export { PLUGIN_FAILURE_WORDS } from './words'
