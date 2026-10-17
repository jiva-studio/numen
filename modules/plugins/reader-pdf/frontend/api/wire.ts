/**
 * Wire adapter for DocumentService and OcrService.
 */
import { Code, createClient, type ConnectError } from '@connectrpc/connect'
import {
  DocumentService,
  OcrService,
  type Box as BoxMessage,
  type Run as RunMessage,
} from '@numen/protocol'
import { transport } from '@numen/wire'
import type { Documents, PageHighlight, Rect } from '../types'

const formatFingerprint = (at?: {
  path: string
  size: bigint
  mtime: bigint
}): string | undefined => at && `${at.size} ${at.mtime} ${at.path}`

const parseFingerprint = (at: string) => {
  const [size = '0', mtime = '0', ...rest] = at.split(' ')
  return { path: rest.join(' '), size: BigInt(size), mtime: BigInt(mtime) }
}

const getAssetUrl = (path: string): string => `/assets/${encodeURIComponent(path)}`

const getBytesQuery = (file: string): string => {
  const at = parseFingerprint(file)
  return `size=${at.size}&mtime=${at.mtime}`
}

const MAX_BUSY_RETRIES = 3
const BUSY_RETRY_DELAY_MS = 1000

const retryWhileBusy = async <T>(ask: () => Promise<T>): Promise<T> => {
  for (let attempt = 0; ; attempt++) {
    try {
      return await ask()
    } catch (error) {
      if (Code.Unavailable !== (error as ConnectError).code || attempt >= MAX_BUSY_RETRIES)
        throw error
      await new Promise((wake) => setTimeout(wake, BUSY_RETRY_DELAY_MS))
    }
  }
}

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
  getPageUrl: (...[path, pageNumber, wide, fingerprint = '']: [string, number, number, string?]) =>
    `${getAssetUrl(path)}/pages/${pageNumber}?wide=${wide}&${getBytesQuery(fingerprint)}`,
  getHighlights: async (path, spans) => {
    const answer = await retryWhileBusy(() => client.ocr.readOcr({ path, spans: [...spans] }))
    return spans.map((_, at) => {
      const run = answer.runs[at]
      return run ? toHighlightedPages(run) : []
    })
  },
}
