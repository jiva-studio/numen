import { isNotePath, type CommandItem, type VaultEntry } from '@numen/plugin-sdk'
import { createTemplateItems, type FoundTemplate } from '../lib/groupTemplates'

/** What the list reads of the vault. */
export interface TemplateFiles {
  list(folder: string): Promise<readonly VaultEntry[]>
}

/** Lists the templates of the folder the setting names. */
export function createTemplateList(deps: { files: TemplateFiles; getFolder: () => string }): {
  listItems(): Promise<readonly CommandItem[]>
} {
  const listFolder = async (folder: string): Promise<readonly VaultEntry[]> => {
    try {
      return await deps.files.list(folder)
    } catch {
      return []
    }
  }

  return {
    listItems: async () => {
      const root = deps.getFolder()
      const found: FoundTemplate[] = []
      const visited = new Set<string>()
      const pending = [root]
      for (let folder = pending.pop(); folder !== undefined; folder = pending.pop()) {
        if (visited.has(folder)) continue
        visited.add(folder)
        for (const entry of await listFolder(folder)) {
          if (entry.isFolder) pending.push(entry.path)
          else if (entry.isNote && isNotePath(entry.path)) found.push({ path: entry.path })
        }
      }
      return createTemplateItems(root, found)
    },
  }
}
