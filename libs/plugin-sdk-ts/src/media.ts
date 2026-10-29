/**
 * Media and file menu calls over the window's transport.
 */
import { createClient } from '@connectrpc/connect'
import { FileMenuAction, MediaService, TextVersionService } from '@numen/protocol'
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

/** What a file menu entry does when chosen. */
export type FileMenuActionKind = 'createText' | 'deleteVersion'

/** One entry of a file's text-version menu. */
export interface FileMenuEntry {
  readonly action: FileMenuActionKind
  readonly pluginId: string
  readonly versionId: string
  readonly isLive: boolean
}

/** The schema action codes of a file's text-version menu. */
export const FILE_MENU_ACTION = {
  UNSPECIFIED: FileMenuAction.UNSPECIFIED,
  CREATE_TEXT: FileMenuAction.CREATE_TEXT,
  DELETE_VERSION: FileMenuAction.DELETE_VERSION,
} as const

const ACTIONS: ReadonlyMap<number, FileMenuActionKind> = new Map([
  [FileMenuAction.CREATE_TEXT, 'createText'],
  [FileMenuAction.DELETE_VERSION, 'deleteVersion'],
])

/** The action code of the schema an entry's action is. */
export const writeFileMenuAction = (action: FileMenuActionKind): number =>
  [...ACTIONS].find(([, one]) => one === action)?.[0] ?? FileMenuAction.UNSPECIFIED

/** The entries of the schema a view reads, leaving out an action it does not know. */
export const readFileMenuEntries = (
  entries: readonly {
    readonly action: number
    readonly pluginId: string
    readonly versionId: string
    readonly isLive: boolean
  }[],
): readonly FileMenuEntry[] =>
  entries.flatMap((one) => {
    const action = ACTIONS.get(one.action)
    return action
      ? [
          {
            action,
            pluginId: one.pluginId,
            versionId: one.versionId,
            isLive: one.isLive,
          },
        ]
      : []
  })

/** Requests the text-version menu entries of a file from the host. */
export async function getFileMenu(path: string): Promise<readonly FileMenuEntry[]> {
  const answer = await retryWhileBusy(() => textVersionClient.getFileMenu({ path }))
  return readFileMenuEntries(answer.entries)
}

/** Runs one text-version menu entry on a file. */
export async function runFileMenuEntry(path: string, entry: FileMenuEntry): Promise<void> {
  await textVersionClient.runFileMenuEntry({
    path,
    action: writeFileMenuAction(entry.action) as FileMenuAction,
    pluginId: entry.pluginId,
    versionId: entry.versionId,
  })
}
