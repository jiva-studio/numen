/**
 * The spine of a book and the markup of one document of it, through the numen.epub JSON API.
 *
 * Each method is proved against the fixtures in testdata/api that the Go messages are proved
 * against: the request a method sends equals the request fixture, and the response fixture is read
 * into the model.
 */
import { asFailure, asValue } from '@numen/plugin-sdk'
import type { Api, ApiError } from '@numen/plugin-sdk'
import { describe, expect, it, vi } from 'vitest'
import findOffsetRequest from '../../testdata/api/FindOffset.request.json'
import findOffsetResponse from '../../testdata/api/FindOffset.response.json'
import getBookRequest from '../../testdata/api/GetBook.request.json'
import getBookResponse from '../../testdata/api/GetBook.response.json'
import getCfiRequest from '../../testdata/api/GetCfi.request.json'
import getCfiResponse from '../../testdata/api/GetCfi.response.json'
import readBookEntryRequest from '../../testdata/api/ReadBookEntry.request.json'
import readBookEntryResponse from '../../testdata/api/ReadBookEntry.response.json'
import readBookMarkupRequest from '../../testdata/api/ReadBookMarkup.request.json'
import readBookMarkupResponse from '../../testdata/api/ReadBookMarkup.response.json'
import type {
  FindOffsetRequest,
  FindOffsetResponse,
  GetBookRequest,
  GetBookResponse,
  GetCfiRequest,
  GetCfiResponse,
  ReadBookEntryRequest,
  ReadBookEntryResponse,
  ReadBookMarkupRequest,
  ReadBookMarkupResponse,
} from './messages'
import { createBooks, BookApiError } from './wire'

// The fixtures are the messages the Go backend encodes; these assignments fail to compile when a
// message and its fixture differ in shape.
const requests: [
  GetBookRequest,
  ReadBookMarkupRequest,
  ReadBookEntryRequest,
  GetCfiRequest,
  FindOffsetRequest,
] = [getBookRequest, readBookMarkupRequest, readBookEntryRequest, getCfiRequest, findOffsetRequest]
const responses: [
  GetBookResponse,
  ReadBookMarkupResponse,
  ReadBookEntryResponse,
  GetCfiResponse,
  FindOffsetResponse,
] = [
  getBookResponse,
  readBookMarkupResponse,
  readBookEntryResponse,
  getCfiResponse,
  findOffsetResponse,
]

const stat = '12 1700000000000000000 Books/Anabasis.epub'

const createApiStub = (reply: unknown) => {
  const call = vi.fn().mockResolvedValue(asValue(reply))
  return { api: { call, stream: vi.fn() } as unknown as Api, call }
}

const createFailingApiStub = (error: ApiError) => {
  const call = vi.fn().mockResolvedValue(asFailure(error))
  return { api: { call, stream: vi.fn() } as unknown as Api, call }
}

describe('the fixtures', () => {
  it('are the JSON of the message types', () => {
    expect(requests).toHaveLength(5)
    expect(responses).toHaveLength(5)
    expect(JSON.parse(JSON.stringify(getBookResponse))).toEqual(getBookResponse)
  })
})

describe('a book', () => {
  it('is read from the GetBook response', async () => {
    const { api, call } = createApiStub(getBookResponse)

    expect(await createBooks(api).getBook('Books/Anabasis.epub')).toEqual({
      title: 'Anabasis',
      span: { start: 0, end: 4000 },
      documents: [
        { path: 'OEBPS/one.xhtml', span: { start: 0, end: 1500 } },
        { path: 'OEBPS/two.xhtml', span: { start: 1500, end: 4000 } },
      ],
      parts: [{ title: 'Book I', offset: 0, level: 1 }],
      printedPages: [{ label: 'iv', offset: 120 }],
      pages: 8,
      pageBytes: 500,
      stat,
    })
    expect(call).toHaveBeenCalledWith('GetBook', getBookRequest)
  })

  it('names no file where the response carries none', async () => {
    const { stat: _omitted, ...withoutStat } = getBookResponse
    const { api } = createApiStub(withoutStat)

    expect((await createBooks(api).getBook('Books/Anabasis.epub')).stat).toBe('')
  })

  it('fails with the error of the method', async () => {
    const { api } = createFailingApiStub({ code: 'not_found', message: 'no book' })

    const failure = await createBooks(api)
      .getBook('missing.epub')
      .catch((error: unknown) => error)

    expect(failure).toBeInstanceOf(BookApiError)
    expect(failure).toMatchObject({ code: 'not_found', message: 'no book' })
  })
})

describe('the markup of one document', () => {
  it('names the file the window read', async () => {
    const { api, call } = createApiStub(readBookMarkupResponse)

    expect(await createBooks(api).readMarkup('Books/Anabasis.epub', 'OEBPS/one.xhtml', stat)).toBe(
      '<p>Up country</p>',
    )
    expect(call).toHaveBeenCalledWith('ReadBookMarkup', readBookMarkupRequest)
  })

  it('names no file where the window read none', async () => {
    const { api, call } = createApiStub(readBookMarkupResponse)

    await createBooks(api).readMarkup('Anabasis.epub', 'one.xhtml', '')

    expect(call).toHaveBeenCalledWith('ReadBookMarkup', {
      path: 'Anabasis.epub',
      document: 'one.xhtml',
    })
  })
})

describe('a place in a book', () => {
  it('is the CFI the backend names an offset by', async () => {
    const { api, call } = createApiStub(getCfiResponse)

    expect(await createBooks(api).getCfi('Books/Anabasis.epub', 120)).toBe(
      'epubcfi(/8/4!/4/2/1:17)',
    )
    expect(call).toHaveBeenCalledWith('GetCfi', getCfiRequest)
  })

  it('is the offset the backend finds a CFI at', async () => {
    const { api, call } = createApiStub(findOffsetResponse)

    expect(
      await createBooks(api).findOffset('Books/Anabasis.epub', 'epubcfi(/8/4!/4/2/1:17)'),
    ).toBe(120)
    expect(call).toHaveBeenCalledWith('FindOffset', findOffsetRequest)
  })

  it('is nothing where the backend finds no such place or reads no CFI', async () => {
    const notFound = createFailingApiStub({ code: 'not_found', message: 'no document' })
    const invalid = createFailingApiStub({ code: 'invalid_argument', message: 'no CFI' })

    expect(
      await createBooks(notFound.api).findOffset('Anabasis.epub', 'epubcfi(/6/2!/4)'),
    ).toBeNull()
    expect(await createBooks(notFound.api).getCfi('Anabasis.epub', 0)).toBeNull()
    expect(await createBooks(invalid.api).findOffset('Anabasis.epub', 'page 3')).toBeNull()
  })

  it('passes on any other failure', async () => {
    const { api } = createFailingApiStub({ code: 'internal', message: 'disk' })

    await expect(createBooks(api).getCfi('Anabasis.epub', 0)).rejects.toThrow('disk')
  })
})

describe('an entry of a book', () => {
  const books = createBooks(createApiStub({}).api)

  it('is served on the plugin route, with the book and the file it was read from', () => {
    expect(books.getEntryUrl('Books/Anabasis.epub', 'OEBPS/images/map one.png', stat)).toBe(
      '/plugin/md.numen.readers.epub/entries/OEBPS/images/map%20one.png?path=Books%2FAnabasis.epub&size=12&mtime=1700000000000000000',
    )
  })

  it('names no file where the book was read from none', () => {
    expect(books.getEntryUrl('Anabasis.epub', 'cover.png', '')).toBe(
      '/plugin/md.numen.readers.epub/entries/cover.png?path=Anabasis.epub',
    )
  })
})
