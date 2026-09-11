/**
 * Wire adapter for BookService.
 */
import { createClient } from '@connectrpc/connect'
import { BookService } from '@numen/protocol'
import type { SpineDocument as SpineDocumentMessage } from '@numen/protocol'
import { transport } from '@numen/wire'
import { asset, fingerprint, named, stamp, waiting } from '../../shared/answers'
import type { Book, Books, SpineDocument } from './types'

const served = {
  books: createClient(BookService, transport),
}

/** One document of the spine, as the window carries it. */
const spined = (one: SpineDocumentMessage): SpineDocument => ({
  path: one.path,
  span: { begins: one.offset, ends: one.offset + one.length },
})

export const books: Books = {
  getBook: async (path) => {
    const answer = await waiting(() => served.books.getBook({ path }))
    return {
      title: answer.title,
      span: { begins: 0, ends: answer.textBytes },
      documents: answer.documents.map(spined),
      parts: answer.parts.map((one) => ({
        title: one.title,
        offset: one.offset,
        level: one.level,
      })),
      printed: answer.printedPages.map((one) => ({
        label: one.label,
        offset: one.offset,
      })),
      pages: answer.pageCount,
      pageBytes: answer.pageBytes,
      fingerprint: stamp(answer.fingerprint) ?? '',
    } satisfies Book
  },
  readMarkup: async (path, document, seen) => {
    const answer = await waiting(() =>
      served.books.readBookMarkup({
        path,
        document,
        ...(seen === '' ? {} : { seen: fingerprint(seen) }),
      }),
    )
    return answer.markup
  },
  getEntryUrl: (path, name, seen) =>
    `${asset(path)}/${name.split('/').map(encodeURIComponent).join('/')}?${named(seen)}`,
}

