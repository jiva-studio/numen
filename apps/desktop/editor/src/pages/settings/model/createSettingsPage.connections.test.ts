/**
 * The settings tab is given the connections of the core, and reads them again after a change of the
 * settings only once a page has read them. Every value here is invented.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { connections } from '@/shared/api/clients'
import { createSettingsStore, SETTINGS_WORDS } from '@/entities/settings'
import type { WindowTabs } from '@/entities/tab'
import type { ConnectionInfo } from '@/entities/connection'
import type { MessageLog } from '@/shared/model/notices/messages'
import { WORDS } from '@/shared/ui/words'
import { createWindowSettingsPart } from '@/testing/windowSettings'

const connection: ConnectionInfo = {
  id: 'c00000001',
  typeId: 'a.b/x',
  name: 'Work',
  revision: 1,
  fields: {},
  secretKeysSet: [],
  isTypeAvailable: true,
  isComplete: true,
  uses: [],
}

const createSettings = () => {
  const log = { getWriter: () => vi.fn() } as unknown as MessageLog
  const windowTabs = { groups: { handle: { openTab: vi.fn(() => Promise.resolve('')) } } }
  return createWindowSettingsPart({
    rest: createSettingsStore(SETTINGS_WORDS, vi.fn()),
    words: WORDS,
    log,
    windowTabs: windowTabs as unknown as WindowTabs,
    onSizeChanged: vi.fn(),
    getSettingsViews: () => new Map(),
    getPlatform: () => 'linux',
  })
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('the connections in the settings tab', () => {
  it('are read when the page asks, and listed in the installation', async () => {
    vi.spyOn(connections, 'listTypes').mockResolvedValue([])
    const list = vi.spyOn(connections, 'list').mockResolvedValue([connection])
    const { connections: installed } = createSettings().configured.state.installation

    await installed.load()

    expect(installed.list.value).toStrictEqual([connection])
    expect(list).toHaveBeenCalledOnce()
  })

  it('give the reason in words when the core refuses a create', async () => {
    vi.spyOn(connections, 'create').mockResolvedValue({
      ok: false,
      error: { errorCode: 'name_taken', detail: 'Work' },
    })
    const { connections: installed } = createSettings().configured.state.installation

    const answer = await installed.create({
      typeId: 'a.b/x',
      name: 'Work',
      fields: {},
      secrets: {},
    })

    expect(answer).toStrictEqual({ ok: false, error: 'Another connection has this name.' })
  })
})
