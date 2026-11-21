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

const broken = new WeakSet<EditorExtensionEntry>()

function canBuild(entry: EditorExtensionEntry): boolean {
  if (broken.has(entry)) return false
  try {
    EditorState.create({ extensions: entry.extension })
    return true
  } catch (error) {
    broken.add(entry)
    failEntry?.(entry, error)
    return false
  }
}

export function getPluginExtensions(entries: readonly EditorExtensionEntry[]): Extension[] {
  return entries.filter(canBuild).map((entry) => entry.extension)
}
