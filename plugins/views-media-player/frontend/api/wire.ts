/**
 * Wire adapter for RecordingService, TranscriptService, TextMenuService and the window's tasks.
 */
import { createClient } from '@connectrpc/connect'
import {
  RecordingService,
  TextMenuAction,
  TextMenuService,
  TranscriptService,
  WindowService,
  type Cue as CueMessage,
} from '@numen/protocol'
import { retryWhileBusy, transport } from '@numen/wire'
import type { Cue } from '../lib/cues'
import type { MenuAction, MenuEntry, Recordings } from '../types'
import { createWorkWatcher } from './work'

export const client = {
  recordings: createClient(RecordingService, transport),
  transcripts: createClient(TranscriptService, transport),
  textMenu: createClient(TextMenuService, transport),
  window: createClient(WindowService, transport),
}

/** The window the player's tabs are drawn in. */
const WINDOW = 'editor'

const ACTIONS: ReadonlyMap<TextMenuAction, MenuAction> = new Map([
  [TextMenuAction.RECOGNISE, 'recognise'],
  [TextMenuAction.PROOFREAD, 'proofread'],
  [TextMenuAction.DELETE_VERSION, 'deleteVersion'],
])

/** The action of the schema an entry's action is. */
export const writeMenuAction = (action: MenuAction): TextMenuAction =>
  [...ACTIONS].find(([, one]) => one === action)?.[0] ?? TextMenuAction.UNSPECIFIED

/** The entries of the schema the tab reads, leaving out an action it does not know. */
export const readMenuEntries = (
  entries: readonly {
    action: TextMenuAction
    pluginId: string
    versionId: string
    isFromFile: boolean
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
            isFromFile: one.isFromFile,
            isLive: one.isLive,
          },
        ]
      : []
  })

const readCue = (one: CueMessage): Cue => ({ text: one.text, from: one.from, to: one.to })

const work = createWorkWatcher(async function* (signal) {
  for await (const update of client.window.watchTasks({ window: WINDOW }, { signal })) {
    yield update.tasks
  }
})

export const recordings: Recordings = {
  getSource: async (path) => {
    const answer = await retryWhileBusy(() => client.recordings.getRecording({ path }))
    return { mediaUrl: answer.mediaUrl, mediaType: answer.mediaType }
  },
  readTranscript: async (path) => {
    const answer = await retryWhileBusy(() => client.transcripts.readTranscript({ path }))
    return answer.cues.map(readCue)
  },
  writeTranscript: async (path, cues) => {
    await retryWhileBusy(() => client.transcripts.writeTranscript({ path, cues: [...cues] }))
  },
  findCueTime: async (path, span) => {
    const answer = await retryWhileBusy(() => client.transcripts.readTranscript({ path, span }))
    return answer.cues[0]?.from ?? null
  },
  getMenu: async (path) => {
    const answer = await retryWhileBusy(() => client.textMenu.getTextMenu({ path }))
    return readMenuEntries(answer.entries)
  },
  runMenuEntry: async (path, entry) => {
    await client.textMenu.runTextMenuEntry({
      path,
      entry: {
        action: writeMenuAction(entry.action),
        pluginId: entry.pluginId,
        versionId: entry.versionId,
        isFromFile: entry.isFromFile,
        isLive: entry.isLive,
      },
    })
  },
  watchWork: (path, onChange) => work.watch(path, onChange),
}
