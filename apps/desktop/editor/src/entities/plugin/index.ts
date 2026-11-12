export type { CatalogEntry } from './lib/catalogEntry'
export { useCatalogEntries } from './model/catalogEntries'
export {
  listDeclaredItemLists,
  listDeclaredCommands,
  runDeclaredCommand,
  runDeclaredHandler,
  type DeclaredCommand,
  type DeclaredKeybinding,
  type DeclaredSlot,
  type ItemListAnswer,
  type ItemListRequest,
} from './api/declaredCommands'
export { agentPort, getPluginTexts } from './api/hostClients'
export { getFrontendPluginIds } from './lib/frontendPlugins'
export { readPluginReadme } from './api/readme'
export { PLUGIN_FAILURE_WORDS } from './words'
export { getPaletteId } from './lib/paletteId'
export { getNamedTarget, toRunTarget } from './lib/runTarget'
