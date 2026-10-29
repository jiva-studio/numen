import { describe, expect, it } from 'vitest'
import { createApi, parseEvents } from './api'

const createFetch =
  (respond: (url: string, init: RequestInit) => Response): typeof fetch =>
  async (url, init) =>
    respond(String(url), init ?? {})

const stream = (text: string): Response =>
  new Response(
    new ReadableStream({ start: (c) => (c.enqueue(new TextEncoder().encode(text)), c.close()) }),
    {
      headers: { 'Content-Type': 'text/event-stream' },
    },
  )

describe('createApi', () => {
  it('posts JSON to the service and the method and returns the response', async () => {
    let seen: { url: string; body: string } | undefined
    const api = createApi('org.example.echo', {
      origin: 'http://host',
      fetch: createFetch((url, init) => {
        seen = { url, body: String(init.body) }
        return Response.json({ text: 'ab' })
      }),
    })

    const result = await api.call<{ text: string }, { text: string }>('Echo', { text: 'a' })

    expect(seen).toEqual({ url: 'http://host/org.example.echo/Echo', body: '{"text":"a"}' })
    expect(result).toEqual({ ok: true, value: { text: 'ab' } })
  })

  it('returns the code and the message of a failing method', async () => {
    const api = createApi('s', {
      origin: 'http://host',
      fetch: createFetch(() =>
        Response.json({ code: 'not_found', message: 'no such text' }, { status: 404 }),
      ),
    })

    expect(await api.call('Echo', {})).toEqual({
      ok: false,
      error: { code: 'not_found', message: 'no such text' },
    })
  })

  it('returns an unavailable failure when the request does not complete', async () => {
    const api = createApi('s', {
      origin: 'http://host',
      fetch: async () => {
        throw new Error('offline')
      },
    })

    expect(await api.call('Echo', {})).toEqual({
      ok: false,
      error: { code: 'unavailable', message: 'offline' },
    })
  })

  it('yields each response of a stream and ends on the end event', async () => {
    const api = createApi('s', {
      origin: 'http://host',
      fetch: createFetch(() =>
        stream('data: {"n":1}\n\ndata: {"n":2}\n\nevent: end\ndata: {}\n\n'),
      ),
    })

    const seen: unknown[] = []
    const iterator = api.stream('Count', {})
    let step = await iterator.next()
    while (!step.done) {
      seen.push(step.value)
      step = await iterator.next()
    }

    expect(seen).toEqual([{ n: 1 }, { n: 2 }])
    expect(step.value).toBeNull()
  })

  it('returns the failure that ends a stream', async () => {
    const api = createApi('s', {
      origin: 'http://host',
      fetch: createFetch(() =>
        stream(
          'data: {"n":1}\n\nevent: error\ndata: {"code":"unavailable","message":"stopped"}\n\n',
        ),
      ),
    })

    const iterator = api.stream('Count', {})
    await iterator.next()
    const last = await iterator.next()

    expect(last).toEqual({ done: true, value: { code: 'unavailable', message: 'stopped' } })
  })
})

describe('parseEvents', () => {
  it('keeps an incomplete event as the rest', () => {
    expect(parseEvents('data: {"a":1}\n\ndata: {"b"')).toEqual({
      events: [{ name: 'message', data: '{"a":1}' }],
      rest: 'data: {"b"',
    })
  })
})
