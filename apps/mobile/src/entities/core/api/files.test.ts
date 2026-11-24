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

interface SyncCall {
  readonly url: string
  readonly authorization: string | null
  readonly vault: string | null
  readonly body: unknown
}

function answerSync(status: number, answer: object) {
  const calls: SyncCall[] = []
  vi.stubGlobal('fetch', async (input: unknown, init?: RequestInit) => {
    const url = String(input)
    if (url.includes('VaultService')) {
      return new Response(JSON.stringify({ id: 'v-test' }), {
        headers: { 'content-type': 'application/json' },
      })
    }
    const headers = new Headers(init?.headers)
    calls.push({
      url,
      authorization: headers.get('Authorization'),
      vault: headers.get('Numen-Vault'),
      body: JSON.parse(String(init?.body)),
    })
    return new Response(JSON.stringify(answer), { status })
  })
  const transport = createBearerTransport('http://127.0.0.1:1', 't0ken')
  return { api: createFileApi(transport, { baseUrl: 'http://127.0.0.1:1', token: 't0ken' }), calls }
}

describe('downloading a file and removing its download', () => {
  it('asks the sync plugin to download the file of the vault', async () => {
    const { api, calls } = answerSync(200, {})

    const answer = await api.fetchFile('Books/Guide.pdf')

    expect(answer).toStrictEqual({ ok: true, value: null })
    expect(calls).toStrictEqual([
      {
        url: 'http://127.0.0.1:1/md.numen.tools.sync/FetchFile',
        authorization: 'Bearer t0ken',
        vault: 'v-test',
        body: { vaultId: 'v-test', path: 'Books/Guide.pdf' },
      },
    ])
  })

  it('asks the sync plugin to remove the download of the file', async () => {
    const { api, calls } = answerSync(200, {})

    const answer = await api.dropDownload('Books/Guide.pdf')

    expect(answer).toStrictEqual({ ok: true, value: null })
    expect(calls.map((call) => call.url)).toStrictEqual([
      'http://127.0.0.1:1/md.numen.tools.sync/DropDownload',
    ])
  })

  it('returns the message of the plugin when it refuses', async () => {
    const { api } = answerSync(412, { message: 'The file has changes that are not sent' })

    const answer = await api.dropDownload('Books/Guide.pdf')

    expect(answer).toStrictEqual({ ok: false, error: 'The file has changes that are not sent' })
  })

  it('names the status when the plugin gives no message', async () => {
    const { api } = answerSync(503, {})

    const answer = await api.fetchFile('Books/Guide.pdf')

    expect(answer).toStrictEqual({ ok: false, error: 'The core answered 503' })
  })

  it('says the sync is not on the phone when no endpoint is given', async () => {
    const api = createFileApi(createBearerTransport('http://127.0.0.1:1', 't0ken'))

    const answer = await api.fetchFile('Books/Guide.pdf')

    expect(answer).toStrictEqual({ ok: false, error: 'The sync is not on this phone' })
  })
})
