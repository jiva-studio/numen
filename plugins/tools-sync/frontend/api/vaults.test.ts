/**
 * The vaults service as the settings view calls it: the fixtures of testdata/api are the JSON the Go
 * messages encode to, read here as the TypeScript messages, and a failed call throws the message of the
 * backend.
 */
import { describe, expect, it } from 'vitest'
import { createApi } from '@numen/plugin-sdk'
import { createVaultsClient, SERVICE } from './vaults'
type VaultsClient = ReturnType<typeof createVaultsClient>

type ListVaultsResponse = Awaited<ReturnType<VaultsClient['listVaults']>>
type ListRemoteVaultsResponse = Awaited<ReturnType<VaultsClient['listRemoteVaults']>>
type ConnectVaultRequest = Parameters<VaultsClient['connectVault']>[0]
type GetStatusResponse = Awaited<ReturnType<VaultsClient['getStatus']>>
type GetConnectPreviewRequest = Parameters<VaultsClient['getConnectPreview']>[0]
type GetConnectPreviewResponse = Awaited<ReturnType<VaultsClient['getConnectPreview']>>

const fixtures = import.meta.glob('../../testdata/api/*.json', {
  eager: true,
  import: 'default',
}) as Record<string, unknown>

const read = (name: string): unknown => fixtures[`../../testdata/api/${name}.json`]

const ORIGIN = 'http://window'

interface Sent {
  readonly url: string
  readonly body: unknown
}

const createClient = (respond: () => Response) => {
  const sent: Sent[] = []
  const client = createVaultsClient(
    createApi(SERVICE, {
      origin: ORIGIN,
      fetch: async (url, init) => {
        sent.push({ url: String(url), body: JSON.parse(String(init?.body)) })
        return respond()
      },
    }),
  )
  return { client, sent }
}

describe('the fixtures of the vaults service', () => {
  it('reads the ListVaults fixture as ListVaultsResponse', () => {
    const response = read('ListVaults.response') as ListVaultsResponse

    expect(read('ListVaults.request')).toStrictEqual({})
    expect(response.vaults.map((one) => one.state)).toStrictEqual(['connected', 'notConnected'])
    expect(response.vaults.at(0)).toStrictEqual({
      id: '01JCX6Q4ZB3W8N2V7T5R9K1M0P',
      name: 'Test',
      path: '/home/person/Test',
      state: 'connected',
      lastSyncedAt: '2026-11-23T12:00:00Z',
      conflictCount: 1,
      pausedDeletions: { count: 5, tracked: 10, direction: 'send' },
    })
  })

  it('reads the ListRemoteVaults fixture as ListRemoteVaultsResponse', () => {
    const response = read('ListRemoteVaults.response') as ListRemoteVaultsResponse

    expect(read('ListRemoteVaults.request')).toStrictEqual({})
    expect(response.vaults.map((one) => one.name)).toStrictEqual(['Test', 'Work'])
    expect(response.vaults.map((one) => one.isConnected)).toStrictEqual([true, false])
  })

  it('reads the ConnectVault fixture as ConnectVaultRequest', () => {
    const request: ConnectVaultRequest = {
      vaultId: '01JCX6Q4ZB3W8N2V7T5R9K1M0Q',
      mode: 'existing',
      remoteId: '00112233445566778899aabbccddeeff',
    }

    expect(read('ConnectVault.request')).toStrictEqual(request)
    expect(read('ConnectVault.response')).toStrictEqual({})
  })

  it('reads the GetConnectPreview fixture as GetConnectPreviewRequest and GetConnectPreviewResponse', () => {
    const request: GetConnectPreviewRequest = {
      vaultId: '01JCX6Q4ZB3W8N2V7T5R9K1M0Q',
      remoteId: '00112233445566778899aabbccddeeff',
    }
    const response: GetConnectPreviewResponse = { differCount: 2 }

    expect(read('GetConnectPreview.request')).toStrictEqual(request)
    expect(read('GetConnectPreview.response')).toStrictEqual(response)
  })

  it('reads the ResolveDeletions fixture as ResolveDeletionsRequest', () => {
    expect(read('ResolveDeletions.request')).toStrictEqual({
      vaultId: '01JCX6Q4ZB3W8N2V7T5R9K1M0P',
      choice: 'keep',
    })
    expect(read('ResolveDeletions.response')).toStrictEqual({})
  })

  it('reads the GetStatus fixture as GetStatusResponse', () => {
    const response: GetStatusResponse = {
      hasIdentity: true,
      isReachable: true,
      items: [{ key: 'conflicts', text: '1 note changed on two computers' }],
    }

    expect(read('GetStatus.request')).toStrictEqual({})
    expect(read('GetStatus.response')).toStrictEqual(response)
  })
})

describe('the vaults client', () => {
  it('asks ListVaults and answers its response', async () => {
    const answer = read('ListVaults.response') as ListVaultsResponse
    const { client, sent } = createClient(() => Response.json(answer))

    const response = await client.listVaults()

    expect(sent).toStrictEqual([{ url: `${ORIGIN}/${SERVICE}/ListVaults`, body: {} }])
    expect(response).toStrictEqual(answer)
  })

  it('asks ListRemoteVaults and answers its response', async () => {
    const answer = read('ListRemoteVaults.response') as ListRemoteVaultsResponse
    const { client, sent } = createClient(() => Response.json(answer))

    const response = await client.listRemoteVaults()

    expect(sent).toStrictEqual([{ url: `${ORIGIN}/${SERVICE}/ListRemoteVaults`, body: {} }])
    expect(response).toStrictEqual(answer)
  })

  it('sends the vault, the mode and the remote vault of ConnectVault', async () => {
    const request = read('ConnectVault.request') as ConnectVaultRequest
    const { client, sent } = createClient(() => Response.json({}))

    await client.connectVault(request)

    expect(sent).toStrictEqual([{ url: `${ORIGIN}/${SERVICE}/ConnectVault`, body: request }])
  })

  it('asks GetConnectPreview and answers its response', async () => {
    const request = read('GetConnectPreview.request') as GetConnectPreviewRequest
    const answer = read('GetConnectPreview.response') as GetConnectPreviewResponse
    const { client, sent } = createClient(() => Response.json(answer))

    const response = await client.getConnectPreview(request)

    expect(sent).toStrictEqual([{ url: `${ORIGIN}/${SERVICE}/GetConnectPreview`, body: request }])
    expect(response).toStrictEqual(answer)
  })

  it('asks GetStatus and answers its response', async () => {
    const answer = read('GetStatus.response') as GetStatusResponse
    const { client, sent } = createClient(() => Response.json(answer))

    const response = await client.getStatus()

    expect(sent).toStrictEqual([{ url: `${ORIGIN}/${SERVICE}/GetStatus`, body: {} }])
    expect(response).toStrictEqual(answer)
  })

  it('throws the message of a failed call', async () => {
    const { client } = createClient(() =>
      Response.json(
        { code: 'failed_precondition', message: 'a copy of a connected vault' },
        { status: 412 },
      ),
    )

    const answer = client.connectVault({ vaultId: 'Copy', mode: 'new' })

    await expect(answer).rejects.toThrow('a copy of a connected vault')
  })
})
