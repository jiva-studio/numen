/**
 * What a file of the vault is, for whatever opens it: a document's pages, a
 * recording's length and where its sound is played from, and the words it was
 * heard as.
 *
 * The application serves both over addresses of its own, because the window is
 * drawn from a scheme a browser loads neither pictures nor sound through.
 */
import { Code, createClient } from '@connectrpc/connect'
import type { ConnectError } from '@connectrpc/connect'
import { ArtifactService, AssetService } from '@numen/protocol'
import type {
  Cue as CueMessage,
  Page as PageMessage,
  ReadTranscriptResponse,
} from '@numen/protocol'
import { transport } from '@numen/wire'
import { fingerprint, stamp } from './answers'
import type { Documents, Page } from './document/open'
import type { Cue, Recordings } from './media/transcript'

/** What the files of the vault are, for whatever opens one. */
const assets = createClient(AssetService, transport)

/** What a model has made from the files of the vault. */
const artifacts = createClient(ArtifactService, transport)

/**
 * The documents the vault holds, over the addresses the application serves the
 * window at. A page is a picture at an address of its own, drawn to the width
 * it is asked for in device pixels.
 */
export const documents: Documents = {
  shape: async (path) => {
    const answer = await waiting(() => assets.getDocument({ path }))
    return {
      pages: answer.pages.map((one) => ({ width: one.width, height: one.height })),
      at: stamp(answer.fingerprint) ?? '',
    }
  },
  page: (path, at, wide, seen) =>
    `${asset(path)}/pages/${at}?wide=${wide}&${named(seen)}`,
  highlights: async (path, stretches) => {
    const answer = await waiting(() => assets.listHighlights({ path, at: [...stretches] }))
    return stretches.map((_, i) => answer.runs[i]?.pages.map(highlighted) ?? [])
  },
}

/** One page of a highlight, as the window carries it. */
const highlighted = (one: PageMessage): Page => ({
  page: one.index,
  rects: one.rects.map((box) => ({
    minX: box.minX,
    minY: box.minY,
    maxX: box.maxX,
    maxY: box.maxY,
  })),
})

/**
 * The recordings the vault holds, over the same addresses. The player is given
 * an address of its own: the window is drawn from a scheme a browser does not
 * load sound through, and the application answers where it does.
 */
export const recordings: Recordings = {
  listened: async (path) => {
    const answer = await waiting(() => assets.getRecording({ path }))
    return {
      length: answer.length,
      media: answer.media,
      type: answer.type,
      url: answer.url,
    }
  },
  cues: async (path) => {
    const answer = await waiting(() => artifacts.readTranscript({ path }))
    return { ...said(answer), editable: answer.editable }
  },
  writes: async (path, cues) => {
    await waiting(() => artifacts.writeTranscript({ path, cues: [...cues] }))
  },
  plays: async (path, stretch) => {
    const answer = await waiting(() => artifacts.readTranscript({ path, at: stretch }))
    return said(answer).cues[0]?.from ?? null
  },
}

/** One stretch of speech, kept as the plain value the window carries it as. */
const spoken = (one: CueMessage): Cue => ({ text: one.text, from: one.from, to: one.to })

/**
 * The text of a file in whichever of the two shapes it came in: words against
 * the clock, or prose nothing timed. A file nothing has been read or heard for
 * came in neither.
 */
const said = (answer: ReadTranscriptResponse): { cues: readonly Cue[]; prose: string } => {
  switch (answer.text.case) {
    case 'spoken':
      return { cues: answer.text.value.cues.map(spoken), prose: '' }
    case 'prose':
      return { cues: [], prose: answer.text.value }
    default:
      return { cues: [], prose: '' }
  }
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
