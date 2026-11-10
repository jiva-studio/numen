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
import type { Documents, PageHighlight, Rect, TextRange } from '../types'
import type {
  DocumentPlace,
  DocumentRect,
  DocumentRun,
  FileStat,
  GetDocumentRequest,
  GetDocumentResponse,
  GetHighlightsRequest,
  GetHighlightsResponse,
  Page,
  Range,
} from './types'

/** The service name the manifest declares. */
const API_SERVICE = 'md.numen.readers.pdf'

/** Where the plugin's backend answers the window: the core forwards /plugin/<id>/ to it. */
const PLUGIN_ROUTE = '/plugin/md.numen.readers.pdf'

const BUSY_RETRY_DELAY_MS = 1000

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

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

const toRect = (rect: DocumentRect): Rect => ({
  minX: rect.minX,
  minY: rect.minY,
  maxX: rect.maxX,
  maxY: rect.maxY,
})

const toHighlightedPage = (place: DocumentPlace): PageHighlight => ({
  page: place.page,
  rects: place.rects.map(toRect),
})

const toHighlightedPages = (run: DocumentRun): PageHighlight[] => run.places.map(toHighlightedPage)

const toPageSize = (page: Page) => ({ width: page.width, height: page.height })

const toRange = (span: TextRange): Range => ({ start: span.start, end: span.end })

const toFingerprint = (stat: FileStat | undefined) =>
  formatFingerprint(
    stat && { path: stat.path, size: BigInt(stat.size), mtime: BigInt(stat.mtime) },
  ) ?? ''

export const createDocuments = (api: Api): Documents => ({
  getDocumentLayout: async (path) => {
    const response = await callWhileBusy<GetDocumentRequest, GetDocumentResponse>(
      api,
      'GetDocument',
      { path },
    )
    return {
      pages: response.pages.map(toPageSize),
      stat: toFingerprint(response.stat),
    }
  },
  getPageUrl: (...[path, pageNumber, width, stat = '']: [string, number, number, string?]) =>
    `${PLUGIN_ROUTE}/pages/${pageNumber}?path=${encodeURIComponent(path)}&wide=${width}&${getBytesQuery(stat)}`,
  getHighlights: async (path, spans) => {
    const response = await callWhileBusy<GetHighlightsRequest, GetHighlightsResponse>(
      api,
      'GetHighlights',
      { path, ranges: spans.map(toRange) },
    )
    return spans.map((_, index) => {
      const run = response.runs[index]
      return run ? toHighlightedPages(run) : []
    })
  },
})

export const documents: Documents = createDocuments(createApi(API_SERVICE))
