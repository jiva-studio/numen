/**
 * Wire adapter for the plugin's JSON API.
 */
import {
  createApi,
  formatFingerprint,
  getBytesQuery,
  MAX_BUSY_RETRIES,
  UNAVAILABLE,
} from '@numen/plugin-sdk'
import type { Api } from '@numen/plugin-sdk'
import type { Documents, PageHighlight, Rect } from '../types'
import type {
  DocumentRect,
  DocumentRun,
  GetDocumentRequest,
  GetDocumentResponse,
  GetHighlightsRequest,
  GetHighlightsResponse,
} from './types'

/** The service name the manifest declares. */
export const API_SERVICE = 'numen.pdf'

/** Where the plugin's backend answers the window: the core forwards /plugin/<id>/ to it. */
const PLUGIN_ROUTE = '/plugin/md.numen.readers.pdf'

const BUSY_RETRY_DELAY_MS = 1000

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

/** Calls a method, retries while the backend is unavailable, and throws on failure. */
const callWhileBusy = async <Req, Res>(api: Api, method: string, request: Req): Promise<Res> => {
  for (let attempt = 0; ; attempt++) {
    const result = await api.call<Req, Res>(method, request)
    if (result.ok) return result.value
    if (result.error.code !== UNAVAILABLE || attempt >= MAX_BUSY_RETRIES) {
      throw new Error(result.error.message)
    }
    await sleep(BUSY_RETRY_DELAY_MS)
  }
}

/** The corners of one rectangle, in fractions of its page. */
const toRect = (rect: DocumentRect): Rect => ({
  minX: rect.minX,
  minY: rect.minY,
  maxX: rect.maxX,
  maxY: rect.maxY,
})

/** The location of one run of text, page by page. */
const toHighlightedPages = (run: DocumentRun): PageHighlight[] =>
  run.places.map((place) => ({ page: place.page, rects: place.rects.map(toRect) }))

export const createDocuments = (api: Api): Documents => ({
  getDocumentLayout: async (path) => {
    const response = await callWhileBusy<GetDocumentRequest, GetDocumentResponse>(
      api,
      'GetDocument',
      { path },
    )
    return {
      pages: response.pages.map((one) => ({ width: one.width, height: one.height })),
      stat:
        formatFingerprint(
          response.stat && {
            path: response.stat.path,
            size: BigInt(response.stat.size),
            mtime: BigInt(response.stat.mtime),
          },
        ) ?? '',
    }
  },
  getPageUrl: (...[path, pageNumber, width, stat = '']: [string, number, number, string?]) =>
    `${PLUGIN_ROUTE}/pages/${pageNumber}?path=${encodeURIComponent(path)}&wide=${width}&${getBytesQuery(stat)}`,
  getHighlights: async (path, spans) => {
    const response = await callWhileBusy<GetHighlightsRequest, GetHighlightsResponse>(
      api,
      'GetHighlights',
      { path, ranges: spans.map((span) => ({ start: span.start, end: span.end })) },
    )
    return spans.map((_, index) => {
      const run = response.runs[index]
      return run ? toHighlightedPages(run) : []
    })
  },
})

export const documents: Documents = createDocuments(createApi(API_SERVICE))
