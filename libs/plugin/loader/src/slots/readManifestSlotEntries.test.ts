/** The slot entries the registry reads from a manifest. Every manifest here is invented. */
import { describe, expect, it } from 'vitest'
import { parseManifest } from '@numen/core-client'
import { readManifestSlotEntries } from './slotRegistry'

const getIcon = (name: string) => ({ iconName: name })

const read = (slots: unknown) => {
  const parsed = parseManifest(
    JSON.stringify({
      id: 'org.example.ledger',
      name: 'Ledger',
      version: '1.0.0',
      contributes: { commands: [{ id: 'ledger.open', title: 'Open ledger' }], slots },
    }),
  )
  if (!parsed.ok) throw new Error(parsed.error)
  return parsed.value
}

describe('readManifestSlotEntries', () => {
  it('reads a command, a view and a submenu with every field the manifest gives, and the icon of the submenu by its name', () => {
    const manifest = read([
      { slot: 'titleBar.commands', command: 'ledger.open', group: 'navigate', order: 40 },
      { slot: 'titleBar.status', view: 'balance' },
      {
        slot: 'fileTree.context',
        submenu: 'ledger.new',
        title: 'New ledger',
        icon: 'plus',
        group: '0-new',
        when: 'isFolder && !isMultiple',
      },
    ])

    expect(readManifestSlotEntries('org.example.ledger', manifest, getIcon)).toStrictEqual([
      {
        pluginId: 'org.example.ledger',
        slot: 'titleBar.commands',
        command: 'ledger.open',
        group: 'navigate',
        order: 40,
      },
      { pluginId: 'org.example.ledger', slot: 'titleBar.status', view: 'balance' },
      {
        pluginId: 'org.example.ledger',
        slot: 'fileTree.context',
        submenu: 'ledger.new',
        title: 'New ledger',
        icon: { iconName: 'plus' },
        group: '0-new',
        when: 'isFolder && !isMultiple',
      },
    ])
  })

  it('keeps an order of zero', () => {
    const manifest = read([{ slot: 'titleBar.commands', command: 'ledger.open', order: 0 }])

    expect(readManifestSlotEntries('org.example.ledger', manifest, getIcon)[0]?.order).toBe(0)
  })

  it('gives no entries for a manifest without slots', () => {
    expect(readManifestSlotEntries('org.example.ledger', read(undefined), getIcon)).toStrictEqual(
      [],
    )
  })
})
