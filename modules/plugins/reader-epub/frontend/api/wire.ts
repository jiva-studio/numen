/**
 * Wire adapter for BookService.
 */
import { Code, createClient, type ConnectError } from '@connectrpc/connect'
import { BookService, type SpineDocument as SpineDocumentMessage } from '@numen/protocol'
import { transport } from '@numen/wire'
import type { Book, Books, SpineDocument } from '../types'

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
  books: createClient(BookService, transport),
}

/** One document of the spine, as the window carries it. */
const readSpineDocument = (one: SpineDocumentMessage): SpineDocument => ({
  path: one.path,
  span: { from: one.offset, to: one.offset + one.length },
})

export const books: Books = {
  getBook: async (path) => {
    const answer = await retryWhileBusy(() => client.books.getBook({ path }))
    return {
      title: answer.title,
      span: { from: 0, to: answer.textBytes },
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
      fingerprint: formatFingerprint(answer.fingerprint) ?? '',
    } satisfies Book
  },
  readMarkup: async (path, document, fingerprint) => {
    const answer = await retryWhileBusy(() =>
      client.books.readBookMarkup({
        path,
        document,
        ...(fingerprint === '' ? {} : { baseline: parseFingerprint(fingerprint) }),
      }),
    )
    return answer.markup
  },
  getEntryUrl: (path, name, fingerprint) =>
    `${getAssetUrl(path)}/${name.split('/').map(encodeURIComponent).join('/')}?${getBytesQuery(fingerprint)}`,
}
