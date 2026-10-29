/**
 * Wire adapter for the plugin's numen.epub JSON API and its entries route.
 */
import { createApi, getBytesQuery } from '@numen/plugin-sdk'
import type { Api, ApiError } from '@numen/plugin-sdk'
import type { Book, Books, SpineDocument } from '../types'
import type {
  FileStat,
  FindOffsetRequest,
  FindOffsetResponse,
  GetBookRequest,
  GetBookResponse,
  GetCfiRequest,
  GetCfiResponse,
  ReadBookMarkupRequest,
  ReadBookMarkupResponse,
  SpineDocument as SpineDocumentMessage,
} from './messages'

/** Where the plugin's backend answers the window: the core forwards /plugin/<id>/ to it. */
const PLUGIN_ROUTE = '/plugin/md.numen.readers.epub'

/** The service name the manifest declares. */
const SERVICE = 'numen.epub'

/** How often a method that answers `unavailable` is asked again before it fails. */
const MAX_BUSY_RETRIES = 3

/** How long the window waits before asking an unavailable method again. */
const BUSY_RETRY_DELAY_MS = 1000

/** A method of the API that failed. */
export class BookApiError extends Error {
  readonly code: string

  constructor(error: ApiError) {
    super(error.message)
    this.name = 'BookApiError'
    this.code = error.code
  }
}

/** The file stat of a fingerprint `<size> <mtime> <path>`. */
const parseStat = (stat: string): FileStat => {
  const [size = '0', mtime = '0', ...rest] = stat.split(' ')
  return { path: rest.join(' '), size: Number(size), mtime }
}

/** The fingerprint `<size> <mtime> <path>` of a file stat. */
const formatStat = (stat: FileStat): string => `${stat.size} ${stat.mtime} ${stat.path}`

/** One document of the spine, as the window carries it. */
const readSpineDocument = (one: SpineDocumentMessage): SpineDocument => ({
  path: one.path,
  span: { start: one.offset, end: one.offset + one.length },
})

const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

/** Creates the books port over the client of the numen.epub API. */
export const createBooks = (api: Api): Books => {
  /** The response of a method, asked again while the method is unavailable. */
  const call = async <Req, Res>(method: string, request: Req): Promise<Res> => {
    for (let attempt = 0; ; attempt++) {
      const result = await api.call<Req, Res>(method, request)
      if (result.ok) return result.value
      if (result.error.code !== 'unavailable' || attempt >= MAX_BUSY_RETRIES) {
        throw new BookApiError(result.error)
      }
      await wait(BUSY_RETRY_DELAY_MS)
    }
  }

  /** The response of a method about a place, and null where the book names no such place. */
  const findPlace = async <Req, Res>(method: string, request: Req): Promise<Res | null> => {
    try {
      return await call<Req, Res>(method, request)
    } catch (error) {
      if (
        error instanceof BookApiError &&
        (error.code === 'not_found' || error.code === 'invalid_argument')
      ) {
        return null
      }
      throw error
    }
  }

  return {
    getBook: async (path) => {
      const answer = await call<GetBookRequest, GetBookResponse>('GetBook', { path })
      return {
        title: answer.title,
        span: { start: 0, end: answer.textBytes },
        documents: answer.documents.map(readSpineDocument),
        parts: answer.parts.map((one) => ({
          title: one.title,
          offset: one.offset,
          level: one.level,
        })),
        printedPages: answer.printedPages.map((one) => ({
          label: one.label,
          offset: one.offset,
        })),
        pages: answer.pageCount,
        pageBytes: answer.pageBytes,
        stat: answer.stat ? formatStat(answer.stat) : '',
      } satisfies Book
    },
    readMarkup: async (path, document, stat) => {
      const answer = await call<ReadBookMarkupRequest, ReadBookMarkupResponse>('ReadBookMarkup', {
        path,
        document,
        ...(stat === '' ? {} : { baseline: parseStat(stat) }),
      })
      return answer.markup
    },
    getEntryUrl: (path, name, stat) =>
      `${PLUGIN_ROUTE}/entries/${name.split('/').map(encodeURIComponent).join('/')}?path=${encodeURIComponent(path)}${stat === '' ? '' : `&${getBytesQuery(stat)}`}`,
    getCfi: async (path, offset) => {
      const answer = await findPlace<GetCfiRequest, GetCfiResponse>('GetCfi', { path, offset })
      return answer?.cfi ?? null
    },
    findOffset: async (path, cfi) => {
      const answer = await findPlace<FindOffsetRequest, FindOffsetResponse>('FindOffset', {
        path,
        cfi,
      })
      return answer?.offset ?? null
    },
  }
}

export const books: Books = createBooks(createApi(SERVICE))
