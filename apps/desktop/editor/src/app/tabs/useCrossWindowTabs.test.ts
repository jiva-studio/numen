import { describe, expect, it, vi } from 'vitest'
import { effectScope, ref } from 'vue'
import { useCrossWindowTabs } from './useCrossWindowTabs'

/** The tabs of a window, opening a tab of a kind with `openTabOfKind`. */
interface WindowTabsStub {
  readonly openTabOfKind: () => Promise<string | undefined>
  readonly tabs: readonly { readonly id: string; readonly title: string }[]
}

/**
 * A window `win-target` listening for dock requests on a channel of its own, and a sender on the
 * same channel that records the replies.
 */
const listenAsTarget = (
  openTabOfKind: WindowTabsStub['openTabOfKind'],
  tabs: WindowTabsStub['tabs'],
) => {
  const channelName = 'test-channel-' + crypto.randomUUID()
  const senderChannel = new BroadcastChannel(channelName)
  const mockWindowTabs = {
    openTabOfKind: vi.fn(openTabOfKind),
    releaseTab: vi.fn(() => true),
    tabs: ref(tabs),
  }
  const replies: unknown[] = []
  senderChannel.onmessage = (event) => {
    replies.push(event.data)
  }
  const scope = effectScope()
  scope.run(() => {
    useCrossWindowTabs({ windowTabs: mockWindowTabs, windowId: 'win-target' }, channelName)
  })
  return { senderChannel, mockWindowTabs, replies, scope }
}

