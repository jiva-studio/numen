/**
 * What a file of the vault is, for whatever opens it: a document's pages, a
 * recording's duration and where its sound is played from, and its
 * transcript.
 *
 * The application serves both over addresses of its own, because the window is
 * drawn from a scheme a browser loads neither pictures nor sound through.
 */
import { Code, createClient } from '@connectrpc/connect'
import type { ConnectError } from '@connectrpc/connect'
import {
  ArticleService,
  DocumentService,
  OcrService,
  RecordingService,
  TranscriptService,
} from '@numen/protocol'
import type { Cue as CueMessage, Run as RunMessage } from '@numen/protocol'
import { transport } from '@numen/wire'
import { fingerprint, stamp } from './shared/answers'
import { running } from './shared/artifacts'
import type { Documents, HighlightedPage, Rect } from './features/document/open'
import type { Recordings } from './features/media/transcript'
import type { Cue } from './features/media/cues'

/** The services the application answers these questions over. */
const served = {
  /** What a document is: how many pages it has and how large each of them is. */
  documents: createClient(DocumentService, transport),
  /** What a recording is: how long it runs and where its bytes are played from. */
  recordings: createClient(RecordingService, transport),
  /** The words of a file that carries the times each stretch was said at. */
  transcripts: createClient(TranscriptService, transport),
  /** The prose a page is written around. */
  articles: createClient(ArticleService, transport),
  /** What a model read off a document's pages, and where each run of it stands. */
  readings: createClient(OcrService, transport),
}

/**
 * The documents the vault holds, over the addresses the application serves the
 * window at. A page is a picture at an address of its own, drawn to the width
 * it is asked for in device pixels.
 */
export const documents: Documents = {
  shape: async (path) => {
    const answer = await waiting(() => served.documents.getDocument({ path }))
    return {
      pages: answer.pages.map((one) => ({ width: one.width, height: one.height })),
      at: stamp(answer.fingerprint) ?? '',
    }
  },
  page: (path, at, wide, seen) =>
    `${asset(path)}/pages/${at}?wide=${wide}&${named(seen)}`,
  highlights: async (path, spans) => {
    const answer = await waiting(() => served.readings.readOcr({ path, spans: [...spans] }))
    return spans.map((_, at) => {
      const run = answer.runs[at]
      return run ? highlighted(run) : []
    })
  },
}

/**
 * Where one run of the text stands, page by page. The boxes come in the order
 * they were read, so a run crossing a page opens a page where it crosses.
 */
const highlighted = (run: RunMessage): HighlightedPage[] => {
  const pages: { page: number; rects: Rect[] }[] = []
  for (const box of run.boxes) {
    const rect: Rect = {
      minX: box.rect?.minX ?? 0,
      minY: box.rect?.minY ?? 0,
      maxX: box.rect?.maxX ?? 0,
      maxY: box.rect?.maxY ?? 0,
    }
    const last = pages.at(-1)
    if (last && last.page === box.page) last.rects.push(rect)
    else pages.push({ page: box.page, rects: [rect] })
  }
  return pages
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
    return { cues: answer.cues.map(spoken), prose: '', editable: await editable(path) }
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

/** One stretch of speech, kept as the plain value the window carries it as. */
const spoken = (one: CueMessage): Cue => ({ text: one.text, from: one.from, to: one.to })

/**
 * Whether the text may be written over. A run putting the words right holds
 * them while it writes, and what the file carries says so.
 */
const editable = async (path: string): Promise<boolean> => {
  const carried = await running.carries(path)
  return carried['transcript.corrected'] !== 'running'
}

/**
 * Where a file of the vault is asked about. The path is written out whole, so a
 * file in a folder is one part of the address and the facet asked of it is the
 * next.
 */
const asset = (path: string): string => `/assets/${encodeURIComponent(path)}`

/**
 * Which bytes an address is about, as the address writes them. The path is a
 * part of the address already, so what is written here is the rest of what says
 * which file it is.
 */
const named = (seen: string): string => {
  const at = fingerprint(seen)
  return `size=${at.size}&mtime=${at.mtime}`
}

/** How often a document that is busy is waited out before it is a refusal. */
const PATIENCE = 3

/** How long the window waits before asking a busy document again. */
const AGAIN = 1000

/**
 * What the application answered. A document held by whoever is drawing from it
 * says so, and is asked again after a wait.
 */
const waiting = async <T>(ask: () => Promise<T>): Promise<T> => {
  for (let asked = 0; ; asked++) {
    try {
      return await ask()
    } catch (error) {
      if (Code.Unavailable !== (error as ConnectError).code || asked >= PATIENCE) throw error
      await sleep(AGAIN)
    }
  }
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))
