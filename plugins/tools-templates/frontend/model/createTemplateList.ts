import { isNotePath, type CommandItem, type VaultEntry } from '@numen/plugin-sdk'
import { createTemplateItems, type FoundTemplate } from '../lib/groupTemplates'
import { canHold, findGroup, type Mask } from '../lib/masks'

/** What the list reads of the vault. */
export interface TemplateFiles {
  list(folder: string): Promise<readonly VaultEntry[]>
}

/** A folder of the vault that could not be listed. */
export class TemplateListError extends Error {
  constructor(folder: string, cause: unknown) {
    const reason = cause instanceof Error ? cause.message : String(cause)
    super(`cannot list the folder "${folder}": ${reason}`, { cause })
    this.name = 'TemplateListError'
  }
}

const NOT_FOUND = new Set<string | number>(['not_found', 5])

const isMissing = (error: unknown): boolean => {
  if (typeof error !== 'object' || error === null) return false
  const { code } = error as { code?: string | number }
  return code !== undefined && NOT_FOUND.has(code)
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
    } catch (error) {
      if (isMissing(error)) return []
      throw new TemplateListError(folder, error)
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
          if (entry.isFolder && canHold(masks, entry.path)) pending.push(entry.path)
          const template = entry.isFolder ? undefined : findTemplate(masks, entry)
          if (template) found.push(template)
        }
      }
      return createTemplateItems(found)
    },
  }
}
