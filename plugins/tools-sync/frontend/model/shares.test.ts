import { describe, expect, it, vi } from 'vitest'
import type { Share, SharesClient } from '../api/shares'
import { createReceivedView, createShareView, isIdentityWarning } from './shares'

const TARGET = { vaultId: 'v1', folder: 'Food' }

const createClient = (patch: Partial<SharesClient> = {}): SharesClient => ({
  getReaderCode: vi.fn(() => Promise.resolve({ code: 'numen-reader-1.abc', fingerprint: 'AAAA BBBB' })),
  readReaderCode: vi.fn(() => Promise.resolve({ fingerprint: 'CCCC DDDD' })),
  getShare: vi.fn((): Promise<Share> => Promise.resolve({ state: 'notShared', readers: [] })),
  addReader: vi.fn(() => Promise.resolve({ invitation: 'numen-invitation-1.xyz', fingerprint: 'CCCC DDDD' })),
  removeReader: vi.fn(() => Promise.resolve()),
  stopShare: vi.fn(() => Promise.resolve()),
  readInvitation: vi.fn(() =>
    Promise.resolve({
      shareName: 'Food',
      ownerFingerprint: 'EEEE FFFF',
      isForThisAccount: true,
      alreadyReceived: false,
    }),
  ),
  acceptInvitation: vi.fn(() => Promise.resolve()),
  listReceived: vi.fn(() => Promise.resolve([])),
  ...patch,
})

const createShare = (client: SharesClient) => createShareView({ client, target: TARGET, close: vi.fn() })

describe('the dialog of a shared folder', () => {
  it('shows the fingerprint of the reader code that was typed', async () => {
    const view = createShare(createClient())

    await view.setCode('  numen-reader-1.def ')

    expect(view.fingerprint).toBe('CCCC DDDD')
  })

  it('drops the answer of a code that was replaced before the answer came', async () => {
    const answers = new Map<string, (value: { fingerprint: string }) => void>()
    const client = createClient({
      readReaderCode: (code) => new Promise((resolve) => answers.set(code, resolve)),
    })
    const view = createShare(client)

    const first = view.setCode('first')
    const second = view.setCode('second')
    answers.get('second')?.({ fingerprint: 'SECOND' })
    answers.get('first')?.({ fingerprint: 'FIRST' })
    await Promise.all([first, second])

    expect(view.fingerprint).toBe('SECOND')
  })

  it('gives the invitation and clears the form once the reader is added', async () => {
    const client = createClient()
    const view = createShare(client)
    await view.setCode('numen-reader-1.def')
    view.setLabel('Anna')

    await view.add(false)

    expect(client.addReader).toHaveBeenCalledWith({
      ...TARGET,
      code: 'numen-reader-1.def',
      label: 'Anna',
      confirmChange: false,
    })
    expect(view.invitation).toBe('numen-invitation-1.xyz')
    expect([view.code, view.label, view.error]).toStrictEqual(['', '', ''])
  })

  it('keeps the form and shows the words of a refusal', async () => {
    const client = createClient({
      addReader: () => Promise.reject(new Error('A shared vault cannot be shared again')),
    })
    const view = createShare(client)
    await view.setCode('numen-reader-1.def')

    await view.add(false)

    expect(view.error).toBe('A shared vault cannot be shared again')
    expect(view.code).toBe('numen-reader-1.def')
    expect(view.invitation).toBe('')
  })

  it('knows the warning that needs the person to confirm', () => {
    expect(isIdentityWarning('This person has another identity than before')).toBe(true)
    expect(isIdentityWarning('That is your own identity')).toBe(false)
  })

  it('asks the share again while it is being made, and stops when it is active', async () => {
    vi.useFakeTimers()
    const states: Share[] = [
      { state: 'creating', readers: [] },
      { state: 'active', readers: [{ label: 'Anna', fingerprint: 'CCCC DDDD', state: 'active' }] },
    ]
    const client = createClient({ getShare: vi.fn(() => Promise.resolve(states.shift() ?? states[0] as Share)) })
    const view = createShareView({ client, target: TARGET, close: vi.fn(), refreshMs: 1000 })

    await view.load()
    expect(view.share.state).toBe('creating')
    await vi.advanceTimersByTimeAsync(1000)

    expect(view.share.state).toBe('active')
    expect(client.getShare).toHaveBeenCalledTimes(2)
    view.stop()
    vi.useRealTimers()
  })

  it('removes a reader and reads the share again', async () => {
    const client = createClient()
    const view = createShare(client)

    await view.removeReader({ label: 'Anna', fingerprint: 'CCCC DDDD', state: 'active' })

    expect(client.removeReader).toHaveBeenCalledWith({ ...TARGET, fingerprint: 'CCCC DDDD' })
    expect(client.getShare).toHaveBeenCalledTimes(1)
  })

  it('forgets the invitation when the sharing stops', async () => {
    const view = createShare(createClient())
    await view.setCode('numen-reader-1.def')
    view.setLabel('Anna')
    await view.add(false)

    await view.stopShare()

    expect(view.invitation).toBe('')
  })
})

