/** What a files tab says: what it is called, and what it offers on a row. */
export const WORDS = {
  files: 'Files',
  /** The tree, said aloud to whoever is not looking at it. */
  tree: 'The folders and files of the vault',
  /** What stands in place of the tree where the vault holds nothing. */
  empty: 'This vault holds no files',
  /** What the menu on a row offers besides the commands over a note. */
  rename: 'Rename',
  newNote: 'New note',
  newDeck: 'New deck',
  newStencil: 'New stencil',
  newPreset: 'New preset',
  newFolder: 'New folder',
  /** The runs a row can be put through: a recording transcribed, a scan recognised. */
  transcribe: 'Transcribe',
  recognise: 'Recognise',
  /** The runs over a url: the text at its address, and a copy of what is there. */
  downloadText: 'Download the text again',
  downloadCopy: 'Download a copy',
  deleteText: 'Delete the text',
  deleteCopy: 'Delete the copy',
  remove: 'Remove',
  /** The name a folder is made under, which renaming it is what changes. */
  folder: 'New folder',
  /** How many rows are being dragged, said at the pointer while they are. */
  dragging: (files: number) => `${files} files`,
  /** The rows that stayed where they were, because the folder holds those names. */
  taken: 'These are filed there already:',
}
