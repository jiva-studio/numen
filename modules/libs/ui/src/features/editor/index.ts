/** The text editor, and the time a recording stands against each of its lines. */
export { default as Editor } from './ui/Editor.vue'
export { createTimeGutter } from './lib/extensions/timeGutter'
export { createWikilinkCompletion } from './lib/extensions/wikilinkCompletion'
export type { WikilinkOption, WikilinkCompletionOptions } from './lib/extensions/wikilinkCompletion'
export type { Extension } from '@codemirror/state'
