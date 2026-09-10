/**
 * Wire adapter for DocumentService and OcrService.
 */
import { createClient } from '@connectrpc/connect'
import { DocumentService, OcrService } from '@numen/protocol'
import type { Run as RunMessage } from '@numen/protocol'
import { transport } from '@numen/wire'
import { asset, named, stamp, waiting } from '../shared/answers'
import type { Documents, PageHighlight, Rect } from './types'

const served = {
  documents: createClient(DocumentService, transport),
  ocr: createClient(OcrService, transport),
  readings: createClient(OcrService, transport),
}

/**
 * Where one run of the text stands, page by page.
 */
const toHighlightedPages = (run: RunMessage): PageHighlight[] => {
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

export const documents: Documents = {
  getDocumentLayout: async (path) => {
    const answer = await waiting(() => served.documents.getDocument({ path }))
    return {
      pages: answer.pages.map((one) => ({ width: one.width, height: one.height })),
      fingerprint: stamp(answer.fingerprint) ?? '',
    }
  },
  getShape: async (path) => {
    const answer = await waiting(() => served.documents.getDocument({ path }))
    return {
      pages: answer.pages.map((one) => ({ width: one.width, height: one.height })),
      at: stamp(answer.fingerprint) ?? '',
    }
  },
  getPageUrl: (path, at, wide, seen = '') =>
    `${asset(path)}/pages/${at}?wide=${wide}&${named(seen)}`,
  getHighlights: async (path, spans) => {
    const answer = await waiting(() => served.ocr.readOcr({ path, spans: [...spans] }))
    return spans.map((_, at) => {
      const run = answer.runs[at]
      return run ? toHighlightedPages(run) : []
    })
  },
}