describe('useCrossWindowTabs', () => {
  it('handles incoming dock request and replies with tab-docked confirmation', async () => {
    const { senderChannel, mockWindowTabs, replies, scope } = listenAsTarget(
      async () => 'tab-new',
      [{ id: 'tab-1', title: 'Plex' }],
    )

    senderChannel.postMessage({
      type: 'tab-dock',
      tabId: 'tab-1',
      kind: 'plex',
      fromWindowId: 'win-source',
      targetWindowId: 'win-target',
    })

    await new Promise((resolve) => setTimeout(resolve, 50))

    expect(mockWindowTabs.openTabOfKind).toHaveBeenCalledWith('plex')
    expect(replies).toContainEqual(
      expect.objectContaining({
        type: 'tab-docked',
        tabId: 'tab-1',
        fromWindowId: 'win-target',
        targetWindowId: 'win-source',
      }),
    )

    scope.stop()
    senderChannel.close()
  })

  it('releases tab only when matching pending dock is acknowledged and closes empty window', async () => {
    const channelName = 'test-channel-' + crypto.randomUUID()
    const peerChannel = new BroadcastChannel(channelName)
    const closeWindow = vi.fn()

    const mockWindowTabs = {
      openTabOfKind: vi.fn(async () => 'tab-new'),
      releaseTab: vi.fn((_id: string) => {
        mockWindowTabs.tabs.value = []
        return true
      }),
      tabs: ref([{ id: 'tab-1', title: 'Plex' }]),
    }

    const scope = effectScope()
    let hook!: ReturnType<typeof useCrossWindowTabs>
    scope.run(() => {
      hook = useCrossWindowTabs(
        { windowTabs: mockWindowTabs, windowId: 'win-source', closeWindowIfEmpty: closeWindow },
        channelName,
      )
    })

    // Unsolicited acknowledgement should NOT release tab
    peerChannel.postMessage({
      type: 'tab-docked',
      tabId: 'tab-1',
      fromWindowId: 'win-target',
      targetWindowId: 'win-source',
    })
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(mockWindowTabs.releaseTab).not.toHaveBeenCalled()

    // Initiate dock
    hook.dockToOtherWindow('tab-1', 'plex', '')
    hook.dockToOtherWindow('tab-1', 'plex', 'win-target')
    await new Promise((resolve) => setTimeout(resolve, 50))

    // Primitive or unknown-type messages are ignored
    peerChannel.postMessage('not-an-object')
    peerChannel.postMessage({
      type: 'unknown-type',
      tabId: 'tab-1',
      fromWindowId: 'win-target',
      targetWindowId: 'win-source',
    })
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(mockWindowTabs.releaseTab).not.toHaveBeenCalled()

    // Acknowledgement targeted at another window should NOT release tab
    peerChannel.postMessage({
      type: 'tab-docked',
      tabId: 'tab-1',
      fromWindowId: 'win-target',
      targetWindowId: 'win-other',
    })
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(mockWindowTabs.releaseTab).not.toHaveBeenCalled()

    // Acknowledgement from unexpected window should NOT release tab
    peerChannel.postMessage({
      type: 'tab-docked',
      tabId: 'tab-1',
      fromWindowId: 'win-other',
      targetWindowId: 'win-source',
    })
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(mockWindowTabs.releaseTab).not.toHaveBeenCalled()

    // Matching acknowledgement from target window releases tab and triggers close
    peerChannel.postMessage({
      type: 'tab-docked',
      tabId: 'tab-1',
      fromWindowId: 'win-target',
      targetWindowId: 'win-source',
    })
    await new Promise((resolve) => setTimeout(resolve, 50))

    expect(mockWindowTabs.releaseTab).toHaveBeenCalledTimes(1)
    expect(mockWindowTabs.releaseTab).toHaveBeenCalledWith('tab-1')
    expect(closeWindow).toHaveBeenCalledTimes(1)

    // Duplicate acknowledgement after pending dock is cleared does not release again
    peerChannel.postMessage({
      type: 'tab-docked',
      tabId: 'tab-1',
      fromWindowId: 'win-target',
      targetWindowId: 'win-source',
    })
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(mockWindowTabs.releaseTab).toHaveBeenCalledTimes(1)

    scope.stop()
    peerChannel.close()
  })

  it('sends tab-dock message and does not close window when other tabs remain', async () => {
    const channelName = 'test-channel-' + crypto.randomUUID()
    const peerChannel = new BroadcastChannel(channelName)
    const received: unknown[] = []
    peerChannel.onmessage = (event) => received.push(event.data)
    const closeWindow = vi.fn()

    const mockWindowTabs = {
      openTabOfKind: vi.fn(async () => 'tab-new'),
      releaseTab: vi.fn(() => true),
      tabs: ref([{ id: 'tab-2', title: 'Remaining' }]),
    }

    const scope = effectScope()
    let hook!: ReturnType<typeof useCrossWindowTabs>
    scope.run(() => {
      hook = useCrossWindowTabs(
        { windowTabs: mockWindowTabs, windowId: 'win-source', closeWindowIfEmpty: closeWindow },
        channelName,
      )
    })

    hook.dockToOtherWindow('tab-1', 'plex', '')
    await new Promise((resolve) => setTimeout(resolve, 30))
    expect(received).toHaveLength(0)

    hook.dockToOtherWindow('tab-1', 'plex', 'win-target')
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(received).toEqual([
      {
        type: 'tab-dock',
        tabId: 'tab-1',
        kind: 'plex',
        fromWindowId: 'win-source',
        targetWindowId: 'win-target',
      },
    ])

    peerChannel.postMessage({
      type: 'tab-docked',
      tabId: 'tab-1',
      fromWindowId: 'win-target',
      targetWindowId: 'win-source',
    })
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(mockWindowTabs.releaseTab).toHaveBeenCalledWith('tab-1')
    expect(closeWindow).not.toHaveBeenCalled()

    const closeSpy = vi.spyOn(BroadcastChannel.prototype, 'close')
    scope.stop()
    expect(closeSpy).toHaveBeenCalled()
    closeSpy.mockRestore()
    peerChannel.close()
  })

  it('operates safely when BroadcastChannel is unavailable and closeWindowIfEmpty is omitted', () => {
    const original = globalThis.BroadcastChannel
    ;(globalThis as { BroadcastChannel: unknown }).BroadcastChannel = function () {
      throw new Error('unavailable')
    }
    try {
      const scope = effectScope()
      scope.run(() => {
        const hook = useCrossWindowTabs({
          windowTabs: {
            openTabOfKind: vi.fn(async () => 't'),
            releaseTab: vi.fn(() => true),
            tabs: ref([]),
          },
        })
        expect(hook.windowId.length).toBeGreaterThan(0)
        hook.dockToOtherWindow('tab-1', 'plex', 'win-target')
      })
      scope.stop()
    } finally {
      globalThis.BroadcastChannel = original
    }
  })

  it('does not send tab-docked if openTabOfKind returns undefined or empty', async () => {
    const { senderChannel, mockWindowTabs, replies, scope } = listenAsTarget(
      async () => undefined,
      [{ id: 'tab-1', title: 'Plex' }],
    )

    senderChannel.postMessage({
      type: 'tab-dock',
      tabId: 'tab-1',
      kind: 'unknown-kind',
      fromWindowId: 'win-source',
      targetWindowId: 'win-target',
    })
    await new Promise((resolve) => setTimeout(resolve, 50))

    expect(mockWindowTabs.openTabOfKind).toHaveBeenCalledWith('unknown-kind')
    expect(replies).toHaveLength(0)

    scope.stop()
    senderChannel.close()
  })

  it('ignores malformed or untargeted messages', async () => {
    const { senderChannel, mockWindowTabs, scope } = listenAsTarget(async () => 'tab-new', [])

    senderChannel.postMessage(null)
    senderChannel.postMessage('not-an-object')
    senderChannel.postMessage({ type: 'unknown-type' })
    senderChannel.postMessage({
      type: 'unknown-type',
      tabId: 't1',
      fromWindowId: 'win-source',
      targetWindowId: 'win-target',
    })
    senderChannel.postMessage({
      type: 'tab-dock',
      tabId: '',
      kind: 'plex',
      fromWindowId: 'win-source',
      targetWindowId: 'win-target',
    })
    senderChannel.postMessage({
      type: 'tab-dock',
      tabId: 123,
      kind: 'plex',
      fromWindowId: 'win-source',
      targetWindowId: 'win-target',
    })
    senderChannel.postMessage({
      type: 'tab-dock',
      tabId: 't1',
      kind: 'plex',
      fromWindowId: '',
      targetWindowId: 'win-target',
    })
    senderChannel.postMessage({
      type: 'tab-dock',
      tabId: 't1',
      kind: 'plex',
      fromWindowId: 'win-source',
      targetWindowId: '',
    })
    senderChannel.postMessage({
      type: 'tab-dock',
      tabId: 't1',
      kind: '',
      fromWindowId: 'win-source',
      targetWindowId: 'win-target',
    })
    senderChannel.postMessage({
      type: 'tab-dock',
      tabId: 't1',
      kind: 'plex',
      fromWindowId: 'win-target',
      targetWindowId: 'win-target',
    })
    senderChannel.postMessage({
      type: 'tab-dock',
      tabId: 't1',
      kind: 'plex',
      fromWindowId: 'win-source',
      targetWindowId: 'win-other',
    })
    await new Promise((resolve) => setTimeout(resolve, 50))

    expect(mockWindowTabs.openTabOfKind).not.toHaveBeenCalled()

    scope.stop()
    senderChannel.close()
  })
})
