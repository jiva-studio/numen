/** What a files tab says: what it is called, and what it offers on a row. */
// Stryker disable StringLiteral: interface copy
export const WORDS = {
  files: 'Files',
  /** The commands the plugin declares: the tree put in front, and a note shown in it. */
  showFiles: 'Show the files of the vault',
  reveal: 'Show this note in the files',
  /** The tree, announced to whoever is not looking at it. */
  tree: 'The folders and files of the vault',
  empty: 'This vault holds no files',
  /** What the tab itself offers in the menu on a row. */
  rename: 'Rename',
  new: 'New',
  newNote: 'New note',
  newFolder: 'New folder',
  /** The name a folder is made under, which renaming it is what changes. */
  folder: 'New folder',
  /** How many rows are being dragged, shown at the pointer while they are. */
  dragging: (files: number) => `${files} files`,
  /** The rows that stayed where they were, because the folder holds those names. */
  taken: 'These are filed there already:',
}
