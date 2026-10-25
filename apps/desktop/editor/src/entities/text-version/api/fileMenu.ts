/**
 * The menu a file's text versions make: making text of the file, making a
 * version of its live text, and deleting a version, each by the plugin named.
 * The application decides what the menu holds from the enabled plugins and the
 * versions the file has.
 */
import { FileMenuAction } from '@numen/protocol'
import { textVersions } from '@/shared/clients'

/** What an entry of a file's menu does. */
export type FileMenuActionName = 'recognise' | 'proofread' | 'delete-version'

/**
 * One entry of a file's menu. `versionId` is the input of a proofreading and
 * the target of a deletion, and empty for a recognition.
 */
export interface FileMenuEntry {
  readonly action: FileMenuActionName
  readonly pluginId: string
  readonly pluginName: string
  readonly versionId: string
}

const ACTIONS: ReadonlyMap<FileMenuAction, FileMenuActionName> = new Map([
  [FileMenuAction.RECOGNISE, 'recognise'],
  [FileMenuAction.PROOFREAD, 'proofread'],
  [FileMenuAction.DELETE_VERSION, 'delete-version'],
])

/** The entries of the menu of the file at the path, as the file is now. */
export async function getFileMenu(path: string): Promise<readonly FileMenuEntry[]> {
  const answer = await textVersions.getFileMenu({ path })
  return answer.entries.flatMap((one) => {
    const action = ACTIONS.get(one.action)
    if (action === undefined) return []
    return [
      { action, pluginId: one.pluginId, pluginName: one.pluginName, versionId: one.versionId },
    ]
  })
}

/** Does what the entry offers over the file at the path. The plugin goes on after this returns. */
export async function runFileMenuEntry(path: string, entry: FileMenuEntry): Promise<void> {
  const action = [...ACTIONS].find(([, name]) => name === entry.action)?.[0]
  await textVersions.runFileMenuEntry({
    path,
    action: action ?? FileMenuAction.UNSPECIFIED,
    pluginId: entry.pluginId,
    versionId: entry.versionId,
  })
}
