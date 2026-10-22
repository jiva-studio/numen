/** What a files tab says: what it is called, and what it offers on a row. */
// Stryker disable StringLiteral: interface copy
export const WORDS = {
  files: 'Files',
  /** The tree, announced to whoever is not looking at it. */
  tree: 'The folders and files of the vault',
  empty: 'This vault holds no files',
  /** What the tab itself offers in the menu on a row. */
  rename: 'Rename',
  newNote: 'New note',
  newDeck: 'New deck',
  newStencil: 'New stencil',
  newPreset: 'New preset',
  newFolder: 'New folder',
  /** The name a folder is made under, which renaming it is what changes. */
  folder: 'New folder',
  /** How many rows are being dragged, shown at the pointer while they are. */
  dragging: (files: number) => `${files} files`,
  /** The rows that stayed where they were, because the folder holds those names. */
  taken: 'These are filed there already:',
}
