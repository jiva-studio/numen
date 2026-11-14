/**
 * The settings tab is given the connections of the core: it reads them when a page asks, and words the
 * reason the core gives for a refused change. Every value here is invented.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSettingsStore, SETTINGS_WORDS } from '@/entities/settings'
import type { WindowTabs } from '@/entities/tab'
import type { ConnectionInfo } from '@/entities/connection'
import type { MessageLog } from '@/shared/model/notices/messages'
import { WORDS } from '@/shared/ui/words'
import { createAnsweredApi } from '@/testing/coreAnswers'
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
  isDetected: false,
  path: '',
  uses: [],
}

/** The calls of a window whose core answers none; a test answers the connections it reads. */
const api = createAnsweredApi({})

const createSettings = () => {
  const log = { getWriter: () => vi.fn() } as unknown as MessageLog
  const windowTabs = { groups: { handle: { openTab: vi.fn(() => Promise.resolve('')) } } }
  return createWindowSettingsPart({
    api,
    rest: createSettingsStore(api.settings, SETTINGS_WORDS, vi.fn()),
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
    vi.spyOn(api.connection, 'listConnectionTypes').mockResolvedValue({
      types: [],
      operationTitles: {},
    })
    const list = vi.spyOn(api.connection, 'listConnections').mockResolvedValue([connection])
    const { connections: installed } = createSettings().configured.state.installation

    await installed.load()

    expect(installed.list.value).toStrictEqual([connection])
    expect(list).toHaveBeenCalledOnce()
  })

  it('give the reason in words when the core refuses a create', async () => {
    vi.spyOn(api.connection, 'createConnection').mockResolvedValue({
      ok: false,
      error: { errorCode: 'name_taken', detail: 'Work' },
    })
    const { connections: installed } = createSettings().configured.state.installation

    const answer = await installed.edits.create({
      typeId: 'a.b/x',
      name: 'Work',
      fields: {},
      secrets: {},
    })

    expect(answer).toStrictEqual({ ok: false, error: 'Another connection has this name.' })
  })
})
