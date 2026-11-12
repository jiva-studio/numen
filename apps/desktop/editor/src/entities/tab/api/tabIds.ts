/** The identities a window gives its tabs and panes. */
export interface TabIds {
  readonly createTabId: (kind: string) => string
  readonly createPaneId: () => string
}

/**
 * The identity given to a tab or conversation when none can be found for it: the
 * kind's name plus a value unique to this window and to any window opened after it.
 * The agent keeps a conversation under each name it receives, and a name maps to one
 * of them.
 */
export const browserTabIds: TabIds = {
  createTabId: (kind) => `${kind}:${crypto.randomUUID()}`,
  createPaneId: () => crypto.randomUUID(),
}
