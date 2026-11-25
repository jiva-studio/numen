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

interface FileCall {
  readonly url: string
  readonly authorization: string | null
  readonly body: unknown
}

function answerFileService(status: number, answer: object) {
  const calls: FileCall[] = []
  vi.stubGlobal('fetch', async (input: unknown, init?: RequestInit) => {
    const request = input instanceof Request ? input : new Request(String(input), init)
    calls.push({
      url: request.url,
      authorization: request.headers.get('Authorization'),
      body: await request.json(),
    })
    return new Response(JSON.stringify(answer), {
      status,
      headers: { 'content-type': 'application/json' },
    })
  })
  return { api: createFileApi(createBearerTransport('http://127.0.0.1:1', 't0ken')), calls }
}

describe('downloading a file and removing its download', () => {
  it('asks the core to download the file', async () => {
    const { api, calls } = answerFileService(200, {})

    const answer = await api.fetchFile('Books/Guide.pdf')

    expect(answer).toStrictEqual({ ok: true, value: null })
    expect(calls).toStrictEqual([
      {
        url: 'http://127.0.0.1:1/numen.v1.FileService/FetchFile',
        authorization: 'Bearer t0ken',
        body: { path: 'Books/Guide.pdf' },
      },
    ])
  })

  it('asks the core to remove the download of the file', async () => {
    const { api, calls } = answerFileService(200, {})

    const answer = await api.dropDownload('Books/Guide.pdf')

    expect(answer).toStrictEqual({ ok: true, value: null })
    expect(calls.map((call) => call.url)).toStrictEqual([
      'http://127.0.0.1:1/numen.v1.FileService/DropDownload',
    ])
    expect(calls[0]?.body).toStrictEqual({ path: 'Books/Guide.pdf' })
  })

  it('returns the message of the core when it refuses', async () => {
    const { api } = answerFileService(400, {
      code: 'failed_precondition',
      message: 'The file has changes that are not sent',
    })

    const answer = await api.dropDownload('Books/Guide.pdf')

    expect(answer).toStrictEqual({ ok: false, error: 'The file has changes that are not sent' })
  })

  it('returns the message of the core when no plugin can fill the file', async () => {
    const { api } = answerFileService(400, {
      code: 'failed_precondition',
      message: 'the plugin for files.provider is not installed',
    })

    const answer = await api.fetchFile('Books/Guide.pdf')

    expect(answer).toStrictEqual({
      ok: false,
      error: 'the plugin for files.provider is not installed',
    })
  })
})
