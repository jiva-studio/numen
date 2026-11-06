/** Which manifests contribute to the interface of a window. Every manifest here is invented. */
import { describe, expect, it } from 'vitest'
import { parseManifest } from '@numen/core-client'
import { isInterfaceManifest } from './module'

const read = (contributes: unknown) => {
  const parsed = parseManifest(
    JSON.stringify({ id: 'org.example.ledger', name: 'Ledger', version: '1.0.0', contributes }),
  )
  if (!parsed.ok) throw new Error(parsed.error)
  return parsed.value
}

describe('isInterfaceManifest', () => {
  it('counts a slot view, tab kinds, readers, menu entries and themes', () => {
    for (const contributes of [
      { views: [{ id: 'bar', slot: 'titleBar' }] },
      { tabKinds: [{ kind: 'ledger', title: 'Ledger' }] },
      { readers: [{ id: 'ledger', contentTypes: ['text/csv'] }] },
      { menus: [{ location: 'titleBar', command: 'ledger.open' }] },
      { themes: [{ id: 'harbor', name: 'Harbor', file: 'themes/harbor.css' }] },
    ]) {
      expect(isInterfaceManifest(read(contributes))).toBe(true)
    }
  })

  it('does not count a plugin that only declares commands or status views', () => {
    expect(isInterfaceManifest(read({ commands: [{ id: 'ledger.sum', title: 'Sum' }] }))).toBe(
      false,
    )
    expect(isInterfaceManifest(read({ statusViews: [{ id: 'sum' }] }))).toBe(false)
    expect(isInterfaceManifest(read(undefined))).toBe(false)
  })
})
