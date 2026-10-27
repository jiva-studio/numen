/**
 * Wire adapter for DocumentService.
 */
import { createClient } from '@connectrpc/connect'
import {
  DocumentService,
  type DocumentRun as DocumentRunMessage,
  type DocumentRect as DocumentRectMessage,
} from '@numen/protocol'
import { formatFingerprint, getBytesQuery, retryWhileBusy, transport } from '@numen/wire'
import type { Documents, PageHighlight, Rect } from '../types'

/** Where the plugin's backend answers the window: the core forwards /plugin/<id>/ to it. */
const PLUGIN_ROUTE = '/plugin/md.numen.readers.pdf'

export const client = {
  documents: createClient(DocumentService, transport),
}

/** The corners of one rectangle, in fractions of its page. */
const toRect = (rect: DocumentRectMessage): Rect => ({
  minX: rect.minX,
  minY: rect.minY,
  maxX: rect.maxX,
  maxY: rect.maxY,
})

/** Where one run of the text stands, page by page. */
const toHighlightedPages = (run: DocumentRunMessage): PageHighlight[] =>
  run.places.map((place) => ({ page: place.page, rects: place.rects.map(toRect) }))

export const documents: Documents = {
  getDocumentLayout: async (path) => {
    const answer = await retryWhileBusy(() => client.documents.getDocument({ path }))
    return {
      pages: answer.pages.map((one) => ({ width: one.width, height: one.height })),
      fingerprint: formatFingerprint(answer.fingerprint) ?? '',
    }
  },
  getPageUrl: (...[path, pageNumber, width, fingerprint = '']: [string, number, number, string?]) =>
    `${PLUGIN_ROUTE}/pages/${pageNumber}?path=${encodeURIComponent(path)}&wide=${width}&${getBytesQuery(fingerprint)}`,
  getHighlights: async (path, spans) => {
    const answer = await retryWhileBusy(() =>
      client.documents.getHighlights({ path, spans: [...spans] }),
    )
    return spans.map((_, index) => {
      const run = answer.runs[index]
      return run ? toHighlightedPages(run) : []
    })
  },
}
