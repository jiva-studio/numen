/**
 * What a recording of the vault is, for whatever opens it: its player address,
 * duration, transcript cues, and article prose.
 */
import { createClient } from '@connectrpc/connect'
import { ArticleService, RecordingService, TranscriptService } from '@numen/protocol'
import type { Cue as CueMessage } from '@numen/protocol'
import { transport } from '@numen/wire'
import { waiting } from '../answers'
import { running } from '../artifacts'
import type { Cue } from './cues'
import type { Recordings } from './transcript'

const served = {
  recordings: createClient(RecordingService, transport),
  transcripts: createClient(TranscriptService, transport),
  articles: createClient(ArticleService, transport),
}

/** One stretch of speech, kept as the plain value the window carries it as. */
const heard = (one: CueMessage): Cue => ({ text: one.text, from: one.from, to: one.to })

/**
 * Whether the text may be written over. A run putting the words right holds
 * them while it writes, and what the file carries says so.
 */
const editable = async (path: string): Promise<boolean> => {
  const carried = await running.carries(path)
  return carried['transcript.corrected'] !== 'running'
}

/**
 * The recordings the vault holds, over the same addresses. The player is given
 * an address of its own: the window is drawn from a scheme a browser does not
 * load sound through, and the application answers where it does.
 */
export const recordings: Recordings = {
  listened: async (path) => {
    const answer = await waiting(() => served.recordings.getRecording({ path }))
    return {
      duration: answer.durationMs,
      mediaUrl: answer.mediaUrl,
      mediaType: answer.mediaType,
      url: answer.url,
    }
  },
  carries: (path) => running.carries(path),
  transcript: async (path) => {
    const answer = await waiting(() => served.transcripts.readTranscript({ path }))
    return { cues: answer.cues.map(heard), prose: '', editable: await editable(path) }
  },
  article: async (path) => {
    const answer = await waiting(() => served.articles.readArticle({ path }))
    return { cues: [], prose: answer.text, editable: await editable(path) }
  },
  writes: async (path, cues) => {
    await waiting(() => served.transcripts.writeTranscript({ path, cues: [...cues] }))
  },
  plays: async (path, span) => {
    const answer = await waiting(() => served.transcripts.readTranscript({ path, span }))
    return answer.cues[0]?.from ?? null
  },
}
