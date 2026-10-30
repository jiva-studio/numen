// @vitest-environment jsdom
/** The summary a window mounts beside each vault, drawn from the one count of the review. */
import { flushPromises, mount } from '@vue/test-utils'
import type { Component } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { VAULT_SUMMARY } from '@numen/plugin-sdk'

const wire = vi.hoisted(() => ({ countsOpened: 0 }))

vi.mock('./shared/clients', () => ({
  cards: {
    watchCardsDue: async function* () {
      wire.countsOpened += 1
      const vault = {
        id: 'roots',
        name: 'Roots',
        path: '/vaults/Roots',
        reviewCards: 4,
        due: 3,
        new: 4,
        decks: [],
        presets: [],
        unread: '',
        isIndexing: false,
      }
      yield { day: '2026-04-02', vaults: [vault] }
      yield { day: '', vaults: [], counted: vault }
    },
    watchReloads: async function* () {
      await new Promise<never>(() => undefined)
    },
  },
}))

const { createVaultCounts, createVaultSummaryView } = await import('./index')

const disposers: (() => void)[] = []

afterEach(() => {
  for (const dispose of disposers.splice(0)) dispose()
  wire.countsOpened = 0
})

const mountSummary = (counts: ReturnType<typeof createVaultCounts>, vault: string) => {
  const view = createVaultSummaryView(counts)
  const summary = mount(view.component as Component, { props: { context: { vault } } })
  disposers.push(() => summary.unmount())
  return { view, summary }
}

describe('the summary of a vault', () => {
  it('is declared as the vault summary, and starts the count the review shares', async () => {
    const counts = createVaultCounts({ reportError: vi.fn() })
    disposers.push(counts.dispose)

    const { view, summary } = mountSummary(counts, 'roots')
    mountSummary(counts, 'other')
    await flushPromises()

    expect(view.capability).toBe(VAULT_SUMMARY)
    expect(wire.countsOpened).toBe(1)
    expect(summary.text()).toContain('7')
  })
})
