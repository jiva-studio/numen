/**
 * What a document of the vault is, for whatever opens it: its pages and how
 * large each of them is, its pictures at device resolution, and what a model
 * read off them.
 */
import { createClient } from '@connectrpc/connect'
import { DocumentService, OcrService } from '@numen/protocol'
import type { Run as RunMessage } from '@numen/protocol'
import { transport } from '@numen/wire'
import { asset, named, stamp, waiting } from '../shared/answers'
import type { Documents, HighlightedPage, Rect } from './open'

const served = {
  documents: createClient(DocumentService, transport),
  readings: createClient(OcrService, transport),
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
