/**
 * What a book that reflows is, for whatever opens it: its spine documents,
 * parts, printed page labels, and markup.
 */
import { createClient } from '@connectrpc/connect'
import { BookService } from '@numen/protocol'
import type { SpineDocument as SpineDocumentMessage } from '@numen/protocol'
import { transport } from '@numen/wire'
import { asset, fingerprint, named, stamp, waiting } from '../shared/answers'
import type { Books, SpineDocument } from './open'

const served = {
  books: createClient(BookService, transport),
}

/** One document of the spine, as the window carries it. */
const spined = (one: SpineDocumentMessage): SpineDocument => ({
  path: one.path,
  span: { begins: one.offset, ends: one.offset + one.length },
})

/**
 * The books that reflow, over the same addresses. What a book is and the markup
 * of one document of it come over the schema; one picture it carries is bytes,
 * and bytes are answered at an address.
 */
export const books: Books = {
  shape: async (path) => {
    const answer = await waiting(() => served.books.getBook({ path }))
    return {
      title: answer.title,
      span: { begins: 0, ends: answer.textBytes },
      documents: answer.documents.map(spined),
      parts: answer.parts.map((one) => ({
        title: one.title,
        at: one.offset,
        level: one.level,
      })),
      printed: answer.printedPages.map((one) => ({ label: one.label, at: one.offset })),
      pages: answer.pageCount,
      pageBytes: answer.pageBytes,
      at: stamp(answer.fingerprint) ?? '',
    }
  },
  markup: async (path, document, seen) => {
    const answer = await waiting(() =>
      served.books.readBookMarkup({
        path,
        document,
        ...(seen === '' ? {} : { seen: fingerprint(seen) }),
      }),
    )
    return answer.markup
  },
  entry: (path, name, seen) =>
    `${asset(path)}/${name.split('/').map(encodeURIComponent).join('/')}?${named(seen)}`,
}
