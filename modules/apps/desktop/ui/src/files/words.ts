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
  newFolder: 'New folder',
  /** The runs a row can be put through: a recording heard, a scan read. */
  transcribe: 'Transcribe',
  recognise: 'Recognise',
  remove: 'Remove',
  /** The name a folder is made under, which renaming it is what changes. */
  folder: 'New folder',
  /** How many rows are being carried, said at the pointer while they are. */
  carrying: (files: number) => `${files} files`,
  /** The rows that stayed where they were, because the folder holds those names. */
  taken: 'These are filed there already:',
}
