/**
 * Wire adapter for RecordingService, TextVersionService and the plugin tasks; the transcript is
 * read and edited through the plugin SDK.
 */
import { createClient } from '@connectrpc/connect'
import { RecordingService, FileMenuAction, TextVersionService } from '@numen/protocol'
import type { TaskHost, VaultTexts } from '@numen/plugin-sdk'
import { retryWhileBusy, transport } from '@numen/wire'
import type { MenuAction, MenuEntry, Recordings } from '../types'
import { createWorkWatcher } from './work'
import { createTranscripts } from './transcripts'

export const client = {
  recordings: createClient(RecordingService, transport),
  textVersions: createClient(TextVersionService, transport),
}

let taskHost: TaskHost | undefined

/** Sets the task host the plugin was activated with. */
export const setTaskHost = (host: TaskHost | undefined) => {
  taskHost = host
}

const ACTIONS: ReadonlyMap<FileMenuAction, MenuAction> = new Map([
  [FileMenuAction.RECOGNISE, 'recognise'],
  [FileMenuAction.PROOFREAD, 'proofread'],
  [FileMenuAction.DELETE_VERSION, 'deleteVersion'],
])

/** The action of the schema an entry's action is. */
export const writeMenuAction = (action: MenuAction): FileMenuAction =>
  [...ACTIONS].find(([, one]) => one === action)?.[0] ?? FileMenuAction.UNSPECIFIED

/** The entries of the schema the tab reads, leaving out an action it does not know. */
export const readMenuEntries = (
  entries: readonly {
    action: FileMenuAction
    pluginId: string
    versionId: string
    isLive: boolean
  }[],
): readonly MenuEntry[] =>
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

/** The texts of the vault, as the plugin's context gave them when it was activated. */
let vaultTexts: VaultTexts | undefined

/** Keeps the texts the plugin's context gives, which the transcript is read and edited through. */
export const bindTexts = (texts: VaultTexts | undefined): void => {
  vaultTexts = texts
}

const work = createWorkWatcher((onTasks) => taskHost?.watchAll(onTasks) ?? { dispose: () => {} })

export const recordings: Recordings = {
  getSource: async (path) => {
    const answer = await retryWhileBusy(() => client.recordings.getRecording({ path }))
    return { mediaUrl: answer.mediaUrl, mediaType: answer.mediaType }
  },
  ...createTranscripts(() => vaultTexts),
  getMenu: async (path) => {
    const answer = await retryWhileBusy(() => client.textVersions.getFileMenu({ path }))
    return readMenuEntries(answer.entries)
  },
  runMenuEntry: async (path, entry) => {
    await client.textVersions.runFileMenuEntry({
      path,
      action: writeMenuAction(entry.action),
      pluginId: entry.pluginId,
      versionId: entry.versionId,
    })
  },
  watchWork: (path, onChange) => work.watch(path, onChange),
}
