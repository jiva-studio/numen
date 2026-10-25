import { buildPluginCatalog, type PluginModuleLoader } from '@/features/plugin-loader'

const manifestFiles = import.meta.glob('../../../../../../plugins/*/manifest.json', {
  eager: true,
  import: 'default',
})
const frontendFiles = import.meta.glob('../../../../../../plugins/*/frontend/index.ts') as Record<
  string,
  PluginModuleLoader
>

/** Every plugin under `modules/plugins`: its manifest, and its frontend where it has one. */
export const pluginCatalog = buildPluginCatalog(manifestFiles, frontendFiles)
