/**
 * Address and file menu calls over the window's transport.
 */
import { createClient } from '@connectrpc/connect'
import { MediaService, TextVersionService } from '@numen/protocol'
import { retryWhileBusy } from './errors/answers'
import { transport } from './transport'

const mediaClient = createClient(MediaService, transport)
const textVersionClient = createClient(TextVersionService, transport)

/** Where the bytes at an address are loaded from, and their content type; both empty where unknown. */
export interface AddressSource {
  readonly url: string
  readonly mediaType: string
}

/**
 * Requests from the host where the file at an address is loaded from: a path of the vault, or
 * `plugin://<plugin id>/<path>` for a file in that plugin's store of the vault.
 */
export async function resolveAddress(address: string): Promise<AddressSource> {
  const answer = await retryWhileBusy(() => mediaClient.resolveAddress({ address }))
  return { url: answer.src, mediaType: answer.mediaType }
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
