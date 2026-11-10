/**
 * What the note calls send to the core and what they make of its answers. The core's side of the
 * wire is a stand-in for `fetch`, which answers in the protocol's JSON encoding.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createConnectTransport } from '@connectrpc/connect-web'
import { NOTE_ERROR_CODE, formatErrorCodeMessage } from '@numen/core-client'
import { createNoteApi } from './notes'

/** The file a note was read from, as the wire writes it. */
const FILE = { path: 'Kept.md', size: '10', mtime: '7' }

function answerWith(answer: object) {
  const requests: unknown[] = []
  vi.stubGlobal('fetch', async (_input: unknown, init?: RequestInit) => {
    requests.push(JSON.parse(new TextDecoder().decode(init?.body as Uint8Array)))
    return new Response(JSON.stringify(answer), {
      headers: { 'content-type': 'application/json' },
    })
  })
  const api = createNoteApi(createConnectTransport({ baseUrl: 'http://127.0.0.1:1' }))
  return { api, requests }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('reading a note', () => {
  it('gives the prose and the file it came out of', async () => {
    const { api, requests } = answerWith({ body: 'One line.\n', at: FILE })

    expect(await api.readNote('Kept.md')).toStrictEqual({
      ok: true,
      value: { prose: 'One line.\n', stamp: { path: 'Kept.md', size: 10n, mtime: 7n } },
    })
    expect(requests).toStrictEqual([{ path: 'Kept.md' }])
  })

  it('says why the core did not read it, in words', async () => {
    const { api } = answerWith({ error: NOTE_ERROR_CODE.NOT_FOUND })

    const answer = await api.readNote('Gone.md')

    expect(answer).toStrictEqual({
      ok: false,
      error: formatErrorCodeMessage(NOTE_ERROR_CODE.NOT_FOUND),
    })
    expect(answer.ok ? '' : answer.error).not.toMatch(/\d/)
  })
})

describe('writing a note', () => {
  it('sends the prose with the version the caller read', async () => {
    const { api, requests } = answerWith({})

    const answer = await api.writeNote('Kept.md', 'Two lines.\n', {
      prose: 'One line.\n',
      stamp: { path: 'Kept.md', size: 10n, mtime: 7n },
    })

    expect(answer).toStrictEqual({ ok: true, value: null })
    expect(requests).toStrictEqual([
      {
        path: 'Kept.md',
        body: 'Two lines.\n',
        baseline: { prose: 'One line.\n', at: FILE },
      },
    ])
  })

  it('sends no version for a note that was read without one', async () => {
    const { api, requests } = answerWith({})

    await api.writeNote('Kept.md', 'Two lines.\n', null)

    expect(requests).toStrictEqual([{ path: 'Kept.md', body: 'Two lines.\n' }])
  })

  it('says why the core did not write it, in words', async () => {
    const { api } = answerWith({ error: NOTE_ERROR_CODE.CONFLICT })

    expect(await api.writeNote('Kept.md', 'Two lines.\n', null)).toStrictEqual({
      ok: false,
      error: formatErrorCodeMessage(NOTE_ERROR_CODE.CONFLICT),
    })
  })
})
