/**
 * The invitation methods as the dialogs call them: the name of the method, the request it sends, and the
 * message of the backend a failed call throws.
 */
import { describe, expect, it } from 'vitest'
import { createApi } from '@numen/plugin-sdk'
import { createAcceptInvitationClient, createShareInvitationClient } from './invitations'
import { SERVICE } from './vaults'

interface Sent {
  readonly url: string
  readonly body: unknown
}

const createClients = (respond: () => Response) => {
  const sent: Sent[] = []
  const api = createApi(SERVICE, {
    origin: 'http://window',
    fetch: async (url, init) => {
      sent.push({ url: String(url), body: JSON.parse(String(init?.body)) })
      return respond()
    },
  })
  return { owner: createShareInvitationClient(api), reader: createAcceptInvitationClient(api), sent }
}

describe('the client of the owner of a folder', () => {
  it('starts a link for a folder of a vault', async () => {
    const { owner, sent } = createClients(() =>
      Response.json({ id: 'i1', link: 'numen-share:abc', expiresAt: '2026-11-25T12:00:00Z' }),
    )

    const started = await owner.startShareInvitation({ vaultId: 'v1', folder: 'Food' })

    expect(started.link).toBe('numen-share:abc')
    expect(sent.at(0)?.url).toMatch(/StartShareInvitation$/)
    expect(sent.at(0)?.body).toStrictEqual({ vaultId: 'v1', folder: 'Food' })
  })

  it('lists the requests that wait', async () => {
    const request = {
      requestId: 'r1',
      vaultId: 'v1',
      folder: 'Food',
      deviceName: 'Phone',
      fingerprint: 'AAAA BBBB',
      code: '482913',
    }
    const { owner } = createClients(() => Response.json({ requests: [request] }))

    expect(await owner.listShareRequests()).toStrictEqual([request])
  })

  it('allows, declines and revokes by the id', async () => {
    const { owner, sent } = createClients(() => Response.json({}))

    await owner.approveShareRequest('r1')
    await owner.declineShareRequest('r2')
    await owner.revokeShareInvitation('i3')

    expect(sent.map((one) => [one.url.split('/').at(-1), one.body])).toStrictEqual([
      ['ApproveShareRequest', { requestId: 'r1' }],
      ['DeclineShareRequest', { requestId: 'r2' }],
      ['RevokeShareInvitation', { id: 'i3' }],
    ])
  })

  it('throws the message of a refusal', async () => {
    const { owner } = createClients(() =>
      Response.json({ code: 'failed_precondition', message: 'The vault is not synced' }, { status: 400 }),
    )

    await expect(owner.listShareInvitations()).rejects.toThrow('The vault is not synced')
  })
})

describe('the client of the person who got a link', () => {
  it('previews the link, sends the request and reads how it stands', async () => {
    const { reader, sent } = createClients(() =>
      Response.json({ requestId: 'r1', folder: 'Food', ownerFingerprint: 'AAAA' }),
    )

    await reader.previewShareInvitation('numen-share:abc')
    await reader.acceptShareInvitation({ link: 'numen-share:abc', deviceName: '' })
    await reader.getShareInvitationStatus('r1')

    expect(sent.map((one) => [one.url.split('/').at(-1), one.body])).toStrictEqual([
      ['PreviewShareInvitation', { link: 'numen-share:abc' }],
      ['AcceptShareInvitation', { link: 'numen-share:abc', deviceName: '' }],
      ['GetShareInvitationStatus', { requestId: 'r1' }],
    ])
  })
})
