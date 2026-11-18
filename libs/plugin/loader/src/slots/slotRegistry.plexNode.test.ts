/** The views plugins place over and under a node of the plex graph, as the plex reads them. Every plugin here is invented. */
import { describe, expect, it, vi } from 'vitest'
import { createSlotRegistry, type DeclaredSlotEntry } from './slotRegistry'

const LEDGER = 'org.example.ledger'
const HARBOR = 'org.example.harbor'

const setup = (entries: readonly DeclaredSlotEntry[]) => {
  const reportIssue = vi.fn()
  const registry = createSlotRegistry({
    getEntries: () => entries,
    findCommand: () => undefined,
    findView: (pluginId, view) => ({ name: `${pluginId}-${view}` }),
    reportIssue,
  })
  return { registry, reportIssue }
}

const readViews = (items: readonly { kind: string; view?: string }[]) =>
  items.map((one) => (one.kind === 'view' ? one.view : one.kind))

describe('the slots over and under a node of the plex graph', () => {
  const entries: readonly DeclaredSlotEntry[] = [
    { pluginId: LEDGER, slot: 'plexGraph.nodeAbove', view: 'parents', order: 2 },
    { pluginId: HARBOR, slot: 'plexGraph.nodeAbove', view: 'tags', order: 1 },
    { pluginId: LEDGER, slot: 'plexGraph.nodeBelow', view: 'related' },
  ]

  it('hand the owner the views of the plugins, in order', () => {
    const { registry } = setup(entries)

    expect(readViews(registry.getEntries('plexGraph.nodeAbove'))).toStrictEqual(['tags', 'parents'])
    expect(readViews(registry.getEntries('plexGraph.nodeBelow'))).toStrictEqual(['related'])
  })

  it('are known to the window, so a view placed in them raises no issue', () => {
    const { registry, reportIssue } = setup(entries)

    registry.getEntries('plexGraph.nodeAbove')

    expect(reportIssue).not.toHaveBeenCalled()
  })
})
