/**
 * Wire adapter for BookService.
 */
import type { SpineDocument as SpineDocumentMessage } from '@numen/protocol'
import * as answers from '@/shared/answers'
import * as clients from '@/shared/clients'
import type { Book, Books, SpineDocument } from '../types'

const served = {
  books: clients.books,
}

/** One document of the spine, as the window carries it. */
const readSpineDocument = (one: SpineDocumentMessage): SpineDocument => ({
  path: one.path,
  span: { from: one.offset, to: one.offset + one.length },
})

export const books: Books = {
  getBook: async (path) => {
    const answer = await answers.retryWhileBusy(() => served.books.getBook({ path }))
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
      fingerprint: answers.stamp(answer.fingerprint) ?? '',
    } satisfies Book
  },
  readMarkup: async (path, document, fingerprint) => {
    const answer = await answers.retryWhileBusy(() =>
      served.books.readBookMarkup({
        path,
        document,
        ...(fingerprint === '' ? {} : { seen: answers.fingerprint(fingerprint) }),
      }),
    )
    return answer.markup
  },
  getEntryUrl: (path, name, fingerprint) =>
    `${answers.asset(path)}/${name.split('/').map(encodeURIComponent).join('/')}?${answers.getBytesQuery(fingerprint)}`,
}
