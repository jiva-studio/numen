/**
 * The service of the saved search notes as the tab calls it: the fixtures of testdata/api are the
 * JSON the Go messages encode to, read here as the TypeScript messages, and the client maps the
 * failures of the wire to ServiceFailure.
 */
import { describe, expect, it } from 'vitest'
import { createApi } from '@numen/plugin-sdk'
import { createSavedSearchService, SERVICE, ServiceFailure } from './service'
import type {
  GetResultsRequest,
  GetResultsResponse,
  ListSearchFunctionsResponse,
  ResultsChange,
  UpdateSearchRequest,
  UpdateSearchResponse,
  WatchResultsRequest,
} from './types'

const fixtures = import.meta.glob('../../../testdata/api/*.json', {
  eager: true,
  import: 'default',
}) as Record<string, unknown>

const read = (name: string): unknown => fixtures[`../../../testdata/api/${name}.json`]

const ORIGIN = 'http://window'
const PATH = 'Harbor/Harbor master - winter storms.md'

interface Sent {
  readonly url: string
  readonly body: unknown
}

const encoder = new TextEncoder()

const createService = (respond: (url: string, init: RequestInit) => Response) => {
  const sent: Sent[] = []
  const service = createSavedSearchService(
    createApi(SERVICE, {
      origin: ORIGIN,
      fetch: async (url, init) => {
        sent.push({ url: String(url), body: JSON.parse(String(init?.body)) })
        return respond(String(url), init ?? {})
      },
    }),
  )
  return { service, sent }
}

const sse = (text: string): Response =>
  new Response(
    new ReadableStream({
      start: (controller) => {
        controller.enqueue(encoder.encode(text))
        controller.close()
      },
    }),
    { headers: { 'Content-Type': 'text/event-stream' } },
  )

const openStream = (init: RequestInit, change: ResultsChange): Response =>
  new Response(
    new ReadableStream({
      start: (controller) => {
        controller.enqueue(encoder.encode(`data: ${JSON.stringify(change)}\n\n`))
        init.signal?.addEventListener('abort', () =>
          controller.error(new DOMException('The call was aborted.', 'AbortError')),
        )
      },
    }),
    { headers: { 'Content-Type': 'text/event-stream' } },
  )

const collect = async <T>(changes: AsyncGenerator<T, void>): Promise<T[]> => {
  const seen: T[] = []
  for await (const change of changes) seen.push(change)
  return seen
}

describe('the fixtures of the saved search service', () => {
  it('reads the GetResults fixture as GetResultsResponse', () => {
    const request: GetResultsRequest = { path: PATH }
    const response = read('GetResults.response') as GetResultsResponse

    expect(read('GetResults.request')).toStrictEqual(request)
    expect(response.path).toBe(PATH)
    expect(response.computedAt).toBe('2026-11-14T09:30:00Z')
    expect(response.failedAt).toBe('')
    expect(response.failure).toBe('')
    expect(response.results.map((one) => one.path)).toStrictEqual([
      'Logbooks/1911.pdf',
      'Harbor/Night of the gale.md',
      'Harbor/Pier plan.png',
    ])
    expect(response.results[0]?.place).toStrictEqual({ page: 14 })
    expect(response.results[2]?.isWholeFile).toBe(true)
    expect(response.errors).toStrictEqual([])
  })

  it('reads the WatchResults fixture as ResultsChange', () => {
    const request: WatchResultsRequest = { path: PATH }
    const change: ResultsChange = { computedAt: '2026-11-14T09:30:00Z', isRunning: false }

    expect(read('WatchResults.request')).toStrictEqual(request)
    expect(read('WatchResults.response')).toStrictEqual(change)
  })

  it('reads the UpdateSearch fixture as UpdateSearchRequest and UpdateSearchResponse', () => {
    const request: UpdateSearchRequest = {
      path: PATH,
      query: 'kind in ["note", "pdf"]\n&& meaning("the harbor master in a winter storm")\n.top(20)',
    }
    const response: UpdateSearchResponse = {
      errors: [{ line: 2, column: 4, message: "undeclared reference to 'meanin'" }],
    }

    expect(read('UpdateSearch.request')).toStrictEqual(request)
    expect(read('UpdateSearch.response')).toStrictEqual(response)
  })
})