describe('the page of the shares received', () => {
  it('shows the code of this account and the shares received', async () => {
    const client = createClient({
      listReceived: () =>
        Promise.resolve([
          { vaultId: 'r1', name: 'Food', ownerFingerprint: 'EEEE FFFF', state: 'ready', lastSyncedAt: '' },
        ]),
    })
    const view = createReceivedView({ client })

    await view.load()

    expect(view.code).toBe('numen-reader-1.abc')
    expect(view.rows).toStrictEqual([
      { vaultId: 'r1', name: 'Food', ownerFingerprint: 'EEEE FFFF', stateLabel: 'Up to date' },
    ])
  })

  it('shows who owns the invitation that was pasted', async () => {
    const view = createReceivedView({ client: createClient() })

    await view.setText('numen-invitation-1.xyz')

    expect([view.shareName, view.ownerFingerprint]).toStrictEqual(['Food', 'EEEE FFFF'])
  })

  it('reports why an invitation for another reader is refused', async () => {
    const client = createClient({
      readInvitation: () => Promise.reject(new Error('This invitation is for another person')),
    })
    const view = createReceivedView({ client })

    await view.setText('numen-invitation-1.xyz')

    expect(view.invitationError).toBe('This invitation is for another person')
    expect(view.shareName).toBe('')
  })

  it('receives the invitation and empties the field', async () => {
    const client = createClient()
    const view = createReceivedView({ client })
    await view.setText(' numen-invitation-1.xyz ')

    await view.receive()

    expect(client.acceptInvitation).toHaveBeenCalledWith('numen-invitation-1.xyz')
    expect(view.text).toBe('')
  })

  it('keeps the invitation and shows the words of a refusal to receive it', async () => {
    const client = createClient({
      acceptInvitation: () => Promise.reject(new Error('The owner identity changed')),
    })
    const view = createReceivedView({ client })
    await view.setText('numen-invitation-1.xyz')

    await view.receive()

    expect(view.error).toBe('The owner identity changed')
    expect(view.text).toBe('numen-invitation-1.xyz')
  })

  it('words the three states of a received share', async () => {
    const base = { ownerFingerprint: 'E', lastSyncedAt: '' }
    const client = createClient({
      listReceived: () =>
        Promise.resolve([
          { ...base, vaultId: 'r1', name: 'One', state: 'receiving' },
          { ...base, vaultId: 'r2', name: 'Two', state: 'ready' },
          { ...base, vaultId: 'r3', name: 'Three', state: 'ended' },
        ]),
    })
    const view = createReceivedView({ client, refreshMs: 1_000_000 })

    await view.load()
    view.stop()

    expect(view.rows.map((row) => row.stateLabel)).toStrictEqual([
      'Receiving…',
      'Up to date',
      'The owner ended this share',
    ])
  })
})
