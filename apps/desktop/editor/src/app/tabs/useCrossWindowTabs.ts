/**
 * Cross-window tab transfer: coordinates tab docking across windows via BroadcastChannel.
 */
import { onScopeDispose } from 'vue'

export interface CrossWindowTabsDeps {
  readonly windowTabs: {
    readonly openTabOfKind: (kind: string) => Promise<string | undefined>
    readonly releaseTab: (tabId: string) => boolean
    readonly tabs: { readonly value: readonly unknown[] }
  }
  readonly closeWindowIfEmpty?: (() => void) | undefined
  readonly windowId?: string | undefined
}

export interface CrossWindowMessage {
  readonly type: 'tab-dock' | 'tab-docked'
  readonly tabId: string
  readonly fromWindowId: string
  readonly targetWindowId: string
  readonly kind?: string
}

function isStringField(val: unknown): val is string {
  return typeof val === 'string' && val.length > 0
}

function isCrossWindowMessage(data: unknown): data is CrossWindowMessage {
  if (!data || typeof data !== 'object') return false
  const m = data as Record<string, unknown>
  if (
    !isStringField(m.tabId) ||
    !isStringField(m.fromWindowId) ||
    !isStringField(m.targetWindowId)
  ) {
    return false
  }
  if (m.type === 'tab-dock') return isStringField(m.kind)
  return m.type === 'tab-docked'
}

function createBroadcastChannel(channelName: string): BroadcastChannel | null {
  try {
    return typeof BroadcastChannel !== 'undefined' ? new BroadcastChannel(channelName) : null
  } catch {
    return null
  }
}

interface ChannelListenerDeps {
  readonly channel: BroadcastChannel | null
  readonly windowId: string
  readonly onDock: (msg: CrossWindowMessage) => Promise<void>
  readonly onDocked: (msg: CrossWindowMessage) => void
}

function setupChannelListener(deps: ChannelListenerDeps) {
  const { channel, windowId, onDock, onDocked } = deps
  if (!channel) return
  channel.onmessage = async (event: MessageEvent<unknown>) => {
    if (!isCrossWindowMessage(event.data)) return
    const msg = event.data
    if (msg.fromWindowId === windowId) return
    try {
      if (msg.type === 'tab-dock') await onDock(msg)
      else if (msg.type === 'tab-docked') onDocked(msg)
    } catch {
      // ignore message handling error
    }
  }
}

export function useCrossWindowTabs(
  deps: CrossWindowTabsDeps,
  channelName = 'numen-cross-window-tabs',
) {
  const windowId = deps.windowId ?? crypto.randomUUID()
  const channel = createBroadcastChannel(channelName)
  const pendingDocks = new Map<string, string>()

  const handleTabDock = async (msg: CrossWindowMessage): Promise<void> => {
    if (msg.targetWindowId !== windowId || !msg.kind) return
    const openedTab = await deps.windowTabs.openTabOfKind(msg.kind)
    if (!openedTab) return
    channel?.postMessage({
      type: 'tab-docked',
      tabId: msg.tabId,
      fromWindowId: windowId,
      targetWindowId: msg.fromWindowId,
    })
  }

  const isMatchingPendingDock = (msg: CrossWindowMessage): boolean => {
    const expected = pendingDocks.get(msg.tabId)
    return Boolean(expected && expected === msg.fromWindowId)
  }

  const handleTabDocked = (msg: CrossWindowMessage): void => {
    if (msg.targetWindowId !== windowId || !isMatchingPendingDock(msg)) return
    pendingDocks.delete(msg.tabId)
    deps.windowTabs.releaseTab(msg.tabId)
    if (deps.windowTabs.tabs.value.length === 0) {
      deps.closeWindowIfEmpty?.()
    }
  }

  setupChannelListener({
    channel,
    windowId,
    onDock: handleTabDock,
    onDocked: handleTabDocked,
  })

  const dockToOtherWindow = (tabId: string, kind: string, targetWindowId: string): void => {
    if (!targetWindowId) return
    pendingDocks.set(tabId, targetWindowId)
    channel?.postMessage({
      type: 'tab-dock',
      tabId,
      kind,
      fromWindowId: windowId,
      targetWindowId,
    })
  }

  onScopeDispose(() => {
    channel?.close()
  })

  return { dockToOtherWindow, windowId }
}
