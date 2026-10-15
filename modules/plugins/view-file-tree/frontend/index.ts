import {
  definePlugin,
  type FileSelectEvent,
  type PluginContext,
  type VaultChangeEvent,
} from '@numen/plugin-sdk'
import FileTreeView from './FileTreeView.vue'

export { FileTreeView }
export * from './types'
export * from './words'

export function activate(context: PluginContext): void {
  context.registerTabKind({
    kind: 'file-tree',
    component: FileTreeView,
    title: 'File Tree Explorer',
  })

  context.onVaultChange((_event: VaultChangeEvent) => {
    // Vault change handling
  })

  context.onFileSelect((_event: FileSelectEvent) => {
    // File selection handling
  })
}

export default definePlugin({
  activate,
})
