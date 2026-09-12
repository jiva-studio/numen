/** The note tab: the text it edits, the notes it makes, and what it says changed. */
export { default as NoteTab } from './ui/NoteTab.vue'
export { useNoteTab } from './kind'
export { noteChanges } from './model/changes'
export { CREATABLE, noteCreator } from './api/maker'
export type { NoteCreator } from './api/maker'
