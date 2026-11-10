/**
 * What the note calls send to the core and what they make of its answers. The core's side of the
 * wire is a stand-in for `fetch`, which answers in the protocol's binary encoding.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createConnectTransport } from '@connectrpc/connect-web'
import { NOTE_ERROR_CODE, formatErrorCodeMessage } from '@numen/core-client'
import { createNoteApi } from './notes'

const encoder = new TextEncoder()

/** A string field of a message. */
const text = (field: number, value: string) => [
  (field << 3) | 2,
  value.length,
  ...encoder.encode(value),
]

/** A number field of a message; a value under 128. */
const count = (field: number, value: number) => [field << 3, value]

/** A message field, holding `parts`. */
const message = (field: number, ...parts: number[][]) => {
  const body = parts.flat()
  return [(field << 3) | 2, body.length, ...body]
}

const stamp = (field: number) => message(field, text(1, 'Kept.md'), count(2, 10), count(3, 7))

/** Answers every request with `answer`, and returns the bodies it was sent. */
function answerWith(answer: number[]) {
  const sent: Uint8Array[] = []
  vi.stubGlobal('fetch', async (_input: unknown, init?: RequestInit) => {
    sent.push(init?.body as Uint8Array)
    return new Response(new Uint8Array(answer), {
      headers: { 'content-type': 'application/proto' },
    })
  })
  const api = createNoteApi(createConnectTransport({ baseUrl: 'http://127.0.0.1:1' }))
  return { api, sent }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('reading a note', () => {
  it('gives the prose and the file it came out of', async () => {
    const { api } = answerWith([...text(1, 'One line.\n'), ...stamp(3)])

    expect(await api.readNote('Kept.md')).toStrictEqual({
      ok: true,
      value: { prose: 'One line.\n', stamp: { path: 'Kept.md', size: 10n, mtime: 7n } },
    })
  })

  it('says why the core did not read it, in words', async () => {
    const { api } = answerWith(count(2, NOTE_ERROR_CODE.NOT_FOUND))

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
    const { api, sent } = answerWith([])

    const answer = await api.writeNote('Kept.md', 'Two lines.\n', {
      prose: 'One line.\n',
      stamp: { path: 'Kept.md', size: 10n, mtime: 7n },
    })

    expect(answer).toStrictEqual({ ok: true, value: null })
    expect([...sent[0]!]).toStrictEqual([
      ...text(1, 'Kept.md'),
      ...text(2, 'Two lines.\n'),
      ...message(3, text(1, 'One line.\n'), stamp(2)),
    ])
  })

  it('sends no version for a note that was read without one', async () => {
    const { api, sent } = answerWith([])

    await api.writeNote('Kept.md', 'Two lines.\n', null)

    expect([...sent[0]!]).toStrictEqual([...text(1, 'Kept.md'), ...text(2, 'Two lines.\n')])
  })

  it('says why the core did not write it, in words', async () => {
    const { api } = answerWith(count(1, NOTE_ERROR_CODE.CONFLICT))

    expect(await api.writeNote('Kept.md', 'Two lines.\n', null)).toStrictEqual({
      ok: false,
      error: formatErrorCodeMessage(NOTE_ERROR_CODE.CONFLICT),
    })
  })
})
