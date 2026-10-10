/** The text editor, and the time a recording stands against each of its lines. */
export { default as Editor } from './ui/Editor.vue'
export { timing } from './lib/timing'
export { createWikilinkCompletion, createWikilinkSource } from './lib/wikilinkCompletion'
export type { WikilinkOption, WikilinkCompletionOptions } from './lib/wikilinkCompletion'
export type { Extension, Extension as EditorExtension } from '@codemirror/state'
