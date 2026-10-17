import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
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
}

export default definePlugin({
  activate,
})
