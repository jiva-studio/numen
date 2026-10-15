import { definePlugin, type PluginContext } from '@numen/plugin-sdk'

export function activate(_context?: PluginContext): void {
  // Activation hook
}

export function deactivate(): void {
  // Deactivation hook
}

export default definePlugin({
  activate,
  deactivate,
})