describe('the saved search service', () => {
  it('asks GetResults for the note and answers its response', async () => {
    const answer = read('GetResults.response') as GetResultsResponse
    const { service, sent } = createService(() => Response.json(answer))

    const response = await service.getResults({ path: PATH }, new AbortController().signal)

    expect(sent).toStrictEqual([{ url: `${ORIGIN}/${SERVICE}/GetResults`, body: { path: PATH } }])
    expect(response).toStrictEqual(answer)
  })

  it('throws the message of a failed call', async () => {
    const { service } = createService(() =>
      Response.json({ code: 'not_found', message: `there is no note at ${PATH}` }, { status: 404 }),
    )

    const answer = service.getResults({ path: PATH }, new AbortController().signal)

    await expect(answer).rejects.toBeInstanceOf(ServiceFailure)
    await expect(answer).rejects.toThrow(`there is no note at ${PATH}`)
  })

  it('yields each change of WatchResults until the stream ends', async () => {
    const { service, sent } = createService(() =>
      sse(
        'data: {"computedAt":"2026-11-14T09:30:00Z","isRunning":false}\n\n' +
          'data: {"computedAt":"2026-11-14T09:30:00Z","isRunning":true}\n\n' +
          'event: end\ndata: {}\n\n',
      ),
    )

    const changes = await collect(
      service.watchResults({ path: PATH }, new AbortController().signal),
    )

    expect(sent).toStrictEqual([{ url: `${ORIGIN}/${SERVICE}/WatchResults`, body: { path: PATH } }])
    expect(changes).toStrictEqual([
      { computedAt: '2026-11-14T09:30:00Z', isRunning: false },
      { computedAt: '2026-11-14T09:30:00Z', isRunning: true },
    ])
  })

  it('throws the message of a watch that ends in failure', async () => {
    const { service } = createService(() =>
      sse(
        'data: {"computedAt":"","isRunning":false}\n\n' +
          'event: error\ndata: {"code":"failed_precondition","message":"the window shows no vault"}\n\n',
      ),
    )

    const watching = collect(service.watchResults({ path: PATH }, new AbortController().signal))

    await expect(watching).rejects.toBeInstanceOf(ServiceFailure)
    await expect(watching).rejects.toThrow('the window shows no vault')
  })

  it('ends the watch when the signal aborts', async () => {
    const change: ResultsChange = { computedAt: '2026-11-14T09:30:00Z', isRunning: false }
    const { service } = createService((_url, init) => openStream(init, change))
    const abort = new AbortController()
    const changes = service.watchResults({ path: PATH }, abort.signal)

    const first = await changes.next()
    abort.abort()
    const last = await changes.next()

    expect(first).toStrictEqual({ done: false, value: change })
    expect(last.done).toBe(true)
  })

  it('sends the query to UpdateSearch', async () => {
    const request = read('UpdateSearch.request') as UpdateSearchRequest
    const { service, sent } = createService(() => Response.json({ errors: [] }))

    const response = await service.updateSearch(request, new AbortController().signal)

    expect(sent).toStrictEqual([{ url: `${ORIGIN}/${SERVICE}/UpdateSearch`, body: request }])
    expect(response).toStrictEqual({ errors: [] })
  })

  it('answers the problems of a query UpdateSearch did not save', async () => {
    const request = read('UpdateSearch.request') as UpdateSearchRequest
    const answer = read('UpdateSearch.response') as UpdateSearchResponse
    const { service } = createService(() => Response.json(answer))

    const response = await service.updateSearch(request, new AbortController().signal)

    expect(response.errors).toStrictEqual([
      { line: 2, column: 4, message: "undeclared reference to 'meanin'" },
    ])
  })

  it('throws the message of an UpdateSearch that failed', async () => {
    const request = read('UpdateSearch.request') as UpdateSearchRequest
    const { service } = createService(() =>
      Response.json(
        { code: 'aborted', message: 'the note changed while it was updated; try again' },
        { status: 409 },
      ),
    )

    const answer = service.updateSearch(request, new AbortController().signal)

    await expect(answer).rejects.toBeInstanceOf(ServiceFailure)
    await expect(answer).rejects.toThrow('the note changed while it was updated; try again')
  })

  it('asks ListSearchFunctions for the catalog and answers its functions', async () => {
    const answer = read('ListSearchFunctions.response') as ListSearchFunctionsResponse
    const { service, sent } = createService(() => Response.json(answer))

    const response = await service.listSearchFunctions({}, new AbortController().signal)

    expect(sent).toStrictEqual([{ url: `${ORIGIN}/${SERVICE}/ListSearchFunctions`, body: {} }])
    expect(response.functions.map((fn) => fn.name)).toStrictEqual(['words', 'top'])
    expect(response.functions[1]).toStrictEqual({
      name: 'top',
      kind: 'stage',
      signatures: ['.top(count)'],
      description: 'Keeps the first results.',
      example: 'words("gale").top(10)',
    })
  })
})
