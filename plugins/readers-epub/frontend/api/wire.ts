/**
 * Wire adapter for the plugin's BookService and its entries route.
 */
import { createClient } from '@connectrpc/connect'
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
      anchors: answer.anchors.map((one) => ({ offset: one.offset, cfi: one.cfi })),
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
    `${PLUGIN_ROUTE}/entries/${name.split('/').map(encodeURIComponent).join('/')}?path=${encodeURIComponent(path)}${fingerprint === '' ? '' : `&${getBytesQuery(fingerprint)}`}`,
}
