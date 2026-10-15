import { definePlugin, type Disposable, type PluginContext } from '@numen/plugin-sdk'
import FileTreeView from './FileTreeView.vue'

export { FileTreeView }
export * from './types'
export * from './words'

export function activate(context: PluginContext): Disposable {
  const tabRegistration = context.registerTabKind({
    kind: 'file-tree',
    component: FileTreeView,
    title: 'File Tree Explorer',
  })

  const vaultSubscription = context.onVaultChange((_event) => {
    // Vault change handling
  })

  const fileSubscription = context.onFileSelect((_event) => {
    // File selection handling
  })

  return {
    dispose: () => {
      tabRegistration.dispose()
      vaultSubscription.dispose()
      fileSubscription.dispose()
    },
  }
}

export default definePlugin({
  activate,
})
