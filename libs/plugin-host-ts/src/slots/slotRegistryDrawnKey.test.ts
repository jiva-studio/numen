/** The layout key the move menu uses is the key of the entry the slot draws. Every plugin here is invented. */
import { describe, expect, it } from 'vitest'
import { createSlotRegistry, type DeclaredSlotEntry } from './slotRegistry'

const LEDGER = 'org.example.ledger'

describe('the layout key of a command placed twice in one slot', () => {
  const entries: DeclaredSlotEntry[] = [
    { pluginId: LEDGER, slot: 'titleBar.commands', command: 'ledger.open', order: 5 },
    { pluginId: LEDGER, slot: 'titleBar.commands', command: `${LEDGER}:ledger.open`, order: 1 },
  ]
  const registry = createSlotRegistry({
    getEntries: () => entries,
    findCommand: () => ({ title: 'Open ledger', shortcut: '' }),
    findView: () => undefined,
  })

  it('is the key of the drawn entry, not of the first declared one', () => {
    const itemKey = `${LEDGER}/command/${LEDGER}:ledger.open`

    expect(registry.findLayoutKey(itemKey, 'titleBar.commands')).toBe(
      `titleBar.commands|${LEDGER}/${LEDGER}:ledger.open`,
    )
    expect(registry.listLayoutEntries('titleBar.commands').map((one) => one.key)).toStrictEqual([
      `titleBar.commands|${LEDGER}/${LEDGER}:ledger.open`,
    ])
  })
})
