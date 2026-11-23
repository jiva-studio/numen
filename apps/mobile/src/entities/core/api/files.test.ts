/**
 * What the file calls send to the core and what they make of its answers. The core's side of the
 * wire is a stand-in for `fetch`, which answers in the protocol's JSON encoding.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createBearerTransport } from '@numen/core-client'
import { createFileApi } from './files'

function answerWith(answer: object) {
  const requests: unknown[] = []
  vi.stubGlobal('fetch', async (_input: unknown, init?: RequestInit) => {
    requests.push(JSON.parse(new TextDecoder().decode(init?.body as Uint8Array)))
    return new Response(JSON.stringify(answer), {
      headers: { 'content-type': 'application/json' },
    })
  })
  return { api: createFileApi(createBearerTransport('http://127.0.0.1:1', 't0ken')), requests }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('listing the files of a folder', () => {
  it('asks for the folder and returns each entry with whether its bytes are on the phone', async () => {
    const { api, requests } = answerWith({
      entries: [
        { path: 'Books', name: 'Books', isFolder: true },
        { path: 'Shopping list.md', name: 'Shopping list.md', isNote: true },
        { path: 'Guide.pdf', name: 'Guide.pdf', isAbsent: true, provider: 'tools-sync' },
      ],
    })

    const rows = await api.listFiles('')

    expect(requests).toStrictEqual([{}])
    expect(rows).toStrictEqual([
      { path: 'Books', name: 'Books', isFolder: true, isAbsent: false },
      { path: 'Shopping list.md', name: 'Shopping list.md', isFolder: false, isAbsent: false },
      { path: 'Guide.pdf', name: 'Guide.pdf', isFolder: false, isAbsent: true },
    ])
  })

  it('lists a folder by its path', async () => {
    const { api, requests } = answerWith({ entries: [] })

    await api.listFiles('Books')

    expect(requests).toStrictEqual([{ path: 'Books' }])
  })
})
