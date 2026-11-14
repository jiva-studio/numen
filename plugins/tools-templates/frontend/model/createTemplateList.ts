import type { CommandItem, VaultEntry } from '@numen/plugin-sdk'

/** What the list reads of the vault. */
export interface TemplateFiles {
  list(folder: string): Promise<readonly VaultEntry[]>
}

/** Lists the templates of the folder the setting names. */
export function createTemplateList(_deps: { files: TemplateFiles; getFolder: () => string }): {
  listItems(): Promise<readonly CommandItem[]>
} {
  return { listItems: () => Promise.resolve([{ id: '', title: '' }]) }
}
