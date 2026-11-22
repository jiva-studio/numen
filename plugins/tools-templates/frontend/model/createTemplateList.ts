import { isNotePath, type CommandItem, type VaultEntry } from '@numen/plugin-sdk'
import { createTemplateItems, type FoundTemplate } from '../lib/groupTemplates'
import { couldHold, findGroup, type Mask } from '../lib/masks'

/** What the list reads of the vault. */
export interface TemplateFiles {
  list(folder: string): Promise<readonly VaultEntry[]>
}

/** Lists the notes the masks of the setting name. */
export function createTemplateList(deps: {
  files: TemplateFiles
  getMasks: () => readonly Mask[]
}): {
  listItems(): Promise<readonly CommandItem[]>
} {
  const listFolder = async (folder: string): Promise<readonly VaultEntry[]> => {
    try {
      return await deps.files.list(folder)
    } catch {
      return []
    }
  }

  const findTemplate = (masks: readonly Mask[], entry: VaultEntry): FoundTemplate | undefined => {
    if (!entry.isNote || !isNotePath(entry.path)) return undefined
    const group = findGroup(masks, entry.path)
    return group === undefined ? undefined : { path: entry.path, group }
  }

  return {
    listItems: async () => {
      const masks = deps.getMasks()
      const found: FoundTemplate[] = []
      const pending = ['']
      for (let folder = pending.pop(); folder !== undefined; folder = pending.pop()) {
        for (const entry of await listFolder(folder)) {
          if (entry.isFolder && couldHold(masks, entry.path)) pending.push(entry.path)
          const template = entry.isFolder ? undefined : findTemplate(masks, entry)
          if (template) found.push(template)
        }
      }
      return createTemplateItems(found)
    },
  }
}
