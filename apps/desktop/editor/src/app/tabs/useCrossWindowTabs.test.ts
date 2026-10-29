import { describe, expect, it, vi } from 'vitest'
import { effectScope, ref } from 'vue'
import { useCrossWindowTabs } from './useCrossWindowTabs'

describe('useCrossWindowTabs', () => {
  it('handles incoming dock request and replies with tab-docked confirmation', async () => {
    const channelName = 'test-channel-' + crypto.randomUUID()
    const senderChannel = new BroadcastChannel(channelName)

    const mockWindowTabs = {
      openTabOfKind: vi.fn(async () => 'tab-new'),
      releaseTab: vi.fn(() => true),
      tabs: ref([{ id: 'tab-1', title: 'Plex' }]),
    }

    const replies: unknown[] = []
    senderChannel.onmessage = (event) => {
      replies.push(event.data)
    }

    const scope = effectScope()
    scope.run(() => {
      useCrossWindowTabs({ windowTabs: mockWindowTabs, windowId: 'win-target' }, channelName)
    })

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
    hook.dockToOtherWindow('tab-1', 'plex', 'win-target')
    await new Promise((resolve) => setTimeout(resolve, 50))

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

    expect(mockWindowTabs.releaseTab).toHaveBeenCalledWith('tab-1')
    expect(closeWindow).toHaveBeenCalled()

    scope.stop()
    peerChannel.close()
  })

  it('does not send tab-docked if openTabOfKind returns undefined or empty', async () => {
    const channelName = 'test-channel-' + crypto.randomUUID()
    const senderChannel = new BroadcastChannel(channelName)

    const mockWindowTabs = {
      openTabOfKind: vi.fn(async () => undefined),
      releaseTab: vi.fn(() => true),
      tabs: ref([{ id: 'tab-1', title: 'Plex' }]),
    }

    const replies: unknown[] = []
    senderChannel.onmessage = (event) => {
      replies.push(event.data)
    }

    const scope = effectScope()
    scope.run(() => {
      useCrossWindowTabs({ windowTabs: mockWindowTabs, windowId: 'win-target' }, channelName)
    })

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
    const channelName = 'test-channel-' + crypto.randomUUID()
    const senderChannel = new BroadcastChannel(channelName)

    const mockWindowTabs = {
      openTabOfKind: vi.fn(async () => 'tab-new'),
      releaseTab: vi.fn(() => true),
      tabs: ref([]),
    }

    const scope = effectScope()
    scope.run(() => {
      useCrossWindowTabs({ windowTabs: mockWindowTabs, windowId: 'win-target' }, channelName)
    })

    senderChannel.postMessage(null)
    senderChannel.postMessage({ type: 'unknown-type' })
    senderChannel.postMessage({ type: 'tab-dock', tabId: 't1' })
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
