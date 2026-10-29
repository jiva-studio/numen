/**
 * Media and file menu calls over the window's transport.
 */
import { createClient } from '@connectrpc/connect'
import { MediaService, TextVersionService } from '@numen/protocol'
import { retryWhileBusy } from './errors/answers'
import { transport } from './transport'

const mediaClient = createClient(MediaService, transport)
const textVersionClient = createClient(TextVersionService, transport)

/** Where a media file of the vault is served from, and its content type. */
export interface MediaSource {
  readonly mediaUrl: string
  readonly mediaType: string
}

/** Requests from the host the address and content type under which a vault media file is served. */
export async function getMediaSource(path: string): Promise<MediaSource> {
  const answer = await retryWhileBusy(() => mediaClient.getMediaSource({ path }))
  return { mediaUrl: answer.src, mediaType: answer.mediaType }
}

/** Requests from the host the address at which a plugin's saved web page is served. */
export async function getPageSource(pluginId: string, path: string): Promise<string> {
  const answer = await retryWhileBusy(() => mediaClient.getPageSource({ pluginId, path }))
  return answer.src
}

/**
 * One entry of a file's menu: a command a plugin declares in its manifest, offered over the file
 * or over a text version of it. `versionId` is the version the command runs over, and empty for a
 * command over the file. `icon` is a Lucide icon name. `isLive` marks the version that is displayed
 * and indexed.
 */
export interface FileMenuEntry {
  readonly pluginId: string
  readonly pluginName: string
  readonly command: string
  readonly title: string
  readonly icon: string
  readonly versionId: string
  readonly isLive: boolean
}

/** Requests from the host the entries of the menu of a file. */
export async function getFileMenu(path: string): Promise<readonly FileMenuEntry[]> {
  const answer = await retryWhileBusy(() => textVersionClient.getFileMenu({ path }))
  return answer.entries.map((one) => ({
    pluginId: one.pluginId,
    pluginName: one.pluginName,
    command: one.command,
    title: one.title,
    icon: one.icon,
    versionId: one.versionId,
    isLive: one.isLive,
  }))
}

/** Runs the command of one entry of the menu of a file. */
export async function runFileMenuEntry(path: string, entry: FileMenuEntry): Promise<void> {
  await textVersionClient.runFileMenuEntry({
    path,
    pluginId: entry.pluginId,
    command: entry.command,
    versionId: entry.versionId,
  })
}
