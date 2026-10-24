/**
 * Wire adapter for BookService.
 */
import { createClient } from '@connectrpc/connect'
import { BookService, type SpineDocument as SpineDocumentMessage } from '@numen/protocol'
import {
  formatFingerprint,
  getAssetUrl,
  getBytesQuery,
  parseFingerprint,
  retryWhileBusy,
  transport,
} from '@numen/wire'
import type { Book, Books, SpineDocument } from '../types'

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
