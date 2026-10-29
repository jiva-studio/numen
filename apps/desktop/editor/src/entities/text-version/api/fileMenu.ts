/**
 * The menu of a file: the commands the enabled plugins declare that apply to
 * the file as it is now, and an entry as the window returns it to be run.
 */
import { textVersions } from '@/shared/clients'

/**
 * One entry of a file's menu. `versionId` is the text version the command runs
 * over, and empty for a command over the file. `title` and `icon` are what the
 * plugin's manifest declares for the command; `icon` is a Lucide icon name.
 */
export interface FileMenuEntry {
  readonly pluginId: string
  readonly pluginName: string
  readonly command: string
  readonly title: string
  readonly icon: string
  readonly versionId: string
}

/** The entries of the menu of the file at the path, as the file is now. */
export async function getFileMenu(path: string): Promise<readonly FileMenuEntry[]> {
  const answer = await textVersions.getFileMenu({ path })
  return answer.entries.map((one) => ({
    pluginId: one.pluginId,
    pluginName: one.pluginName,
    command: one.command,
    title: one.title,
    icon: one.icon,
    versionId: one.versionId,
  }))
}

/** Runs the command of the entry over the file at the path, and returns once the plugin has run it. */
export async function runFileMenuEntry(path: string, entry: FileMenuEntry): Promise<void> {
  await textVersions.runFileMenuEntry({
    path,
    pluginId: entry.pluginId,
    command: entry.command,
    versionId: entry.versionId,
  })
}
