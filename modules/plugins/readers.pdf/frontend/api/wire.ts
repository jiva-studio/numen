/**
 * Wire adapter for DocumentService and OcrService.
 */
import { createClient } from '@connectrpc/connect'
import {
  DocumentService,
  OcrService,
  type Box as BoxMessage,
  type Run as RunMessage,
} from '@numen/protocol'
import {
  formatFingerprint,
  getAssetUrl,
  getBytesQuery,
  retryWhileBusy,
  transport,
} from '@numen/wire'
import type { Documents, PageHighlight, Rect } from '../types'

export const client = {
  documents: createClient(DocumentService, transport),
  ocr: createClient(OcrService, transport),
}

/** The corners of one box, in the page's own numbers. */
const toRect = (box: BoxMessage): Rect => ({
  minX: box.rect?.minX ?? 0,
  minY: box.rect?.minY ?? 0,
  maxX: box.rect?.maxX ?? 0,
  maxY: box.rect?.maxY ?? 0,
})

/**
 * Where one run of the text stands, page by page.
 */
const toHighlightedPages = (run: RunMessage): PageHighlight[] => {
  const pages: { page: number; rects: Rect[] }[] = []
  for (const box of run.boxes) {
    const rect = toRect(box)
    const last = pages.at(-1)
    if (last && last.page === box.page) last.rects.push(rect)
    else pages.push({ page: box.page, rects: [rect] })
  }
  return pages
}

export const documents: Documents = {
  getDocumentLayout: async (path) => {
    const answer = await retryWhileBusy(() => client.documents.getDocument({ path }))
    return {
      pages: answer.pages.map((one) => ({ width: one.width, height: one.height })),
      fingerprint: formatFingerprint(answer.fingerprint) ?? '',
    }
  },
  getPageUrl: (...[path, pageNumber, width, fingerprint = '']: [string, number, number, string?]) =>
    `${getAssetUrl(path)}/pages/${pageNumber}?wide=${width}&${getBytesQuery(fingerprint)}`,
  getHighlights: async (path, spans) => {
    const answer = await retryWhileBusy(() => client.ocr.readOcr({ path, spans: [...spans] }))
    return spans.map((_, at) => {
      const run = answer.runs[at]
      return run ? toHighlightedPages(run) : []
    })
  },
}
