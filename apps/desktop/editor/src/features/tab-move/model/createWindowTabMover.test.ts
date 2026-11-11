import { describe, expect, it, vi } from 'vitest'
import { answerCore } from '@/testing/coreAnswers'
import { createTestTabs } from '@/testing/tabs'
import { createWindowTabMover } from './createWindowTabMover'

const setup = (asking = false) => {
  const requests: string[] = []
  const gates: (() => void)[] = []
  answerCore({
    workspace: {
      highlightWindow: ({ windowId, isOn }) => {
        requests.push(`${windowId}:${isOn ? 'on' : 'off'}`)
        return new Promise<Record<string, never>>((resolve) => gates.push(() => resolve({})))
      },
    },
  })
  const mover = createWindowTabMover({
    windowId: 'window-a',
    windowTabs: createTestTabs(),
    notes: { asking: () => asking, flushPending: () => Promise.resolve() },
    closeTab: () => true,
    writeMessage: vi.fn(),
  })
  return { mover, requests, gates }
}

describe('the outline a window offers over another', () => {
  it('sends the requests one at a time, so an off cannot pass the on it ends', async () => {
    const { mover, requests, gates } = setup()

    mover.highlightWindow('window-b', true)
    mover.highlightWindow('window-b', false)
    await vi.waitFor(() => expect(requests).toStrictEqual(['window-b:on']))
    gates[0]?.()

    await vi.waitFor(() => expect(requests).toStrictEqual(['window-b:on', 'window-b:off']))
  })

  it('goes on after a request that failed', async () => {
    const { mover, requests } = setup()
    answerCore({
      workspace: {
        highlightWindow: ({ isOn }) => {
          requests.push(isOn ? 'on' : 'off')
          if (isOn) throw new Error('window gone')
          return {}
        },
      },
    })

    mover.highlightWindow('window-b', true)
    mover.highlightWindow('window-b', false)

    await vi.waitFor(() => expect(requests).toStrictEqual(['on', 'off']))
  })
})

describe('the tabs of a window', () => {
  it('say whether a tab waits for an answer, as the notes do', () => {
    expect(setup(true).mover.isTabAsking('plex:1')).toBe(true)
    expect(setup(false).mover.isTabAsking('plex:1')).toBe(false)
  })
})
