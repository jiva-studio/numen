/**
 * What a recording of the vault is, for whatever opens it: its player address,
 * duration, transcript cues, and article prose.
 */
import type { Cue as CueMessage } from '@numen/protocol'
import { retryWhileBusy } from '@/shared/answers'
import { running } from '@/shared/artifacts'
import * as clients from '@/shared/clients'
import type { Cue } from '../lib/cues'
import type { Recordings } from '../types'

const served = {
  recordings: clients.recordings,
  transcripts: clients.transcripts,
  articles: clients.articles,
}

/** Maps a protobuf Cue message to internal Cue type. */
const toCue = (one: CueMessage): Cue => ({ text: one.text, from: one.from, to: one.to })

/**
 * Whether the text may be written over. A run putting the words right holds
 * them while it writes, and what the file carries says so.
 */
const isEditable = async (path: string): Promise<boolean> => {
  const carried = await running.getArtifactStates(path)
  return carried['transcript.corrected'] !== 'running'
}

/**
 * The recordings the vault holds, over the same addresses. The player is given
 * an address of its own: the window is drawn from a scheme a browser does not
 * load sound through, and the application answers where it does.
 */
export const recordings: Recordings = {
  getSummary: async (path) => {
    const answer = await retryWhileBusy(() => served.recordings.getRecording({ path }))
    return {
      duration: answer.durationMs,
      mediaUrl: answer.mediaUrl,
      mediaType: answer.mediaType,
      url: answer.url,
    }
  },
  getTaskStates: (path) => running.getArtifactStates(path),
  readTranscript: async (path) => {
    const answer = await retryWhileBusy(() => served.transcripts.readTranscript({ path }))
    return { cues: answer.cues.map(toCue), prose: '', isEditable: await isEditable(path) }
  },
  readArticle: async (path) => {
    const answer = await retryWhileBusy(() => served.articles.readArticle({ path }))
    return { cues: [], prose: answer.text, isEditable: await isEditable(path) }
  },
  writeTranscript: async (path, cues) => {
    await retryWhileBusy(() => served.transcripts.writeTranscript({ path, cues: [...cues] }))
  },
  findCueTime: async (path, span) => {
    const answer = await retryWhileBusy(() => served.transcripts.readTranscript({ path, span }))
    return answer.cues[0]?.from ?? null
  },
}

