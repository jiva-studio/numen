/**
 * The settings tab is given the connections of the core when the window has a client for them, and
 * reads them again after a change of the settings only once a page has read them. Every value here
 * is invented.
 */
import { describe, expect, it, vi } from 'vitest'
import { asValue, type ConnectionClient, type ConnectionInfo } from '@numen/core-client'
import { useSettings } from './useSettings'
import { WORDS } from '@/shared/words'
import type { SettingsPort } from '@/app/ports/settings'
import type { MessageLog } from '@/shared/notices/messages'
import type { useWindowTabs } from '@/entities/tab'

const connection: ConnectionInfo = {
  id: 'c00000001',
  typeId: 'a.b/x',
  name: 'Work',
  revision: 1,
  fields: {},
  secretKeysSet: [],
  isTypeAvailable: true,
  isComplete: true,
  isPrivateNetworkAllowed: false,
  uses: [],
}

const createClient = (): ConnectionClient => ({
  listTypes: vi.fn(() => Promise.resolve([])),
  list: vi.fn(() => Promise.resolve([connection])),
  create: vi.fn(() => Promise.resolve(asValue(connection))),
  update: vi.fn(() => Promise.resolve(asValue(connection))),
  delete: vi.fn(() => Promise.resolve(asValue(undefined))),
})

const createSettings = (connections?: ConnectionClient) => {
  const log = { getWriter: () => vi.fn() } as unknown as MessageLog
  const windowTabs = {
    handle: { openTab: vi.fn(() => Promise.resolve('')) },
  } as unknown as ReturnType<typeof useWindowTabs>
  return useSettings({
    core: {} as SettingsPort,
    words: WORDS,
    log,
    windowTabs,
    onSizeChanged: vi.fn(),
    ...(connections ? { connections } : {}),
  })
}

describe('the connections in the settings tab', () => {
  it('are absent while the window has no client for them', () => {
    const installation = createSettings().configured.state.installation

    expect(installation.connections).toBeUndefined()
    expect(installation.createConnection).toBeUndefined()
  })

  it('are read when the page asks, and listed in the installation', async () => {
    const client = createClient()
    const installation = createSettings(client).configured.state.installation

    await installation.loadConnections?.()

    expect(installation.connections?.value).toStrictEqual([connection])
    expect(client.list).toHaveBeenCalledOnce()
  })

  it('give the reason in words when the core refuses a create', async () => {
    const client = createClient()
    vi.mocked(client.create).mockResolvedValue({
      ok: false,
      error: { errorCode: 'name_taken', detail: 'Work' },
    })
    const installation = createSettings(client).configured.state.installation

    const answer = await installation.createConnection?.({
      typeId: 'a.b/x',
      name: 'Work',
      fields: {},
      secrets: {},
      isPrivateNetworkAllowed: false,
    })

    expect(answer).toStrictEqual({ ok: false, error: 'Another connection has this name.' })
  })
})
