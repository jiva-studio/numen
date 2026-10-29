/**
 * The document wire adapter turns API responses into layouts and highlights, and sends the
 * requests the backend contract fixtures describe.
 */
import { describe, expect, it, vi } from 'vitest'
import type { Api, ApiError } from '@numen/plugin-sdk'
import { createDocuments } from './wire'
import type { GetDocumentResponse, GetHighlightsResponse } from './types'
import getDocumentRequest from '../../testdata/api/GetDocument.request.json'
import getDocumentResponse from '../../testdata/api/GetDocument.response.json'
import getHighlightsRequest from '../../testdata/api/GetHighlights.request.json'
import getHighlightsResponse from '../../testdata/api/GetHighlights.response.json'

const createApiOf = (response: unknown) => {
  const call = vi.fn(async () => ({ ok: true as const, value: response }))
  const api = { call, stream: vi.fn() } as unknown as Api
  return { api, call }
}

const failure: ApiError = { code: 'not_found', message: 'no such file' }

describe('contract fixtures', () => {
  it('parse into the API types', () => {
    const document: GetDocumentResponse = getDocumentResponse
    const highlights: GetHighlightsResponse = getHighlightsResponse
    expect(document.pages).toHaveLength(2)
    expect(document.stat).toEqual({ path: 'Books/Book.pdf', size: 12, mtime: 1700000000 })
    expect(highlights.runs[0]?.places[0]?.rects[0]).toEqual({
      minX: 0.25,
      minY: 0.5,
      maxX: 0.75,
      maxY: 0.625,
    })
  })
})

describe('getDocumentLayout', () => {
  it('returns the page sizes and the fingerprint of the file', async () => {
    const { api, call } = createApiOf(getDocumentResponse)

    const layout = await createDocuments(api).getDocumentLayout(getDocumentRequest.path)

    expect(layout).toEqual({
      pages: [
        { width: 612, height: 792 },
        { width: 595.5, height: 842 },
      ],
      stat: '12 1700000000 Books/Book.pdf',
    })
    expect(call).toHaveBeenCalledWith('GetDocument', getDocumentRequest)
  })

  it('returns an empty fingerprint where the response has no stat', async () => {
    const { api } = createApiOf({ pages: [] })

    expect(await createDocuments(api).getDocumentLayout('Book.pdf')).toEqual({
      pages: [],
      stat: '',
    })
  })

  it('throws the message of a failed call', async () => {
    const api = {
      call: vi.fn(async () => ({ ok: false as const, error: failure })),
      stream: vi.fn(),
    } as unknown as Api

    await expect(createDocuments(api).getDocumentLayout('Book.pdf')).rejects.toThrow('no such file')
    expect(api.call).toHaveBeenCalledTimes(1)
  })

  it('asks again while the backend is unavailable', async () => {
    vi.useFakeTimers()
    const call = vi
      .fn()
      .mockResolvedValueOnce({ ok: false, error: { code: 'unavailable', message: 'busy' } })
      .mockResolvedValueOnce({ ok: true, value: { pages: [] } })
    const api = { call, stream: vi.fn() } as unknown as Api

    const layout = createDocuments(api).getDocumentLayout('Book.pdf')
    await vi.advanceTimersByTimeAsync(1000)

    expect(await layout).toEqual({ pages: [], stat: '' })
    expect(call).toHaveBeenCalledTimes(2)
    vi.useRealTimers()
  })
})

describe('getPageUrl', () => {
  it('names the width and the bytes of the file', () => {
    const { api } = createApiOf({})

    expect(createDocuments(api).getPageUrl('Books/Book.pdf', 3, 1200, '12 34 Books/Book.pdf')).toBe(
      '/plugin/md.numen.readers.pdf/pages/3?path=Books%2FBook.pdf&wide=1200&size=12&mtime=34',
    )
  })

  it('names zero bytes where the caller has no fingerprint', () => {
    const { api } = createApiOf({})

    expect(createDocuments(api).getPageUrl('Book.pdf', 1, 800)).toBe(
      '/plugin/md.numen.readers.pdf/pages/1?path=Book.pdf&wide=800&size=0&mtime=0',
    )
  })
})

describe('getHighlights', () => {
  it('returns the places of each range and sends the ranges', async () => {
    const { api, call } = createApiOf(getHighlightsResponse)

    const highlights = await createDocuments(api).getHighlights(
      getHighlightsRequest.path,
      getHighlightsRequest.ranges,
    )

    expect(highlights).toEqual([
      [{ page: 2, rects: [{ minX: 0.25, minY: 0.5, maxX: 0.75, maxY: 0.625 }] }],
      [],
    ])
    expect(call).toHaveBeenCalledWith('GetHighlights', getHighlightsRequest)
  })

  it('returns no places for a range the response omits', async () => {
    const { api } = createApiOf({ runs: [] })

    expect(await createDocuments(api).getHighlights('Book.pdf', [{ start: 0, end: 9 }])).toEqual([
      [],
    ])
  })
})
