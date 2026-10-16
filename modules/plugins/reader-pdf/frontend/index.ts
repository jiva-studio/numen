import { definePlugin, type PluginContext } from '@numen/plugin-sdk'

export function activate(_context?: PluginContext): void {
  // PDF reader plugin activation
}

export function deactivate(): void {
  // Deactivation hook
}

export default definePlugin({
  activate,
  deactivate,
})
