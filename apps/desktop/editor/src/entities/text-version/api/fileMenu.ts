/**
 * The menu a file's text versions make: the entry each enabled plugin declares
 * for files or texts it takes, and deleting a version by the plugin that made
 * it. The application decides what the menu holds from the enabled plugins and
 * the versions the file has.
 */
import { FILE_MENU_ACTION } from '@numen/plugin-sdk'
import { textVersions } from '@/shared/clients'

/** What an entry of a file's menu does: a plugin makes text, or deletes a version it made. */
export type FileMenuActionName = 'create-text' | 'delete-version'

/**
 * One entry of a file's menu. `versionId` is the text a plugin makes text from
 * and the target of a deletion, and empty for text made from the file. `title`
 * and `icon` are what the plugin's manifest declares for its entry, and empty
 * for a deletion; `icon` is a Lucide icon name.
 */
export interface FileMenuEntry {
  readonly action: FileMenuActionName
  readonly pluginId: string
  readonly pluginName: string
  readonly title: string
  readonly icon: string
  readonly versionId: string
}

type FileMenuActionCode = (typeof FILE_MENU_ACTION)[keyof typeof FILE_MENU_ACTION]

const ACTIONS: ReadonlyMap<FileMenuActionCode, FileMenuActionName> = new Map([
  [FILE_MENU_ACTION.CREATE_TEXT, 'create-text'],
  [FILE_MENU_ACTION.DELETE_VERSION, 'delete-version'],
])

/** The entries of the menu of the file at the path, as the file is now. */
export async function getFileMenu(path: string): Promise<readonly FileMenuEntry[]> {
  const answer = await textVersions.getFileMenu({ path })
  return answer.entries.flatMap((one) => {
    const action = ACTIONS.get(one.action)
    if (action === undefined) return []
    return [
      {
        action,
        pluginId: one.pluginId,
        pluginName: one.pluginName,
        title: one.title,
        icon: one.icon,
        versionId: one.versionId,
      },
    ]
  })
}

/** Does what the entry offers over the file at the path. The plugin goes on after this returns. */
export async function runFileMenuEntry(path: string, entry: FileMenuEntry): Promise<void> {
  const action = [...ACTIONS].find(([, name]) => name === entry.action)?.[0]
  await textVersions.runFileMenuEntry({
    path,
    action: action ?? FILE_MENU_ACTION.UNSPECIFIED,
    pluginId: entry.pluginId,
    versionId: entry.versionId,
  })
}
