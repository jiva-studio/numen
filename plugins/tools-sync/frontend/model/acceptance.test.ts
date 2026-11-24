import { afterEach, describe, expect, it, vi } from 'vitest'
import type { AcceptInvitationClient, RequestStatus } from '../api/invitations'
import { createAcceptanceView } from './acceptance'

const PREVIEW = {
  folder: 'Food',
  ownerFingerprint: 'AAAA BBBB',
  expiresAt: '2026-11-25T12:00:00Z',
  isReadOnly: true,
}
const status = (patch: Partial<RequestStatus>): RequestStatus => ({
  phase: 'connecting',
  folder: 'Food',
  ownerFingerprint: 'AAAA BBBB',
  expiresAt: '2026-11-25T12:00:00Z',
  ...patch,
})

const createClient = (statuses: RequestStatus[]): AcceptInvitationClient => ({
  readSharePreview: vi.fn(() => Promise.resolve(PREVIEW)),
  acceptShareInvitation: vi.fn(() =>
    Promise.resolve({ requestId: 'r1', folder: 'Food', ownerFingerprint: 'AAAA BBBB' }),
  ),
  getShareInvitationStatus: vi.fn(() =>
    Promise.resolve(statuses.length > 1 ? (statuses.shift() as RequestStatus) : statuses[0]!),
  ),
})

const create = (client: AcceptInvitationClient) =>
  createAcceptanceView({ client, close: vi.fn(), refreshMs: 1000 })

afterEach(() => vi.useRealTimers())

describe('the dialog that accepts a shared folder', () => {
  it('shows what the pasted link offers, and trims the text', async () => {
    const client = createClient([])
    const view = create(client)

    await view.setLink('  numen-share:abc ')

    expect(client.readSharePreview).toHaveBeenCalledWith('numen-share:abc')
    expect(view.preview).toStrictEqual(PREVIEW)
  })

  it('shows the words of a link the backend refuses', async () => {
    const client = createClient([])
    vi.mocked(client.readSharePreview).mockRejectedValue(new Error('The link expired'))
    const view = create(client)

    await view.setLink('numen-share:abc')

    expect([view.preview, view.previewError]).toStrictEqual([undefined, 'The link expired'])
  })

  it('drops the answer of a link that was replaced before the answer came', async () => {
    const answers = new Map<string, (value: typeof PREVIEW) => void>()
    const client = createClient([])
    vi.mocked(client.readSharePreview).mockImplementation(
      (link) => new Promise((resolve) => answers.set(link, resolve)),
    )
    const view = create(client)

    const first = view.setLink('first')
    const second = view.setLink('second')
    answers.get('second')?.({ ...PREVIEW, folder: 'Second' })
    answers.get('first')?.({ ...PREVIEW, folder: 'First' })
    await Promise.all([first, second])

    expect(view.preview?.folder).toBe('Second')
  })

  it('shows the code while the owner decides, and the folder once it is allowed', async () => {
    vi.useFakeTimers()
    const client = createClient([
      status({ phase: 'waiting', code: '482913' }),
      status({ phase: 'received', code: '482913' }),
    ])
    const view = create(client)
    await view.setLink('numen-share:abc')

    await view.accept()
    expect([view.phase, view.code]).toStrictEqual(['waiting', '482913'])
    await vi.advanceTimersByTimeAsync(1000)

    expect(client.acceptShareInvitation).toHaveBeenCalledWith({
      link: 'numen-share:abc',
      deviceName: '',
    })
    expect(view.phase).toBe('received')
    const asked = vi.mocked(client.getShareInvitationStatus).mock.calls.length
    await vi.advanceTimersByTimeAsync(5000)
    expect(vi.mocked(client.getShareInvitationStatus).mock.calls.length).toBe(asked)
  })

  it.each(['declined', 'used', 'expired'] as const)('ends as %s', async (phase) => {
    const view = create(createClient([status({ phase })]))
    await view.setLink('numen-share:abc')

    await view.accept()

    expect(view.phase).toBe(phase)
  })

  it('ends as failed with the reason the backend gives', async () => {
    const view = create(createClient([status({ phase: 'failed', error: 'The owner is offline' })]))
    await view.setLink('numen-share:abc')

    await view.accept()

    expect([view.phase, view.reason]).toStrictEqual(['failed', 'The owner is offline'])
  })

  it('goes back to the link with the words of a request the backend refuses', async () => {
    const client = createClient([])
    vi.mocked(client.acceptShareInvitation).mockRejectedValue(new Error('No identity'))
    const view = create(client)
    await view.setLink('numen-share:abc')

    await view.accept()

    expect([view.phase, view.error]).toStrictEqual(['link', 'No identity'])
  })
})
