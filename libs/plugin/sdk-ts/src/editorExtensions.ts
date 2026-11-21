import { Facet, type Extension } from '@codemirror/state'
import type { Caught } from './errors'
import type { Disposable } from './types'

// EditorExtensionRegistration is a CodeMirror extension a plugin adds to the note editor, under the id
// the manifest declares in `contributes.editor.extensions`.
export interface EditorExtensionRegistration {
  readonly id: string
  readonly extension: Extension
}

// EditorExtensionEntry is a registered extension with the plugin that added it.
export interface EditorExtensionEntry extends EditorExtensionRegistration {
  readonly pluginId: string
}

// EditorExtensionHost is how the note editor reads the extensions plugins add.
export interface EditorExtensionHost {
  // The extensions that are on, in the order they were registered.
  get(this: void): readonly EditorExtensionEntry[]
  // Calls the handler whenever an extension is registered, disposed, switched off or on again.
  onChange(this: void, handler: (entries: readonly EditorExtensionEntry[]) => void): Disposable
  // Switches an extension off for the rest of the session after it failed in the editor, and reports
  // the error under the plugin that registered it.
  fail(this: void, entry: EditorExtensionEntry, error: Caught): void
}

// EditorExtensionRegistry keeps the extensions plugins register and tells the editor which are on.
export interface EditorExtensionRegistry extends EditorExtensionHost {
  registerEditorExtension(
    this: void,
    pluginId: string,
    registration: EditorExtensionRegistration,
  ): Disposable
  // Turns an extension of a plugin on or off, as the person chooses. A failed extension stays off.
  setEnabled(this: void, pluginId: string, id: string, isEnabled: boolean): void
}

// editorNotePath is the path of the note an editor holds, which the editor provides to the extensions
// plugins add; it is empty where the editor holds no note of the vault.
export const editorNotePath = Facet.define<string, string>({
  combine: (values) => values[0] ?? '',
})
