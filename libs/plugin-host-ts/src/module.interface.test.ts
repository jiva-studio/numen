/** Which manifests fill a slot of the window, the plugins a window waits for. Every manifest here is invented. */
import { describe, expect, it } from 'vitest'
import { parseManifest } from '@numen/core-client'
import { isWindowSlotManifest } from './module'

const read = (contributes: unknown) => {
  const parsed = parseManifest(
    JSON.stringify({ id: 'org.example.ledger', name: 'Ledger', version: '1.0.0', contributes }),
  )
  if (!parsed.ok) throw new Error(parsed.error)
  return parsed.value
}

describe('isWindowSlotManifest', () => {
  it('counts a plugin that places an entry in the title bar or the status bar of the window', () => {
    for (const slot of ['titleBar', 'statusBar']) {
      expect(isWindowSlotManifest(read({ slots: [{ slot, view: 'bar' }] }))).toBe(true)
    }
  })

  it('does not count a plugin that fills a slot inside a bar, or a menu, or declares tabs', () => {
    expect(
      isWindowSlotManifest(
        read({
          slots: [
            { slot: 'titleBar.commands', command: 'ledger.open' },
            { slot: 'titleBar.status', view: 'balance' },
            { slot: 'fileTree.context', command: 'ledger.open' },
          ],
          tabKinds: [{ kind: 'ledger', title: 'Ledger' }],
          themes: [{ id: 'harbor', name: 'Harbor', file: 'themes/harbor.css' }],
        }),
      ),
    ).toBe(false)
    expect(isWindowSlotManifest(read(undefined))).toBe(false)
  })
})
