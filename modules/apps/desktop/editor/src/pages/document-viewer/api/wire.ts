/**
 * Wire adapter for DocumentService and OcrService.
 */
import type { Run as RunMessage } from '@numen/protocol'
import { asset, getBytesQuery, stamp, retryWhileBusy } from '@/shared/answers'
import * as clients from '@/shared/clients'
import type { Documents, PageHighlight, Rect } from '../types'

const served = {
  documents: clients.documents,
  ocr: clients.ocr,
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
    const answer = await retryWhileBusy(() => served.documents.getDocument({ path }))
    return {
      pages: answer.pages.map((one) => ({ width: one.width, height: one.height })),
      fingerprint: stamp(answer.fingerprint) ?? '',
    }
  },
  getPageUrl: (path, pageNumber, wide, fingerprint = '') =>
    `${asset(path)}/pages/${pageNumber}?wide=${wide}&${getBytesQuery(fingerprint)}`,
  getHighlights: async (path, spans) => {
    const answer = await retryWhileBusy(() => served.ocr.readOcr({ path, spans: [...spans] }))
    return spans.map((_, at) => {
      const run = answer.runs[at]
      return run ? toHighlightedPages(run) : []
    })
  },
}

