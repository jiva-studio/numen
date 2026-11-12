/** The strings of a files tab: its name and the actions it offers on a row. */
// Stryker disable StringLiteral: interface copy
export const WORDS = {
  files: 'Files',
  /** The tree, announced to screen readers. */
  tree: 'Vault files',
  empty: 'No files in this vault',
  /** What the tab itself offers in the menu on a row. */
  rename: 'Rename',
  new: 'New',
  newNote: 'Create note',
  newFolder: 'New folder',
  /** The row of a submenu of items while they are read. */
  loadingItems: 'Loading…',
  /** The name a folder is made under, which renaming it is what changes. */
  /** How many rows are being dragged, shown at the pointer while they are. */
  dragging: (files: number) => `${files} files`,
  /** The rows that kept their positions, because the folder holds those names. */
  taken: 'Not moved, names already exist:',
}
