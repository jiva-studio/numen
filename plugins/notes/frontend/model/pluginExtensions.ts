/**
 * The CodeMirror extensions other plugins add to the editor, as the host lists them.
 */
import { shallowRef } from 'vue'
import { EditorState, type Extension } from '@codemirror/state'
import type { EditorExtensionEntry, EditorExtensionHost } from '@numen/plugin-sdk'

export const pluginExtensionEntries = shallowRef<readonly EditorExtensionEntry[]>([])

let failEntry: EditorExtensionHost['fail'] | undefined

export function followPluginExtensions(host: EditorExtensionHost): void {
  pluginExtensionEntries.value = host.get()
  failEntry = host.fail
  host.onChange((entries) => {
    pluginExtensionEntries.value = entries
  })
}

const buildable = new WeakMap<object, boolean>()

function canBuild(entry: EditorExtensionEntry): boolean {
  const known = buildable.get(entry.extension)
  if (known !== undefined) return known
  try {
    EditorState.create({ extensions: entry.extension })
    buildable.set(entry.extension, true)
    return true
  } catch (error) {
    buildable.set(entry.extension, false)
    queueMicrotask(() => failEntry?.(entry, error))
    return false
  }
}

export function getPluginExtensions(entries: readonly EditorExtensionEntry[]): Extension[] {
  return entries.filter(canBuild).map((entry) => entry.extension)
}
