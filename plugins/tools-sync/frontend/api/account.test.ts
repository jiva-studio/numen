/**
 * The account methods of the service as the settings view calls them: the fixtures of testdata/api are the JSON the
 * Go messages encode to, and a failed call throws an error with the code of the backend.
 */
import { describe, expect, it } from 'vitest'
import { createApi } from '@numen/plugin-sdk'
import { SERVICE } from './vaults'
import { createAccountClient, type AccountClient } from './account'

const fixtures = import.meta.glob('../../testdata/api/*.json', {
  eager: true,
  import: 'default',
}) as Record<string, unknown>

const read = (name: string): unknown => fixtures[`../../testdata/api/${name}.json`]

const createClient = (respond: () => Response) => {
  const sent: { url: string; body: unknown }[] = []
  const client = createAccountClient(
    createApi(SERVICE, {
      origin: 'http://window',
      fetch: async (url, init) => {
        sent.push({ url: String(url), body: JSON.parse(String(init?.body)) })
        return respond()
      },
    }),
  )
  return { client, sent }
}

const answer = (body: unknown): Response => Response.json(body)

describe('the fixtures of the identity methods', () => {
  it('reads the GetAccountState fixture as an account state', () => {
    const response: Awaited<ReturnType<AccountClient['getAccountState']>> = {
      hasIdentity: true,
      account: '0123456789abcdef0123456789abcdef',
      isDeleting: true,
      deleteAt: '2026-11-30T12:00:00Z',
    }

    expect(read('GetAccountState.request')).toStrictEqual({})
    expect(read('GetAccountState.response')).toStrictEqual(response)
  })

  it('reads the fixtures of the deletion, the restore and the export', () => {
    expect(read('DeleteCloudData.response')).toStrictEqual({ deleteAt: '2026-11-30T12:00:00Z' })
    expect(read('RestoreAccount.response')).toStrictEqual({})
    expect(read('ExportAccountData.response')).toStrictEqual({
      document: '{"account":"0123456789abcdef0123456789abcdef","vaults":[]}',
    })
  })
})

describe('createAccountClient', () => {
  it('answers a device with no identity when the host does not start the backend', async () => {
    const { client } = createClient(() =>
      Response.json({ code: 'requirement_unmet', message: 'needs identity' }, { status: 503 }),
    )

    expect((await client.getAccountState()).hasIdentity).toBe(false)
  })

  it('sends the method of the service and returns the time the server deletes the account', async () => {
    const { client, sent } = createClient(() => answer(read('DeleteCloudData.response')))

    const deleteAt = await client.deleteCloudData()

    expect(deleteAt).toBe('2026-11-30T12:00:00Z')
    expect(sent).toStrictEqual([{ url: `http://window/${SERVICE}/DeleteCloudData`, body: {} }])
  })

  it('returns the document of the export as the text the server sent', async () => {
    const { client } = createClient(() => answer(read('ExportAccountData.response')))

    expect(await client.exportAccountData()).toBe(
      '{"account":"0123456789abcdef0123456789abcdef","vaults":[]}',
    )
  })

  it('throws an error with the code of a failed call', async () => {
    const { client } = createClient(
      () =>
        new Response(JSON.stringify({ code: 'unavailable', message: 'down' }), {
          status: 503,
          headers: { 'Content-Type': 'application/json' },
        }),
    )

    await expect(client.restoreAccount()).rejects.toMatchObject({ code: 'unavailable' })
  })
})
