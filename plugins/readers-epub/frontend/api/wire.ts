/**
 * Wire adapter for the plugin's BookService and its entries route.
 */
import { Code, ConnectError, createClient } from '@connectrpc/connect'
import {
  formatFingerprint,
  getBytesQuery,
  parseFingerprint,
  retryWhileBusy,
  transport,
} from '@numen/wire'
import {
  BookService,
  type SpineDocument as SpineDocumentMessage,
} from '../gen/numen/epub/v1/book_pb.js'
import type { Book, Books, SpineDocument } from '../types'

/** Where the plugin's backend answers the window: the core forwards /plugin/<id>/ to it. */
const PLUGIN_ROUTE = '/plugin/md.numen.readers.epub'

export const client = {
  books: createClient(BookService, transport),
}

/** One document of the spine, as the window carries it. */
const readSpineDocument = (one: SpineDocumentMessage): SpineDocument => ({
  path: one.path,
  span: { from: one.offset, to: one.offset + one.length },
})

/** The answer of a call about a place, and null where the book names no such place. */
const findPlace = async <T>(call: () => Promise<T>): Promise<T | null> => {
  try {
    return await retryWhileBusy(call)
  } catch (error) {
    const { code } = ConnectError.from(error)
    if (code === Code.NotFound || code === Code.InvalidArgument) return null
    throw error
  }
}

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
      stat: formatFingerprint(answer.stat) ?? '',
    } satisfies Book
  },
  readMarkup: async (path, document, stat) => {
    const answer = await retryWhileBusy(() =>
      client.books.readBookMarkup({
        path,
        document,
        ...(stat === '' ? {} : { baseline: parseFingerprint(stat) }),
      }),
    )
    return answer.markup
  },
  getEntryUrl: (path, name, stat) =>
    `${PLUGIN_ROUTE}/entries/${name.split('/').map(encodeURIComponent).join('/')}?path=${encodeURIComponent(path)}${stat === '' ? '' : `&${getBytesQuery(stat)}`}`,
  getCfi: async (path, offset) => {
    const answer = await findPlace(() => client.books.getCfi({ path, offset }))
    return answer?.cfi ?? null
  },
  findOffset: async (path, cfi) => {
    const answer = await findPlace(() => client.books.findOffset({ path, cfi }))
    return answer?.offset ?? null
  },
}
