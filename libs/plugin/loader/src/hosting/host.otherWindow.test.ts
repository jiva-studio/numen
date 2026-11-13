/**
 * A window other than the editor over the same host: the target it names, the slot entries it reads
 * from the manifests of the plugins it activated, and the tabs and readers it does not offer.
 */
import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import type { Manifest } from '@numen/core-client'
import { createPluginHost } from './host'
import type { PluginHostDeps } from './types'

const createDeps = (extra: Partial<PluginHostDeps> = {}): PluginHostDeps => ({
  target: 'desktop.flashcards',
  registerKind: vi.fn(() => vi.fn()),
  ...extra,
})

const LEDGER = createMockManifest('org.example.ledger', { name: 'Ledger', version: '1' })
const TIDE = createMockManifest('org.example.tide', { name: 'Tide', version: '1' })

describe('a window that names its target', () => {
  it('activates its plugins in that target', () => {
    const cards = createPluginHost(createDeps({ target: 'desktop.flashcards' }))
    const editor = createPluginHost(createDeps({ target: 'desktop.editor' }))

    expect(cards.lifecycle.createContext('org.example.ledger', LEDGER).plugin.target).toBe(
      'desktop.flashcards',
    )
    expect(editor.lifecycle.createContext('org.example.ledger', LEDGER).plugin.target).toBe(
      'desktop.editor',
    )
  })
})

describe('the slot entries of a window', () => {
  it('are read from the manifests of the plugins activated, until each is disposed', () => {
    const read: string[][] = []
    const getSlotEntries = (manifests: ReadonlyMap<string, Manifest>) => {
      read.push([...manifests.keys()])
      return []
    }
    const host = createPluginHost(createDeps({ getSlotEntries }))

    host.lifecycle.createContext('org.example.ledger', LEDGER)
    host.lifecycle.createContext('org.example.tide', TIDE)
    host.lifecycle.createContext('org.example.bare')
    host.slots.getEntries('titleBar')
    host.lifecycle.disposePlugin('org.example.ledger')
    host.slots.getEntries('titleBar')

    expect(read.slice(-2)).toStrictEqual([
      ['org.example.ledger', 'org.example.tide'],
      ['org.example.tide'],
    ])
  })
})

describe('a window that opens no tab for a plugin', () => {
  it('takes a reader as a tab kind and registers no extension for it', () => {
    const registerKind = vi.fn(() => vi.fn())
    const host = createPluginHost(
      createDeps({ registerKind, getReaderExtensions: () => ['pdf', 'epub'] }),
    )
    const context = host.lifecycle.createContext(
      'org.example.ledger',
      createMockManifest('org.example.ledger', { name: 'Ledger', readers: ['document'] }),
    )

    context.register.reader?.({ id: 'document', title: 'Document', component: {} } as never)

    expect(registerKind).toHaveBeenCalledOnce()
  })
})
