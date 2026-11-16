/**
 * The plugin context of a test as the window gives it for the editor settings: the application's
 * two booleans, absent until the test sets them, and handlers told when they arrive or change.
 * Everything else a plugin reads of its context is left out.
 */
import { vi } from 'vitest'
import type { EditorSettings, PluginContext, ViewRegistration } from '@numen/plugin-sdk'

export function createEditorSettingsContext(initial?: EditorSettings) {
  let held = initial
  const views: ViewRegistration[] = []
  const handlers = new Set<(settings: EditorSettings) => void>()
  const context = {
    register: {
      reader: vi.fn(),
      tabKind: vi.fn(),
      command: vi.fn(),
      view: vi.fn((view: ViewRegistration) => views.push(view)),
    },
    settings: {
      get: () => undefined,
      onChange: () => ({ dispose: () => undefined }),
      getEditor: () => held,
      onEditorChange: (handler: (settings: EditorSettings) => void) => {
        handlers.add(handler)
        return { dispose: () => handlers.delete(handler) }
      },
    },
  } as unknown as PluginContext
  return {
    context,
    /** The views the plugin registered. */
    views,
    /** The window loads its settings, or one of them changes. */
    set: (next: EditorSettings) => {
      held = next
      for (const handler of [...handlers]) handler(next)
    },
  }
}
