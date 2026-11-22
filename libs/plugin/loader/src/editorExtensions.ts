/**
 * The CodeMirror extensions plugins add to the note editor. The registry keeps them in the order they
 * were registered and tells the editor which are on; an extension that failed stays off until the
 * window restarts. Disposing a registration removes its extension.
 */
import type { Caught, EditorExtensionEntry, EditorExtensionRegistry } from '@numen/plugin-sdk'

export interface EditorExtensionRegistryDeps {
  readonly reportError?: (pluginId: string, error: Caught) => void
}

const getKey = (pluginId: string, id: string): string => `${pluginId}/${id}`

export function createEditorExtensionRegistry(
  deps: EditorExtensionRegistryDeps = {},
): EditorExtensionRegistry {
  const held: EditorExtensionEntry[] = []
  const failed = new Set<string>()
  const handlers = new Set<(entries: readonly EditorExtensionEntry[]) => void>()

  const get = (): readonly EditorExtensionEntry[] =>
    held.filter((one) => !failed.has(getKey(one.pluginId, one.id)))

  const notify = (): void => {
    const entries = get()
    for (const handler of [...handlers]) handler(entries)
  }

  return {
    get,
    registerEditorExtension: (pluginId, registration) => {
      const key = getKey(pluginId, registration.id)
      if (held.some((one) => getKey(one.pluginId, one.id) === key)) {
        throw new Error(
          `Plugin "${pluginId}" registers the editor extension "${registration.id}" twice`,
        )
      }
      const entry = { pluginId, id: registration.id, extension: registration.extension }
      held.push(entry)
      notify()
      return {
        dispose: () => {
          const index = held.indexOf(entry)
          if (index === -1) return
          held.splice(index, 1)
          notify()
        },
      }
    },
    onChange: (handler) => {
      handlers.add(handler)
      return { dispose: () => handlers.delete(handler) }
    },
    fail: (entry, error) => {
      if (!held.includes(entry)) return
      const key = getKey(entry.pluginId, entry.id)
      if (failed.has(key)) return
      failed.add(key)
      deps.reportError?.(entry.pluginId, error)
      notify()
    },
  }
}
